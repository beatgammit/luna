package luna

import (
	"encoding"
	"fmt"
	"io"
	"log"
	"reflect"
	"strings"
	"sync"

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
	mut *sync.Mutex
}

func convertBasic(src LuaValue, dst interface{}) error {
	var destVal reflect.Value
	var ok bool
	if destVal, ok = dst.(reflect.Value); !ok {
		destVal = reflect.ValueOf(dst)
		if destVal.Type().Kind() != reflect.Ptr {
			return fmt.Errorf("Must pass a pointer type to Unmarshal")
		}
	}

	if v, ok := src.(LuaString); ok {
		dst := destVal.Interface()
		if unmarshaler, ok := dst.(encoding.TextUnmarshaler); ok {
			return unmarshaler.UnmarshalText([]byte(v))
		}
		dst = reflect.Indirect(destVal).Interface()
		if unmarshaler, ok := dst.(encoding.TextUnmarshaler); ok {
			return unmarshaler.UnmarshalText([]byte(v))
		}
	}

	destVal = reflect.Indirect(destVal)

	destType := destVal.Type()

	srcVal := reflect.ValueOf(src)
	if !srcVal.Type().ConvertibleTo(destType) {
		return fmt.Errorf("Cannot assign '%s' to '%s': given = %v", srcVal.Type(), destType, src)
	}
	destVal.Set(srcVal.Convert(destType))
	return nil
}

type LuaNumber float64

func (lv LuaNumber) Unmarshal(d interface{}) error {
	return convertBasic(lv, d)
}

type LuaBool bool

func (lv LuaBool) Unmarshal(d interface{}) error {
	return convertBasic(lv, d)
}

type LuaString string

func (lv LuaString) Unmarshal(d interface{}) error {
	return convertBasic(lv, d)
}

// the type here isn't significant, as long as it's nil-able
type LuaNil []int

func (lv LuaNil) Unmarshal(d interface{}) error {
	destVal := reflect.ValueOf(d)
	if destVal.Type().Kind() != reflect.Ptr {
		return fmt.Errorf("Must pass a pointer type to Unmarshal")
	}
	switch destVal.Type().Kind() {
	// I don't think lua can return a pointer or an interface,
	// but it's not hurting anything
	case reflect.Ptr, reflect.Map, reflect.Slice, reflect.Interface:
		destVal.Elem().Set(reflect.Zero(destVal.Elem().Type()))
	default:
		return fmt.Errorf("Unmarshal unsupported for %T", d)
	}
	return nil
}

type LuaTable struct {
	indexed map[float64]LuaValue
	mapped  map[string]LuaValue
	booled  map[bool]LuaValue
}

func (lv LuaTable) GetIndex(i float64) LuaValue {
	return lv.indexed[i]
}
func (lv LuaTable) Get(i string) LuaValue {
	return lv.mapped[i]
}
func (lv LuaTable) Map() map[string]LuaValue {
	return lv.mapped
}
func (lv LuaTable) Slice() (ret []LuaValue) {
	for i := 1; i <= len(lv.indexed); i++ {
		if v, ok := lv.indexed[float64(i)]; ok {
			ret = append(ret, v)
		} else {
			break
		}
	}
	return
}

func convertTableVal(src LuaValue, d interface{}) error {
	if _, ok := src.(LuaTable); ok {
		return src.Unmarshal(d)
	}
	return convertBasic(src, d)
}

func setMap(destVal reflect.Value, k interface{}, v LuaValue, destType reflect.Type) error {
	dest := reflect.New(destType.Elem())
	if err := convertTableVal(v, dest.Interface()); err != nil {
		return err
	}
	destVal.SetMapIndex(reflect.ValueOf(k), dest.Elem())
	return nil
}

func (lv LuaTable) Unmarshal(d interface{}) (err error) {
	var destVal reflect.Value
	var ok bool
	if destVal, ok = d.(reflect.Value); !ok {
		destVal = reflect.ValueOf(d)
		if destVal.Type().Kind() != reflect.Ptr {
			return fmt.Errorf("Must pass a pointer type to Unmarshal")
		}
	}
	destVal = reflect.Indirect(destVal)

	destType := destVal.Type()
	switch k := destType.Kind(); k {
	case reflect.Slice, reflect.Array:
		items := lv.Slice()
		if k == reflect.Slice {
			// recreate or change length of slice to fit our items
			if destVal.Len() >= len(items) || destVal.Cap() >= len(items) {
				destVal.SetLen(len(items))
			} else {
				destVal.Set(reflect.MakeSlice(destType, len(items), len(items)))
			}
		} else if destVal.Len() < len(items) {
			return fmt.Errorf("Array not big enough to hold values; needed '%d', have '%d'", len(items), destVal.Len())
		}

		for i, v := range items {
			dest := reflect.New(destType.Elem())
			if er := convertTableVal(v, dest.Interface()); er != nil {
				err = er
			} else {
				destVal.Index(i).Set(dest.Elem())
			}
		}
	case reflect.Struct:
		// TODO: find a better way to check for a non-existant field
		zero := reflect.Value{}
		for k, v := range lv.mapped {
			field := destVal.FieldByName(strings.Title(k))
			if field == zero {
				continue
			}

			if er := convertTableVal(v, field); err != nil {
				err = er
			}
		}
	case reflect.Map:
		if destVal.IsNil() {
			destVal.Set(reflect.MakeMap(destType))
		}

		keyType := destType.Key()
		if keyType.Kind() >= reflect.Int && keyType.Kind() <= reflect.Complex128 {
			for k, v := range lv.indexed {
				setMap(destVal, k, v, destType)
			}
		} else if keyType.Kind() == reflect.String {
			for k, v := range lv.mapped {
				setMap(destVal, k, v, destType)
			}
		} else if keyType.Kind() == reflect.Bool {
			for k, v := range lv.booled {
				setMap(destVal, k, v, destType)
			}
		} else if keyType.Kind() == reflect.Struct {
			return fmt.Errorf("Struct key types not currently supported")
		} else {
			return fmt.Errorf("Invalid key type: %s", keyType)
		}
	}
	return nil
}

type luaTypeError string

func (lv luaTypeError) Unmarshal(interface{}) error {
	return fmt.Errorf("%s", lv)
}

type LuaValue interface {
	Unmarshal(interface{}) error
}

// New creates a new Luna instance, opening all libs provided.
func New(libs Lib) *Luna {
	l := &Luna{lua.NewState(), libs, &sync.Mutex{}}
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

// Stdout changes where print() writes to (default os.Stdout).
// Note, this does **not** change anything in the io package.
func (l *Luna) Stdout(w io.Writer) {
	l.mut.Lock()
	defer l.mut.Unlock()
	l.L.Register("print", wrapperGen(l, reflect.ValueOf(printGen(w))))
}

// loads and executes a Lua source file
func (l *Luna) LoadFile(path string) error {
	l.mut.Lock()
	defer l.mut.Unlock()
	return l.L.DoFile(path)
}

// loads and executes Lua source
func (l *Luna) Load(src string) error {
	l.mut.Lock()
	defer l.mut.Unlock()
	return l.L.DoString(src)
}

func (l *Luna) Close() {
	l.mut.Lock()
	defer l.mut.Unlock()
	l.L.Close()
}

type LuaRet []LuaValue

func (lr LuaRet) Unmarshal(vals ...interface{}) error {
	if len(vals) != len(lr) {
		return fmt.Errorf("")
	}
	for i, v := range vals {
		if err := lr[i].Unmarshal(v); err != nil {
			return err
		}
	}
	return nil
}

// Call calls a Lua function named <string> with the provided arguments.
func (l *Luna) Call(name string, args ...interface{}) (ret LuaRet, err error) {
	l.mut.Lock()
	defer l.mut.Unlock()

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
		iret := l.L.GetTop()
		ret = make(LuaRet, iret)
		for i := l.L.GetTop(); i > 0; i = l.L.GetTop() {
			ret[i-1] = l.pop(i)
			l.L.Pop(1)
		}
	}
	// likely unnecessary
	l.L.SetTop(top)
	return
}

// CreateLibrary registers a library <name> with the given members.
// An error is returned if one of the members is of an unsupported type.
func (l *Luna) CreateLibrary(name string, members ...TableKeyValue) (err error) {
	l.mut.Lock()
	defer l.mut.Unlock()

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

// helper functions

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
