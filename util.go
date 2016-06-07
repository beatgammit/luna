package luna

import (
	"fmt"
	"io"
	"reflect"

	"github.com/aarzilli/golua/lua"
)

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

	return func(L *lua.State) int {
		for i := range params {
			params[i] = reflect.New(typ.In(i)).Elem()
		}
		args := L.GetTop()
		if args < len(params) {
			panic(fmt.Sprintf("Args: %d, Params: %d", args, len(params)))
		}

		var varargs reflect.Value
		if typ.IsVariadic() {
			params[len(params)-1] = params[len(params)-1].Slice(0, 0)
			varargs = params[len(params)-1]
		}

		for i := 1; i <= args; i++ {
			if i >= len(params) && typ.IsVariadic() {
				val := reflect.New(varargs.Type().Elem()).Elem()
				l.set(val, i)
				varargs = reflect.Append(varargs, val)
			} else if i > len(params) {
				// ignore extra args
				break
			} else {
				if err := l.set(params[i-1], i); err != nil {
					panic(err)
				}
			}
		}

		var ret []reflect.Value
		if typ.IsVariadic() {
			params[len(params)-1] = varargs
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
