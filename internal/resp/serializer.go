package resp

import (
	"bytes"
)

// SerializeCommand uses a standard Encoder to convert the command to bytes
func SerializeCommand(cmd string, args []Value) ([]byte, error) {
	var buf bytes.Buffer
	enc := NewEncoder(&buf)

	elements := make([]Value, 1+len(args))

	elements[0] = MakeBulkString(cmd)

	copy(elements[1:], args)

	root := MakeArray(elements)

	if err := enc.Write(root); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
