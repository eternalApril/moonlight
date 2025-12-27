package resp

import (
	"bufio"
	"errors"
	"io"
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

	switch _type {
	case TypeSimpleString, TypeError:
		str, err := r.readSimpleString()
		if err != nil {
			return Value{}, nil
		}

		return Value{
			Type:   _type,
			String: str,
		}, nil

	case TypeArray:
	case TypeInteger:
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
		return nil, errors.New("invalid line ending")
	}

	return line[:len(line)-2], nil
}
