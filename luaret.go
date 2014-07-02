package luna

import (
	"fmt"
)

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
