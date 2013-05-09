package main

import (
	"fmt"
	"../.."
)

func main() {
	l := luna.New(luna.AllLibs)
	l.LoadFile("call-function.lua")

	err := l.Call("noparams")
	if err != nil {
		fmt.Println("Error calling 'noparams':", err)
	}
	err = l.Call("basicTypes", 3, 4.2, "hello", false, nil)
	if err != nil {
		fmt.Println("Error calling 'basicTypes':", err)
	}
}
