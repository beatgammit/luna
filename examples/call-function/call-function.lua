function noparams()
	print("Called without params")
end

function basicTypes(numInt, numFloat, tStr, tBool, tNil)
	print("Called with basic types:")
	print(string.format("\t%s:%s", type(numInt), numInt))
	print(string.format("\t%s:%s", type(numFloat), numFloat))
	print(string.format("\t%s:%s", type(tStr), tStr))
	print(string.format("\t%s:%s", type(tBool), tostring(tBool)))
	print(string.format("\t%s:%s", type(tNil), tostring(tNil)))
end
