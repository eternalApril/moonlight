package resp

// Reader defines the interface for decoding data from the Redis Serialization Protocol
type Reader interface {
	// Read parses the next available RESP value from the underlying source
	Read() (Value, error)
}

// Writer defines the interface for encoding and sending data using the Redis Serialization Protocol
type Writer interface {
	// Write encodes the given Value into the RESP format and writes it to the destination
	Write(v Value) error
}
