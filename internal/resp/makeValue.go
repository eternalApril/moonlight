package resp

import "fmt"

// MakeSimpleString construct SimpleString Value from string
func MakeSimpleString(s string) Value {
	return Value{
		Type:   TypeSimpleString,
		String: []byte(s),
	}
}

// MakeError construct Error Value from string
func MakeError(s string) Value {
	return Value{
		Type:   TypeError,
		String: []byte(s),
	}
}

// MakeErrorWrongNumberOfArguments construct Error Value that command had wrong number of arguments for command
func MakeErrorWrongNumberOfArguments(cmd string) Value {
	return MakeError(fmt.Sprintf("wrong number of arguments for %s command", cmd))
}

// MakeBulkString construct BulkString Value from string
func MakeBulkString(s string) Value {
	return Value{
		Type:   TypeBulkString,
		String: []byte(s),
	}
}

// MakeNilBulkString construct nil BulkSting Value
func MakeNilBulkString() Value {
	return Value{
		Type:   TypeBulkString,
		IsNull: true,
	}
}

// MakeInteger construct Integer Value from int64
func MakeInteger(n int64) Value {
	return Value{
		Type:    TypeInteger,
		Integer: n,
	}
}

// MakeArray creates a standard RESP array containing the provided elements
func MakeArray(values []Value) Value {
	return Value{
		Type:  TypeArray,
		Array: values,
	}
}
