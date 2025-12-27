package resp

import "io"

type Reader interface {
	Read() (Value, error)
}

type Writer interface {
	Write(v Value) error
}

type Stream interface {
	Reader
	Writer
	io.Closer
}
