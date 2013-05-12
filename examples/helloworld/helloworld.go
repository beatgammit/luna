package main

import (
	"github.com/beatgammit/luna"
)

func main() {
	l := luna.New(luna.AllLibs)
	l.LoadFile("helloworld.lua")
}
