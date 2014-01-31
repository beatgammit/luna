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

function struct(obj)
	print("Called with struct")
	for k,v in pairs(obj) do
		print(string.format("\t[%s] = %s:%s", k, type(v), tostring(v)))
	end
end

function slice(arr)
	print("Called with slice")
	for k,v in pairs(arr) do
		if type(v) == "table" then
			print(string.format("Printing struct at [%d]", k))
			struct(v)
		else
			print(string.format("\t[%d] = %s:%s", k, type(v), tostring(v)))
		end
	end
end

function map(m)
  print("pushed "..string.len(table.concat(m))/string.len(m[1]).." strings")
  return "everything worked I guess"
end

function ret()
	return 5, 3
end
