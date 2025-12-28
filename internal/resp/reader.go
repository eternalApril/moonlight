package resp

import (
	"bufio"
	"errors"
	"io"
	"strconv"
)

var (
	ErrInvalidEnding = errors.New("invalid line ending")
)

type RespReader struct {
	rd *bufio.Reader
}

func NewReader(rd io.Reader) *RespReader {
	return &RespReader{rd: bufio.NewReader(rd)}
}

func (r *RespReader) Read() (Value, error) {
	_type, err := r.rd.ReadByte()
	if err != nil {
		return Value{}, err
	}

	val := Value{
		Type: _type,
	}

	switch val.Type {
	case TypeSimpleString, TypeError:
		str, err := r.readSimpleString()
		if err != nil {
			return Value{}, nil
		}

		val.String = str
		return val, nil
	case TypeArray:
	case TypeInteger:
		num, err := r.readInteger()
		if err != nil {
			return Value{}, err
		}

		val.Num = num
		return val, nil
	case TypeBulkString:
	}

	return Value{}, errors.New("unexpected type")
}

// readSimpleString read Simple String and Error from command
func (r *RespReader) readSimpleString() ([]byte, error) {
	line, err := r.rd.ReadBytes('\n')
	if err != nil {
		return nil, err
	}

	if len(line) < 2 || line[len(line)-2] != '\r' {
		return nil, ErrInvalidEnding
	}

	return line[:len(line)-2], nil
}

func (r *RespReader) readInteger() (int, error) {
	line, err := r.rd.ReadBytes('\n')
	if err != nil {
		return 0, err
	}

	// Command with integer cant be empty
	if len(line) < 3 || line[len(line)-2] != '\r' {
		return 0, ErrInvalidEnding
	}

	strNum := string(line[:len(line)-2])

	num, err := strconv.ParseInt(strNum, 10, 64)
	if err != nil {
		return 0, err
	}

	return int(num), nil
}
