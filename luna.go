package luna

import (
	"fmt"
	"github.com/aarzilli/golua/lua"
	"io"
	"log"
	"reflect"
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
}

// New creates a new Luna instance, opening all libs provided.
func New(libs Lib) *Luna {
	l := &Luna{lua.NewState(), libs}
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

	return l
}

// printGen generates a print() function that writes to the given io.Writer.
func printGen(w io.Writer) func(...string) {
	return func(args ...string) {
		// TODO: support interface{} parameters
		var _args []interface{}
		for _, arg := range args {
			_args = append(_args, arg)
		}
		fmt.Fprintln(w, _args...)
	}
}

// Stdout changes where print() writes to (default os.Stdout).
// Note, this does **not** change anything in the io package.
func (l *Luna) Stdout(w io.Writer) {
	l.L.Register("print", wrapperGen(l, reflect.ValueOf(printGen(w))))
}

// loads and executes a Lua source file
func (l *Luna) LoadFile(path string) error {
	return l.L.DoFile(path)
}

// loads and executes Lua source
func (l *Luna) Load(src string) error {
	return l.L.DoString(src)
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
			return nil
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
	typ := arg.Type()
	if typ.Key().Kind() != reflect.String {
		return fmt.Errorf("map key type: %s invalid, must be string", typ.Key())
	}
	for _, k := range arg.MapKeys() {
		v := arg.MapIndex(k)
		if l.pushBasicType(v.Interface()) {
			l.L.SetField(-2, k.Interface().(string))
			continue
		}
		if err := l.pushComplexType(v.Interface()); err != nil {
			return err
		}
		l.L.SetField(-2, k.Interface().(string))
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

func (l *Luna) pop(i int) interface{} {
	switch t := l.L.Type(i); t {
	case lua.LUA_TNUMBER:
		return l.L.ToNumber(i)
	case lua.LUA_TBOOLEAN:
		return l.L.ToBoolean(i)
	case lua.LUA_TSTRING:
		return l.L.ToString(i)
	case lua.LUA_TNIL:
		return nil
		/*
			case lua.LUA_TTABLE:
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

// Call calls a Lua function named <string> with the provided arguments.
func (l *Luna) Call(name string, args ...interface{}) (ret []interface{}, err error) {
	top := l.L.GetTop()
	defer func() {
		if err == nil {
			return
		}

		// undo...
		l.L.SetTop(top)
	}()

	l.L.GetField(lua.LUA_GLOBALSINDEX, name)
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
		iret := l.L.GetTop()
		for i := 1; i < iret+1; i++ {
			ret = append(ret, l.pop(i))
		}
	}
	l.L.SetTop(top)
	return
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

func wrapperGen(l *Luna, impl reflect.Value) lua.LuaGoFunction {
	typ := impl.Type()
	params := make([]reflect.Value, typ.NumIn())
	for i := range params {
		params[i] = reflect.New(typ.In(i)).Elem()
	}

	return func(L *lua.State) int {
		args := L.GetTop()
		if args < len(params) {
			panic(fmt.Sprintf("Args: %d, Params: %d", args, len(params)))
		}

		if typ.IsVariadic() {
			params[len(params)-1] = params[len(params)-1].Slice(0, 0)
		}

		for i := 1; i <= args; i++ {
			if i >= len(params) && typ.IsVariadic() {
				val := reflect.New(params[i-1].Type().Elem()).Elem()
				l.set(val, i)
				params[i-1] = reflect.Append(params[i-1], val)
			} else if i > len(params) {
				// ignore extra args
				break
			} else {
				l.set(params[i-1], i)
			}
		}

		var ret []reflect.Value
		if typ.IsVariadic() {
			ret = impl.CallSlice(params)
		} else {
			ret = impl.Call(params)
		}
		for _, val := range ret {
			if l.pushBasicType(val.Interface()) {
				continue
			}
			if err := l.pushComplexType(val.Interface()); err != nil {
				panic(err)
			}
		}
		return len(ret)
	}
}

// CreateLibrary registers a library <name> with the given members.
// An error is returned if one of the members is of an unsupported type.
func (l *Luna) CreateLibrary(name string, members ...TableKeyValue) (err error) {
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
