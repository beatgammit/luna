package main

import (
	"fmt"
	"github.com/beatgammit/luna"
)

type Data struct {
	A, B int
}

func main() {
	l := luna.New(luna.AllLibs)
	l.LoadFile("call-function.lua")

	if l.FunctionExists("asdf") {
		fmt.Println("function asdf shouldn't exist!")
	} else {
		fmt.Println("function asdf doesn't exist")
	}
	if !l.FunctionExists("noparams") {
		fmt.Println("function noparams should exist!")
	} else {
		fmt.Println("function noparams exists")
	}
	_, err := l.Call("noparams")
	if err != nil {
		fmt.Println("Error calling 'noparams':", err)
	}
	_, err = l.Call("basicTypes", 3, 4.2, "hello", false, nil)
	if err != nil {
		fmt.Println("Error calling 'basicTypes':", err)
	}
	_, err = l.Call("struct", Data{3, 2})
	if err != nil {
		fmt.Println("Error calling 'struct':", err)
	}
	_, err = l.Call("slice", []int{3, 2})
	if err != nil {
		fmt.Println("Error calling 'slice':", err)
	}
	_, err = l.Call("slice", []Data{{3, 2}})
	if err != nil {
		fmt.Println("Error calling 'slice':", err)
	}
	ret, err := l.Call("ret")
	if err != nil {
		fmt.Println("Error calling 'ret':", err)
	} else {
		fmt.Println("Return values:", ret)
	}
}
