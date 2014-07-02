package luna

import (
	"fmt"
	"io"
	"log"
	"reflect"
	"runtime"

	"github.com/beatgammit/golua/lua"
)

type Lib uint

const (
	LibBase Lib = 1 << iota
	LibIO
	LibMath
	LibPackage
	LibString
	LibTable
	LibOS
)

const (
	NoLibs Lib = 0

	// AllLibs represents all available Lua standard libraries
	AllLibs = LibBase | LibIO | LibMath | LibPackage | LibString | LibTable | LibOS
)

type TableKeyValue struct {
	Key string
	Val interface{}
}

type Luna struct {
	L   *lua.State
	lib Lib
	q   chan<- packet
}

type response struct {
	ret interface{}
	err error
}

type packet struct {
	fn func() (interface{}, error)
	c  chan response
}

// New creates a new Luna instance, opening all libs provided.
func New(libs Lib) *Luna {
	// TODO: make queue size user-settable
	c := make(chan packet, 5)

	l := &Luna{lua.NewState(), libs, c}
	if libs == AllLibs {
		l.L.OpenLibs()
	} else {
		if libs&LibBase != 0 {
			l.L.OpenBase()
		}
		if libs&LibIO != 0 {
			l.L.OpenIO()
		}
		if libs&LibMath != 0 {
			l.L.OpenMath()
		}
		if libs&LibPackage != 0 {
			l.L.OpenPackage()
		}
		if libs&LibString != 0 {
			l.L.OpenString()
		}
		if libs&LibTable != 0 {
			l.L.OpenTable()
		}
		if libs&LibOS != 0 {
			l.L.OpenOS()
		}
	}

	go l.worker(c)

	return l
}

func (l *Luna) worker(c <-chan packet) {
	runtime.LockOSThread()
	for p := range c {
		ret, err := p.fn()
		p.c <- response{ret, err}
	}
}

func (l *Luna) stdCall(fn func() (LuaRet, error)) (LuaRet, error) {
	c := make(chan response, 1)
	l.q <- packet{
		func() (interface{}, error) {
			return fn()
		},
		c,
	}

	resp := <-c
	ret, _ := resp.ret.(LuaRet)
	return ret, resp.err
}

func (l *Luna) errCall(fn func() error) error {
	c := make(chan response, 1)
	l.q <- packet{
		func() (interface{}, error) {
			return nil, fn()
		},
		c,
	}

	resp := <-c
	return resp.err
}

// Stdout changes where print() writes to (default os.Stdout).
// Note, this does **not** change anything in the io package.
func (l *Luna) Stdout(w io.Writer) {
	c := make(chan response, 1)
	l.q <- packet{
		func() (interface{}, error) {
			l.L.Register("print", wrapperGen(l, reflect.ValueOf(printGen(w))))
			return nil, nil
		},
		c,
	}
	<-c
}

func (l *Luna) loadFile(path string) (LuaRet, error) {
	err := l.L.DoFile(path)
	if err != nil {
		return nil, err
	}
	return l.getReturnValues(), nil
}

func (l *Luna) load(src string) (LuaRet, error) {
	err := l.L.DoString(src)
	if err != nil {
		return nil, err
	}
	return l.getReturnValues(), nil
}

// loads and executes a Lua source file
func (l *Luna) LoadFile(path string) (LuaRet, error) {
	return l.stdCall(func() (LuaRet, error) { return l.loadFile(path) })
}

// loads and executes Lua source
func (l *Luna) Load(src string) (LuaRet, error) {
	return l.stdCall(func() (LuaRet, error) { return l.load(src) })
}

func (l *Luna) Close() {
	c := make(chan response, 1)
	l.q <- packet{
		func() (interface{}, error) {
			l.L.Close()
			return nil, nil
		},
		c,
	}
	<-c
}

func (l *Luna) getReturnValues() LuaRet {
	iret := l.L.GetTop()
	ret := make(LuaRet, iret)
	for i := l.L.GetTop(); i > 0; i = l.L.GetTop() {
		ret[i-1] = l.pop(i)
		l.L.Pop(1)
	}
	return ret
}

// Call calls a Lua function named <string> with the provided arguments.
func (l *Luna) call(name string, args ...interface{}) (ret LuaRet, err error) {
	top := l.L.GetTop()
	defer func() {
		if err == nil {
			return
		}

		// undo...
		l.L.SetTop(top)
	}()

	l.L.GetGlobal(name)
	for _, arg := range args {
		if l.pushBasicType(arg) {
			continue
		}

		if err = l.pushComplexType(arg); err != nil {
			return
		}
	}
	err = l.L.Call(len(args), lua.LUA_MULTRET)
	if err == nil {
		return l.getReturnValues(), nil
	}
	return nil, err
}

func (l *Luna) Call(name string, args ...interface{}) (LuaRet, error) {
	return l.stdCall(func() (LuaRet, error) { return l.call(name, args...) })
}

// CreateLibrary registers a library <name> with the given members.
// An error is returned if one of the members is of an unsupported type.
func (l *Luna) createLibrary(name string, members ...TableKeyValue) (err error) {
	top := l.L.GetTop()
	defer func() {
		if err != nil {
			l.L.SetTop(top)
		}
	}()

	l.L.NewTable()
	for _, kv := range members {
		if l.pushBasicType(kv.Val) {
			l.L.SetField(-2, kv.Key)
			continue
		}
		if err = l.pushComplexType(kv.Val); err != nil {
			return
		}
		l.L.SetField(-2, kv.Key)
	}

	l.L.SetGlobal(name)
	return
}

func (l *Luna) CreateLibrary(name string, members ...TableKeyValue) error {
	return l.errCall(func() error { return l.createLibrary(name, members...) })
}

func (l *Luna) pushBasicType(arg interface{}) bool {
	switch t := arg.(type) {
	case float32:
		l.L.PushNumber(float64(t))
	case float64:
		l.L.PushNumber(t)
	case int:
		l.L.PushInteger(int64(t))
	case int8:
		l.L.PushInteger(int64(t))
	case int16:
		l.L.PushInteger(int64(t))
	case int32:
		l.L.PushInteger(int64(t))
	case int64:
		l.L.PushInteger(t)
	case uint:
		l.L.PushInteger(int64(t))
	case uint8:
		l.L.PushInteger(int64(t))
	case uint16:
		l.L.PushInteger(int64(t))
	case uint32:
		l.L.PushInteger(int64(t))
	case uint64:
		l.L.PushInteger(int64(t))
	case string:
		l.L.PushString(t)
	case bool:
		l.L.PushBoolean(t)
	case nil:
		l.L.PushNil()
	default:
		return false
	}

	return true
}

func (l *Luna) pushStruct(arg reflect.Value) error {
	l.L.NewTable()
	typ := arg.Type()
	for i := 0; i < arg.NumField(); i++ {
		field := arg.Field(i)
		fieldTyp := typ.Field(i)
		if !field.CanInterface() {
			// probably an unexported field, don't try to push
			continue
		}
		if l.pushBasicType(field.Interface()) {
			l.L.SetField(-2, fieldTyp.Name)
			continue
		}

		if err := l.pushComplexType(field.Interface()); err != nil {
			return err
		}
		l.L.SetField(-2, fieldTyp.Name)
	}

	/*
		for i := 0; i < arg.NumMethod(); i++ {
			//method := arg.Method(i)
		}
	*/
	return nil
}

func (l *Luna) pushSlice(arg reflect.Value) error {
	l.L.NewTable()
	// for i := arg.Len() - 1; i >= 0; i-- {
	for i := 0; i < arg.Len(); i++ {
		// lua has 1-based arrays
		l.L.PushInteger(int64(i + 1))
		if l.pushBasicType(arg.Index(i).Interface()) {
			l.L.SetTable(-3)
			continue
		}

		if err := l.pushComplexType(arg.Index(i).Interface()); err != nil {
			return err
		}
		l.L.SetTable(-3)
	}
	return nil
}

func (l *Luna) pushMap(arg reflect.Value) error {
	l.L.NewTable()
	for _, k := range arg.MapKeys() {
		// push map key
		l.pushBasicType(k.Interface())
		// push value
		v := arg.MapIndex(k)
		if !l.pushBasicType(v.Interface()) {
			if err := l.pushComplexType(v.Interface()); err != nil {
				return err
			}
		}
		l.L.SetTable(-3)
	}
	return nil
}

func (l *Luna) pushComplexType(arg interface{}) (err error) {
	typ := reflect.TypeOf(arg)
	switch typ.Kind() {
	case reflect.Struct:
		return l.pushStruct(reflect.ValueOf(arg))
	case reflect.Func:
		l.L.PushGoFunction(wrapperGen(l, reflect.ValueOf(arg)))
	case reflect.Array, reflect.Slice:
		return l.pushSlice(reflect.ValueOf(arg))
	case reflect.Map:
		return l.pushMap(reflect.ValueOf(arg))
	case reflect.Ptr:
		// TODO: this should eventually use lua userdata instead of just dereferencing
		val := reflect.ValueOf(arg).Elem().Interface()
		if l.pushBasicType(val) {
			return nil
		}
		return l.pushComplexType(val)
	default:
		err = fmt.Errorf("Invalid type: %s", typ.Kind())
	}
	return
}

func (l *Luna) pop(i int) LuaValue {
	switch t := l.L.Type(i); t {
	case lua.LUA_TNUMBER:
		return LuaNumber(l.L.ToNumber(i))
	case lua.LUA_TBOOLEAN:
		return LuaBool(l.L.ToBoolean(i))
	case lua.LUA_TSTRING:
		return LuaString(l.L.ToString(i))
	case lua.LUA_TNIL:
		return LuaNil(nil)
	case lua.LUA_TTABLE:
		table := LuaTable{make(map[float64]LuaValue), make(map[string]LuaValue), make(map[bool]LuaValue)}

		l.L.PushNil()
		for l.L.Next(i) != 0 {
			switch l.L.Type(i + 1) {
			case lua.LUA_TNUMBER:
				table.indexed[l.L.ToNumber(i+1)] = l.pop(i + 2)
			case lua.LUA_TBOOLEAN:
				table.booled[l.L.ToBoolean(i+1)] = l.pop(i + 2)
			case lua.LUA_TSTRING:
				table.mapped[l.L.ToString(i+1)] = l.pop(i + 2)
			}

			l.L.Pop(1)
		}

		return table
		/*
			case lua.LUA_TFUNCTION:
				// TODO: implement
				fallthrough
			case lua.LUA_TUSERDATA:
				// TODO: implement
				fallthrough
			case lua.LUA_TTHREAD:
				// TODO: implement
				fallthrough
			case lua.LUA_TLIGHTUSERDATA:
				// TODO: implement
				fallthrough
		*/
	default:
		return luaTypeError(fmt.Sprintf("Unexpected type: %d", t))
	}
	return nil
}

func (l *Luna) tableToStruct(val reflect.Value, i int) error {
	l.L.PushNil()
	for l.L.Next(i) != 0 {
		// TODO: ignore bad values?
		if !l.L.IsString(-2) {
			return fmt.Errorf("Keys must be strings")
		}
		name := l.L.ToString(-2)
		field := val.FieldByName(name)
		if field.IsValid() {
			if err := l.set(field, -1); err != nil {
				return err
			}
		} else {
			// TODO: get rid of this log
			log.Println("Field doesn't exist:", name)
		}
		l.L.Pop(1)
	}
	l.L.Pop(1)
	return nil
}

func (l *Luna) set(val reflect.Value, i int) error {
	typ := val.Type()
	switch t := l.L.Type(i); t {
	case lua.LUA_TNUMBER:
		if typ.Kind() >= reflect.Int && typ.Kind() <= reflect.Int64 {
			val.SetInt(int64(l.L.ToNumber(i)))
		} else if typ.Kind() >= reflect.Uint && typ.Kind() <= reflect.Uint64 {
			val.SetUint(uint64(l.L.ToNumber(i)))
		} else if typ.Kind() == reflect.Float32 || typ.Kind() == reflect.Float64 {
			val.SetFloat(l.L.ToNumber(i))
		} else {
			return fmt.Errorf("Wrong type")
		}
	case lua.LUA_TBOOLEAN:
		val.SetBool(l.L.ToBoolean(i))
	case lua.LUA_TSTRING:
		val.SetString(l.L.ToString(i))
	case lua.LUA_TTABLE:
		return l.tableToStruct(val, i)
		/*
			case lua.LUA_TNIL:
				// TODO: implement
				fallthrough
			case lua.LUA_TFUNCTION:
				// TODO: implement
				fallthrough
			case lua.LUA_TUSERDATA:
				// TODO: implement
				fallthrough
			case lua.LUA_TTHREAD:
				// TODO: implement
				fallthrough
			case lua.LUA_TLIGHTUSERDATA:
				// TODO: implement
				fallthrough
		*/
	default:
		return fmt.Errorf("Unexpected type: %d", t)
	}
	return nil
}

// FunctionExists checks if a global function named <string> exists in the global table
func (l *Luna) FunctionExists(name string) bool {
	top := l.L.GetTop()
	l.L.GetGlobal(name)
	// the golua documentation for IsFunction indicates that it only works for
	// functions pushed from Go to lua, but it seems to work for all lua functions
	exists := l.L.IsFunction(l.L.GetTop())
	l.L.SetTop(top)
	return exists
}
