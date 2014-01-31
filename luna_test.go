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

func test(t *testing.T, expected, actual []string) {
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

func TestCallEmpty(t *testing.T) {
	noparamsExpected := []string{
		"Called without params\n",
	}

	l := New(LibBase | LibString | LibTable)
	defer l.Close()
	c := new(stdout)
	l.Stdout(c)
	l.Load(`function noparams()
				print("Called without params")
			end`)
	if _, err := l.Call("noparams"); err != nil {
		t.Error("Error calling 'noparams':", err)
	}
	test(t, noparamsExpected, *c)
}

func TestCallIntegers(t *testing.T) {
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

	l := New(LibBase | LibString | LibTable)
	defer l.Close()
	c := new(stdout)
	l.Stdout(c)
	l.Load(`function do_int(num)
				print(string.format("Called with number: %s:%s", type(num), num))
			end`)
	for _, i := range numbers {
		if _, err := l.Call("do_int", i); err != nil {
			t.Error("Error calling 'num':", err)
		}
		test(t, numExpected, *c)
		*c = (*c)[:0]
	}
}

func TestCallFloats(t *testing.T) {
	floats := []interface{}{
		float32(4.2),
		float64(4.2),
	}

	floatExpected := []string{
		"Called with float: number:4.2\n",
	}

	l := New(LibBase | LibString | LibTable)
	defer l.Close()
	c := new(stdout)
	l.Stdout(c)
	l.Load(`function float(num)
				print(string.format("Called with float: %s:%1.1f", type(num), num))
			end`)
	for _, i := range floats {
		if _, err := l.Call("float", i); err != nil {
			t.Error("Error calling 'float':", err)
		}
		test(t, floatExpected, *c)
		*c = (*c)[:0]
	}
}

func TestBasicTypes(t *testing.T) {
	basicTypesExpected := []string{
		"Called with basic types:\n",
		"string:hello\n",
		"boolean:true\n",
		"nil:nil\n",
	}

	l := New(LibBase | LibString | LibTable)
	defer l.Close()
	c := new(stdout)
	l.Stdout(c)
	l.Load(`function basicTypes(tStr, tBool, tNil)
				print("Called with basic types:")
				print(string.format("%s:%s", type(tStr), tStr))
				print(string.format("%s:%s", type(tBool), tostring(tBool)))
				print(string.format("%s:%s", type(tNil), tostring(tNil)))
			end`)

	if _, err := l.Call("basicTypes", "hello", true, nil); err != nil {
		t.Error("Error calling 'basicTypes':", err)
	}
	test(t, basicTypesExpected, *c)
}

func TestCall(t *testing.T) {
	type Data struct {
		A int
		B uint
	}
	type NestedData struct {
		A Data
	}
	type NestedDataPtr struct {
		A *Data
	}
	type DataWithPrivate struct {
		A int
		b string
	}

	sliceData := []int{3, 5, 7, 9}
	sliceExpected := []string{
		"Called with slice\n",
		"[1] = number:3\n",
		"[2] = number:5\n",
		"[3] = number:7\n",
		"[4] = number:9\n",
	}
	complexSliceData := []Data{{3, 5}}
	complexSliceExpected := []string{
		"Called with slice\n",
		"[1] = table:{A=3,B=5,}\n",
	}
	structData := Data{3, 2}
	structExpected := []string{
		"Called with struct\n",
		"[A] = number:3\n",
		"[B] = number:2\n",
	}
	structWithPrivateData := DataWithPrivate{3, "secret"}
	structWithPrivateExpected := []string{
		"Called with struct\n",
		"[A] = number:3\n",
	}
	nestedStructData := NestedData{Data{3, 2}}
	nestedStructExpected := []string{
		"Called with struct\n",
		"[A] = table:{A=3,B=2,}\n",
	}
	nestedStructPtrData := NestedDataPtr{&Data{3, 2}}
	nestedStructPtrExpected := []string{
		"Called with struct\n",
		"[A] = table:{A=3,B=2,}\n",
	}
	mapData := map[string]interface{}{"A": 3, "B": "hello"}
	mapExpected := []string{
		"Called with map\n",
		"[A] = number:3\n",
		"[B] = string:hello\n",
	}
  mapData2 := map[int]interface{}{3: "A", 5: 123}
  mapExpected2 := []string{
    "Called with map\n",
    "[3] = string:A\n",
    "[5] = number:123\n",
  }

	l := New(LibBase | LibString | LibTable)
	c := new(stdout)
	l.Stdout(c)
	file := `
function table_to_string(tab)
  local str = "{"
  for k,v in pairs(tab) do
    str = str..k.."="..tostring(v)..","
  end
  str = str.."}"
  return str
end

function struct(obj)
	print("Called with struct")
	object(obj)
end

function map(obj)
  print("Called with map")
  object(obj)
end

function object(obj)
	for k,v in pairs(obj) do
    if type(v) == "table" then
		print(string.format("[%s] = %s:%s", k, type(v), table_to_string(v)))
    else
		print(string.format("[%s] = %s:%s", k, type(v), tostring(v)))
    end
	end
end

function slice(arr)
	print("Called with slice")
	for k,v in pairs(arr) do
		if type(v) == "table" then
			print(string.format("[%d] = %s:%s", k, type(v), table_to_string(v)))
		else
			print(string.format("[%d] = %s:%s", k, type(v), tostring(v)))
		end
	end
end
`
	if err := l.Load(file); err != nil {
		t.Error("Error loading test lua code:", err)
	}

	if _, err := l.Call("struct", structData); err != nil {
		t.Error("Error calling 'struct':", err)
	}
	test(t, structExpected, *c)
	*c = (*c)[:0]

	// this will panic if it tries to push the private field
	if _, err := l.Call("struct", structWithPrivateData); err != nil {
		t.Error("Error calling 'struct' with an unexported field:", err)
	}
	test(t, structWithPrivateExpected, *c)
	*c = (*c)[:0]

	if _, err := l.Call("struct", nestedStructData); err != nil {
		t.Error("Error calling 'struct' with a nested struct:", err)
	}
	test(t, nestedStructExpected, *c)
	*c = (*c)[:0]

	if _, err := l.Call("struct", nestedStructPtrData); err != nil {
		t.Error("Error calling 'struct' with a nested struct pointer:", err)
	}
	test(t, nestedStructPtrExpected, *c)
	*c = (*c)[:0]

	if _, err := l.Call("map", mapData); err != nil {
		t.Error("Error calling 'map':", err)
	}
	test(t, mapExpected, *c)
	*c = (*c)[:0]

	if _, err := l.Call("map", mapData2); err != nil {
		t.Error("Error calling 'map':", err)
	}
	test(t, mapExpected2, *c)
	*c = (*c)[:0]

	if _, err := l.Call("slice", sliceData); err != nil {
		t.Error("Error calling 'slice':", err)
	}
	test(t, sliceExpected, *c)
	*c = (*c)[:0]

	if _, err := l.Call("slice", complexSliceData); err != nil {
		t.Error("Error calling 'slice' with a nested struct:", err)
	}
	test(t, complexSliceExpected, *c)
}

func TestCallCallback(t *testing.T) {
	var callbackCalled int
	callback := func() {
		callbackCalled++
	}

	l := New(LibBase | LibString | LibTable)
	defer l.Close()
	c := new(stdout)
	l.Stdout(c)
	l.Load(`function callback(cb)
				cb()
			end`)
	if _, err := l.Call("callback", callback); err != nil {
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
