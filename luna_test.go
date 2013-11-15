package luna

import (
	"os"
	"testing"
)

func (l *Luna) loaded(libs Lib) bool {
	if libs&LibBase != 0 {
		l.L.GetGlobal("_VERSION")
		if l.L.IsNil(-1) {
			return false
		}
	}
	if libs&LibIO != 0 {
		l.L.GetGlobal("io")
		if l.L.IsNil(-1) {
			return false
		}
	}
	if libs&LibMath != 0 {
		l.L.GetGlobal("math")
		if l.L.IsNil(-1) {
			return false
		}
	}
	if libs&LibPackage != 0 {
		l.L.GetGlobal("package")
		if l.L.IsNil(-1) {
			return false
		}
	}
	if libs&LibString != 0 {
		l.L.GetGlobal("string")
		if l.L.IsNil(-1) {
			return false
		}
	}
	if libs&LibTable != 0 {
		l.L.GetGlobal("table")
		if l.L.IsNil(-1) {
			return false
		}
	}
	if libs&LibOS != 0 {
		l.L.GetGlobal("os")
		if l.L.IsNil(-1) {
			return false
		}
	}
	return true
}

type stdout []string

func (msgs *stdout) Write(msg []byte) (int, error) {
	*msgs = append(*msgs, string(msg))
	return len(msg), nil
}

func TestLoad(t *testing.T) {
	msg := "Hello World"
	src := "print(\"" + msg + "\")"

	c := new(stdout)
	l := New(NoLibs)
	l.Stdout(c)
	if err := l.Load(src); err != nil {
		t.Error("Error loading lua code:", err)
	}

	if len(*c) != 1 {
		t.Error("Should have exactly one message", c)
	} else if (*c)[0] != msg+"\n" {
		t.Errorf("Expected '%s', printed '%s'", msg+"\n", (*c)[0])
	}
}

func TestLoadFile(t *testing.T) {
	fname := "test.lua"
	msg := "Hello World"

	f, err := os.Create(fname)
	if err != nil {
		panic(err)
	}
	defer os.Remove(fname)
	f.Write([]byte("print(\"" + msg + "\")"))
	f.Close()

	c := new(stdout)
	l := New(NoLibs)
	l.Stdout(c)
	if err := l.LoadFile(fname); err != nil {
		t.Error("Error loading lua script:", err)
	}

	if len(*c) != 1 {
		t.Error("Should have exactly one message", c)
	} else if (*c)[0] != msg+"\n" {
		t.Errorf("Expected '%s', printed '%s'", msg+"\n", (*c)[0])
	}
}

func TestNew(t *testing.T) {
	libs := []Lib{
		LibBase,
		LibIO,
		LibMath,
		LibPackage,
		LibString,
		LibTable,
		LibOS,
	}

	for i, l := 0, len(libs); i < l-1; i++ {
		lib := libs[i]
		for j := i + 1; j < l; j++ {
			lib |= libs[j]
		}
		libs = append(libs, lib)
	}

	for _, lib := range libs {
		l := New(lib)
		if !l.loaded(lib) {
			t.Error("Library not loaded:", lib)
		}
	}
}

func TestCreateLibrary(t *testing.T) {
	var funcCalled int
	var paramValue int
	paramPassed := 5
	fun := func(val int) {
		funcCalled++
		paramValue = val
	}

	l := New(LibBase)
	libMembers := []TableKeyValue{
		{"fun", fun},
		{"val", paramPassed},
	}
	err := l.CreateLibrary("testlib", libMembers...)
	if err != nil {
		t.Fatal("Error creating library:", err)
	}
	if err := l.Load("testlib.fun(testlib.val)"); err != nil {
		t.Error("Error loading test lua code:", err)
	}
	if funcCalled != 1 {
		t.Error("Library function not called exactly 1 time:", funcCalled)
	}
	if paramValue != paramPassed {
		t.Error("Expected parameter: '%d', Passed: '%d'", paramPassed, paramValue)
	}
}

func TestInvalidLibrary(t *testing.T) {
	l := New(LibBase)
	libMembers := []TableKeyValue{
		{"invalid", make(chan bool)},
	}
	err := l.CreateLibrary("testlib", libMembers...)
	if err == nil {
		t.Error("Expected library load to fail")
	}
}

func TestCall(t *testing.T) {
	type Data struct {
		A int
		B uint
	}

	type NestedData struct {
		A Data
	}

	test := func(expected, actual []string) {
		for i, act := range actual {
			if i >= len(expected) {
				t.Error("Extra print sent:", act)
				continue
			}
			if expected[i] != act {
				t.Errorf("Expected: '%s', Actual: '%s'", expected[i], act)
			}
		}
	}

	var callbackCalled int
	callback := func() {
		callbackCalled++
	}

	noparamsExpected := []string{
		"Called without params\n",
	}
	numbers := []interface{}{
		int(5),
		int8(5),
		int16(5),
		int32(5),
		int64(5),
		uint(5),
		uint8(5),
		uint16(5),
		uint32(5),
		uint64(5),
	}
	numExpected := []string{
		"Called with number: number:5\n",
	}
	floats := []interface{}{
		float32(4.2),
		float64(4.2),
	}
	floatExpected := []string{
		"Called with float: number:4.2\n",
	}
	basicTypesExpected := []string{
		"Called with basic types:\n",
		"string:hello\n",
		"bool:true\n",
		"nil:nil\n",
	}
	structExpected := []string{
		"Called with struct\n",
		"[A] = number:3\n",
		"[B] = number:2\n",
	}
	sliceData := []int{3, 5}
	sliceExpected := []string{
		"Called with slice\n",
		"[0] = number:3\n",
		"[1] = number:5\n",
	}
	complexSliceData := []Data{{3, 5}}
	complexSliceExpected := []string{
		"Called with complex slice\n",
		"[0] = table:{A=3,B=5}\n",
	}
	nestedStructExpected := []string{
		"Called with struct\n",
		"[A] = table:{A=3,B=2}\n",
	}

	l := New(LibBase)
	c := new(stdout)
	l.Stdout(c)
	file := `
function noparams()
	print("Called without params")
end

function num(num)
	print(string.format("Called with number: %s:%s", type(numInt), numInt))
end

function basicTypes(tStr, tBool, tNil)
	print("Called with basic types:")
	print(string.format("%s:%s", type(tStr), tStr))
	print(string.format("%s:%s", type(tBool), tostring(tBool)))
	print(string.format("%s:%s", type(tNil), tostring(tNil)))
end

function struct(obj)
	print("Called with struct")
	for k,v in pairs(obj) do
		print(string.format("[%s] = %s:%s", k, type(v), tostring(v)))
	end
end

function slice(arr)
	print("Called with complex slice")
	for k,v in pairs(arr) do
		print(string.format("[%d] = %s:%s", k, type(v), tostring({A=v.A,B=v.B})))
	end
end

function slice(arr)
	print("Called with slice")
	for k,v in pairs(arr) do
		print(string.format("[%d] = %s:%s", k, type(v), tostring(v)))
	end
end
function callback(cb)
  cb()
end
	`
	if err := l.Load(file); err != nil {
		t.Error("Error loading test lua code:", err)
	}

	_, err := l.Call("noparams")
	if err != nil {
		t.Error("Error calling 'noparams':", err)
	}
	test(noparamsExpected, *c)
	*c = (*c)[:0]

	for _, i := range numbers {
		_, err = l.Call("num", i)
		if err != nil {
			t.Error("Error calling 'num':", err)
		}
		test(numExpected, *c)
		*c = (*c)[:0]
	}
	for _, i := range floats {
		_, err = l.Call("num", i)
		if err != nil {
			t.Error("Error calling 'num':", err)
		}
		test(floatExpected, *c)
		*c = (*c)[:0]
	}
	_, err = l.Call("basiicTypes", "hello", true, nil)
	if err != nil {
		t.Error("Erroir calling 'basicTypes':", err)
	}
	test(basicTypesExpected, *c)
	*c = (*c)[:0]

	_, err = l.Call("struct", Data{3, 2})
	if err != nil {
		t.Error("Error calling 'struct':", err)
	}
	test(structExpected, *c)
	*c = (*c)[:0]

	_, err = l.Call("struct", NestedData{Data{3, 2}})
	if err != nil {
		t.Error("Error calling 'struct' with a nested struct:", err)
	}
	test(nestedStructExpected, *c)
	*c = (*c)[:0]

	_, err = l.Call("slice", sliceData)
	if err != nil {
		t.Error("Error calling 'slice' with a nested struct:", err)
	}
	test(sliceExpected, *c)
	*c = (*c)[:0]

	_, err = l.Call("complexSlice", complexSliceData)
	if err != nil {
		t.Error("Error calling 'complexSlice' with a nested struct:", err)
	}
	test(complexSliceExpected, *c)
	*c = (*c)[:0]

	_, err = l.Call("callback", callback)
	if err != nil {
		t.Error("Error calling 'callback':", err)
	} else if callbackCalled != 1 {
		t.Error("callback not called exactly one time:", callbackCalled)
	}
}

func TestInvalidCall(t *testing.T) {
	l := New(LibBase)
	type invalidStruct struct {
		C chan bool
	}
	type empty struct {
	}
	_, err := l.Call("noexists", invalidStruct{})
	if err == nil {
		t.Error("Error expected")
	}
	// TODO: remove when pointers are implemented
	_, err = l.Call("noexists", l)
	if err == nil {
		t.Error("Error expected")
	}

	_, err = l.Call("noexists", []chan bool{make(chan bool)})
	if err == nil {
		t.Error("Error expected")
	}
}

func TestLuaTableToGoStruct(t *testing.T) {
	type Data struct {
		A int
		B uint
		C float64
		D bool
		E string
	}

	var called int
	var data Data
	expected := Data{3, 2, 4.2, true, "hello"}
	test := func(d Data) {
		called++
		data = d
	}

	libMembers := []TableKeyValue{
		{"func", test},
	}

	l := New(LibBase)
	if err := l.Load("function callMe() testlib.func({A=3,B=2,C=4.2,D=true,E='hello',F=nil,G=callMe,Z='hi'}) end"); err != nil {
		t.Error("Error loading test code:", err)
	}
	err := l.CreateLibrary("testlib", libMembers...)
	if err != nil {
		t.Fatal("Error loading library:", err)
	}
	l.Call("callMe")
	if called != 1 {
		t.Error("Function not called exactly one time")
	}
	if data != expected {
		t.Errorf("Exected: '%+v', Sent: '%+v'", expected, data)
	}
}

func TestInvalidLuaToGo(t *testing.T) {
	test := func(d string) {
	}

	libMembers := []TableKeyValue{
		{"func", test},
	}

	l := New(LibBase)
	code := `
function callMe()
	testlib.func(5)
	testlib.func(5, 6)
end`
	if err := l.Load(code); err != nil {
		t.Error("Error loading test code:", err)
	}
	err := l.CreateLibrary("testlib", libMembers...)
	if err != nil {
		t.Fatal("Error loading library:", err)
	}
	_, err = l.Call("callMe")
}

func TestReturns(t *testing.T) {
	l := New(LibBase)
	code := `
function echo(v)
	return v
end
function returnMult()
	return 5, 3
end`
	if err := l.Load(code); err != nil {
		t.Error("Error loading test code:", err)
	}

	calls := []interface{}{
		4.2, "hi", true, nil,
	}

	for _, val := range calls {
		ret, err := l.Call("echo", val)
		if err != nil {
			t.Error("Error calling echo:", err)
			continue
		}

		if len(ret) != 1 {
			t.Errorf("Incorrect number of return vals. Expected '%d', Actual: '%d'", 1, len(ret))
		} else if val != ret[0] {
			t.Errorf("Expected: %v, Actual: %v", val, ret[0])
		}
	}
}
