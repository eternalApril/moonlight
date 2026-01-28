package resp

import (
	"bufio"
	"io"
	"strconv"
)

// Encoder handles the serialization of RESP Value objects into an output stream
type Encoder struct {
	writer *bufio.Writer
}

// NewEncoder initializes an Encoder with a buffered writer
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{
		writer: bufio.NewWriter(w)}
}

// Write serializes a RESP Value and writes it to the underlying stream
func (e *Encoder) Write(v Value) error {
	var err error

	switch v.Type {
	case TypeInteger:
		err = e.writeHeader(':', v.Integer)

	case TypeSimpleString:
		err = e.writeRaw('+', v.String)

	case TypeError:
		err = e.writeRaw('-', v.String)

	case TypeBulkString:
		if v.IsNull {
			_, err = e.writer.WriteString("$-1\r\n")
		} else {
			if err = e.writeHeader('$', int64(len(v.String))); err == nil {
				if _, err = e.writer.Write(v.String); err == nil {
					_, err = e.writer.WriteString("\r\n")
				}
			}
		}

	case TypeArray:
		if v.IsNull {
			_, err = e.writer.WriteString("*-1\r\n")
		} else {
			if err = e.writeHeader('*', int64(len(v.Array))); err == nil {
				for _, el := range v.Array {
					if err = e.Write(el); err != nil {
						break
					}
				}
			}
		}
	}

	if err != nil {
		return err
	}

	return e.writer.Flush()
}

// WriteHeader writes the type prefix, numeric value, and CRLF
func (e *Encoder) writeHeader(prefix byte, n int64) error {
	if err := e.writer.WriteByte(prefix); err != nil {
		return err
	}
	e.appendInt(n)
	_, err := e.writer.WriteString("\r\n")
	return err
}

// writeRaw writes the type prefix, raw bytes, and CRLF (for SimpleString and Error)
func (e *Encoder) writeRaw(prefix byte, b []byte) error {
	if err := e.writer.WriteByte(prefix); err != nil {
		return err
	}
	if _, err := e.writer.Write(b); err != nil {
		return err
	}
	_, err := e.writer.WriteString("\r\n")
	return err
}

// appendInt converts an integer to a string and writes it to the buffer
func (e *Encoder) appendInt(n int64) {
	b := e.writer.AvailableBuffer()
	b = strconv.AppendInt(b, n, 10)
	e.writer.Write(b) //nolint:errcheck
}
