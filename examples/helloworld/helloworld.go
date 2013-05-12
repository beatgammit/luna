package main

import (
	"luna"
)

func main() {
	l := luna.New(luna.AllLibs)
	l.LoadFile("helloworld.lua")
}
