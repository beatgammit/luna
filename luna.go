package luna

import (
	"fmt"
	"github.com/aarzilli/golua/lua"
	"reflect"
)

type Lib uint
func (l Lib) LibBase() bool {
	return l & LibBase != 0
}
func (l Lib) LibIO() bool {
	return l & LibIO != 0
}
func (l Lib) LibMath() bool {
	return l & LibMath != 0
}
func (l Lib) LibPackage() bool {
	return l & LibPackage != 0
}
func (l Lib) LibString() bool {
	return l & LibString != 0
}
func (l Lib) LibTable() bool {
	return l & LibTable != 0
}
func (l Lib) LibOS() bool {
	return l & LibOS != 0
}

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
	AllLibs = LibBase | LibIO | LibMath | LibPackage | LibString | LibTable | LibOS
)

type TableKeyValue struct {
	Key string
	Val interface{}
}

type Luna struct {
	L *lua.State
}

func New(libs Lib) *Luna {
	l := &Luna{lua.NewState()}
	if libs == AllLibs {
		l.L.OpenLibs()
	} else {
		if libs.LibBase() {
			l.L.OpenBase()
		}
		if libs.LibIO() {
			l.L.OpenIO()
		}
		if libs.LibMath() {
			l.L.OpenMath()
		}
		if libs.LibPackage() {
			l.L.OpenPackage()
		}
		if libs.LibString() {
			l.L.OpenString()
		}
		if libs.LibTable() {
			l.L.OpenTable()
		}
		if libs.LibOS() {
			l.L.OpenOS()
		}
	}

	return l
}

// loads and executes a Lua source file
func (l *Luna) LoadFile(path string) {
	l.L.DoFile(path)
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
		if l.pushBasicType(field.Interface()) {
			l.L.SetField(-2, fieldTyp.Name)
			continue
		}

		if err := l.pushComplexType(field.Interface()); err != nil {
			return err
		}
		l.L.SetField(-2, fieldTyp.Name)
	}
	return nil
}

func (l *Luna) pushComplexType(arg interface{}) (err error) {
	typ := reflect.TypeOf(arg)
	switch typ.Kind() {
	case reflect.Struct:
		if err = l.pushStruct(reflect.ValueOf(arg)); err != nil {
			return
		}
	case reflect.Func:
		l.L.PushGoFunction(wrapperGen(l, reflect.ValueOf(arg)))
	case reflect.Ptr:
		/*
		if typ.Elem().Kind() == reflect.Struct {
			l.L.PushGoStruct(arg)
			break
		}
		*/
		fallthrough
	default:
		err = fmt.Errorf("Invalid type: %s", typ.Kind())
	}
	return
}

func (l *Luna) Call(name string, args ...interface{}) (err error) {
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
	l.L.Call(len(args), 0)
	return
}

func isInt(kind reflect.Kind) bool {
	return kind >= reflect.Int && kind <= reflect.Int64
}

func isUint(kind reflect.Kind) bool {
	return kind >= reflect.Uint && kind <= reflect.Uint64
}

func isFloat(kind reflect.Kind) bool {
	return kind == reflect.Float32 || kind == reflect.Float64
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
		for i := 1; i <= args; i++ {
			val := params[i-1]
			typ := val.Type()

			switch t := L.Type(i); t {
			case lua.LUA_TNUMBER:
				if isInt(typ.Kind()) {
					val.SetInt(int64(L.ToNumber(i)))
				} else if isUint(typ.Kind()) {
					val.SetUint(uint64(L.ToNumber(i)))
				} else if isFloat(typ.Kind()) {
					val.SetFloat(L.ToNumber(i))
				} else {
					panic("Wrong type")
				}
			case lua.LUA_TBOOLEAN:
				params[i-1].SetBool(L.ToBoolean(i))
			case lua.LUA_TSTRING:
				params[i-1].SetString(L.ToString(i))
			case lua.LUA_TNIL:
				// TODO: implement
				fallthrough
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
			default:
				// TODO: handle this better
				panic(fmt.Sprintf("Unexpected type: %d", t))
			}
		}

		ret := impl.Call(params)
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
