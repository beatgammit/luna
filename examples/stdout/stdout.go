package main

import (
	"fmt"

	"github.com/beatgammit/luna"
)

type Data struct {
	A, B int
}

type printer string

func (p printer) Write(msg []byte) (int, error) {
	fmt.Print(p, string(msg))
	return len(msg), nil
}

func main() {
	l := luna.New(luna.AllLibs)
	_, err := l.LoadFile("stdout.lua")
	if err != nil {
		fmt.Println("Error loading file:", err)
	}
	l.Stdout(printer("test: "))

	_, err = l.Call("hello")
	if err != nil {
		fmt.Println("Error calling 'hello':", err)
	}
}
