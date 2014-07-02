package luna

import (
	"encoding"
	"fmt"
	"reflect"
	"strings"
)

type LuaValue interface {
	Unmarshal(interface{}) error
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
