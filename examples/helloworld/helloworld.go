package main

import (
	"fmt"

	"github.com/beatgammit/luna"
)

func main() {
	l := luna.New(luna.AllLibs)
	ret, err := l.LoadFile("helloworld.lua")
	if err != nil {
		fmt.Println("Error loading file:", err)
		return
	}

	if len(ret) > 0 {
		for _, v := range ret {
			fmt.Println(v)
		}
	}
}
