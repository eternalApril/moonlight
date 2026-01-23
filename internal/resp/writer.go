package resp

import (
	"bufio"
	"io"
	"strconv"
)

type Encoder struct {
	writer *bufio.Writer
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{
		writer: bufio.NewWriter(w)}
}

//nolint:gocyclo
func (e *Encoder) Write(v Value) error {
	switch v.Type {
	case TypeInteger:
		if err := e.writer.WriteByte(':'); err != nil {
			return err
		}
		e.appendInt(v.Integer)
		if _, err := e.writer.WriteString("\r\n"); err != nil {
			return err
		}

	case TypeSimpleString:
		if err := e.writer.WriteByte('+'); err != nil {
			return err
		}
		if _, err := e.writer.Write(v.String); err != nil {
			return err
		}
		if _, err := e.writer.WriteString("\r\n"); err != nil {
			return err
		}

	case TypeError:
		if err := e.writer.WriteByte('-'); err != nil {
			return err
		}
		if _, err := e.writer.Write(v.String); err != nil {
			return err
		}
		if _, err := e.writer.WriteString("\r\n"); err != nil {
			return err
		}

	case TypeBulkString:
		if v.IsNull {
			if _, err := e.writer.WriteString("$-1\r\n"); err != nil {
				return err
			}
		} else {
			if err := e.writer.WriteByte('$'); err != nil {
				return err
			}
			e.appendInt(int64(len(v.String)))
			if _, err := e.writer.WriteString("\r\n"); err != nil {
				return err
			}
			if _, err := e.writer.Write(v.String); err != nil {
				return err
			}
			if _, err := e.writer.WriteString("\r\n"); err != nil {
				return err
			}
		}

	case TypeArray:
		if v.IsNull {
			if _, err := e.writer.WriteString("*-1\r\n"); err != nil {
				return err
			}
		} else {
			if err := e.writer.WriteByte('*'); err != nil {
				return err
			}
			e.appendInt(int64(len(v.Array)))
			if _, err := e.writer.WriteString("\r\n"); err != nil {
				return err
			}

			for _, el := range v.Array {
				if err := e.Write(el); err != nil {
					return err
				}
			}
		}
	}

	return e.writer.Flush()
}

func (e *Encoder) appendInt(n int64) {
	b := e.writer.AvailableBuffer()
	b = strconv.AppendInt(b, n, 10)
	e.writer.Write(b) //nolint:errcheck
}
