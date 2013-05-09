package main

import (
	"../.."
)

func main() {
	l := luna.New(luna.AllLibs)
	l.LoadFile("helloworld.lua")
}
