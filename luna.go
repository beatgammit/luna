package luna

import (
	"github.com/aarzilli/golua/lua"
)

type Lib uint
func (l Lib) LibBase() bool {
	return l & LibBase != 0
}
func (l Lib) LibIO() bool {
	return l & LibIO != 0
}
func (l Lib) LibMath() bool {
	return l & LibMath != 0
}
func (l Lib) LibPackage() bool {
	return l & LibPackage != 0
}
func (l Lib) LibString() bool {
	return l & LibString != 0
}
func (l Lib) LibTable() bool {
	return l & LibTable != 0
}
func (l Lib) LibOS() bool {
	return l & LibOS != 0
}

const (
	LibBase Lib = 1 << iota
	LibIO
	LibMath
	LibPackage
	LibString
	LibTable
	LibOS
)

const (
	AllLibs = LibBase | LibIO | LibMath | LibPackage | LibString | LibTable | LibOS
)

type Luna struct {
	L *lua.State
}

func New(libs Lib) *Luna {
	l := &Luna{lua.NewState()}
	if libs == AllLibs {
		l.L.OpenLibs()
	} else {
		if libs.LibBase() {
			l.L.OpenBase()
		}
		if libs.LibIO() {
			l.L.OpenIO()
		}
		if libs.LibMath() {
			l.L.OpenMath()
		}
		if libs.LibPackage() {
			l.L.OpenPackage()
		}
		if libs.LibString() {
			l.L.OpenString()
		}
		if libs.LibTable() {
			l.L.OpenTable()
		}
		if libs.LibOS() {
			l.L.OpenOS()
		}
	}

	return l
}

// loads and executes a Lua source file
func (l *Luna) LoadFile(path string) {
	l.L.DoFile(path)
}
