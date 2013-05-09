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
