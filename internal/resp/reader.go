package resp

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
)

var (
	// ErrInvalidEnding is returned when a RESP element does not end with "\r\n"
	ErrInvalidEnding = errors.New("invalid line ending")
)

// Decoder provides a high-level API for reading RESP values from an input stream
type Decoder struct {
	rd *bufio.Reader
}

// NewDecoder creates a new Decoder with an internal buffer for efficient reading
func NewDecoder(rd io.Reader) *Decoder {
	return &Decoder{rd: bufio.NewReader(rd)}
}

// Read parses the next complete RESP Value from the stream
func (d *Decoder) Read() (Value, error) {
	_type, err := d.rd.ReadByte()
	if err != nil {
		return Value{}, err
	}

	val := Value{
		Type: _type,
	}

	switch val.Type {
	case TypeSimpleString, TypeError:
		str, err := d.readLine()
		if err != nil {
			return Value{}, err
		}

		val.String = str
		return val, nil

	case TypeArray:
		array, err := d.readArray()
		if err != nil {
			return Value{}, err
		}

		if array == nil {
			val.IsNull = true
		}

		val.Array = array
		return val, nil

	case TypeInteger:
		num, err := d.readInteger()
		if err != nil {
			return Value{}, err
		}

		val.Integer = num
		return val, nil

	case TypeBulkString:
		str, err := d.readBulkString()
		if err != nil {
			return Value{}, err
		}

		if str == nil {
			val.IsNull = true
		}

		val.String = str
		return val, nil
	}

	return Value{}, errors.New("unexpected type")
}

// readLine reads bytes until \n and validates the \r\n sequence
func (d *Decoder) readLine() ([]byte, error) {
	line, err := d.rd.ReadSlice('\n')
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, ErrInvalidEnding
		}
		return nil, err
	}

	if len(line) < 2 || line[len(line)-2] != '\r' {
		return nil, ErrInvalidEnding
	}

	return line[:len(line)-2], nil
}

// readInteger parses a RESP integer
func (d *Decoder) readInteger() (int64, error) {
	line, err := d.readLine()
	if err != nil {
		return 0, err
	}

	i, err := strconv.ParseInt(string(line), 10, 64)
	if errors.Is(err, strconv.ErrSyntax) {
		return 0, ErrInvalidEnding
	}

	return i, nil
}

// readBulkString parses a bulk string
func (d *Decoder) readBulkString() ([]byte, error) {
	size, err := d.readInteger()
	if err != nil {
		return nil, err
	}

	if size == -1 {
		return nil, nil
	}

	buf := make([]byte, size)
	_, err = io.ReadFull(d.rd, buf)
	if err != nil {
		if errors.Is(err, io.ErrUnexpectedEOF) {
			return nil, ErrInvalidEnding
		}
		return nil, err
	}

	ending, err := d.rd.Peek(2)
	if err != nil || ending[0] != '\r' || ending[1] != '\n' {
		return nil, ErrInvalidEnding
	}
	d.rd.Discard(2) //nolint:errcheck

	return buf, nil
}

// readArray parses a RESP array recursively
func (d *Decoder) readArray() ([]Value, error) {
	size, err := d.readInteger()
	if err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return nil, ErrInvalidEnding
		}
		return nil, err
	}

	if size == -1 {
		return nil, nil
	}

	if size == 0 {
		return []Value{}, nil
	}

	buf := make([]Value, 0, size)

	for range size {
		el, err := d.Read()
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				return nil, ErrInvalidEnding
			}
			return nil, err
		}

		buf = append(buf, el)
	}

	return buf, nil
}

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
