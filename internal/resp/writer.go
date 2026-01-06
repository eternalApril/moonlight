package resp

import (
	"io"
	"strconv"
)

type Encoder struct {
	writer io.Writer
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{writer: w}
}

func (e *Encoder) Write(v Value) error {
	var bytes []byte

	switch v.Type {
	case TypeInteger:
		bytes = []byte(":" + strconv.Itoa(v.Num) + "\r\n")

	case TypeSimpleString:
		bytes = []byte("+" + string(v.String) + "\r\n")

	case TypeError:
		bytes = []byte("-" + string(v.String) + "\r\n")

	case TypeBulkString:
		if v.IsNull {
			bytes = []byte("$-1\r\n")
		} else {
			bytes = []byte("$" + strconv.Itoa(len(v.String)) + "\r\n" + string(v.String) + "\r\n")
		}

	case TypeArray:
		if v.IsNull {
			bytes = []byte("*-1\r\n")
		} else {
			prefix := []byte("*" + strconv.Itoa(len(v.Array)) + "\r\n")
			if _, err := e.writer.Write(prefix); err != nil {
				return err
			}

			for _, el := range v.Array {
				if err := e.Write(el); err != nil {
					return err
				}
			}

			return nil
		}
	}
	_, err := e.writer.Write(bytes)
	return err
}
