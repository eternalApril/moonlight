package storage

type DataType byte

const (
	TypeString DataType = iota + 1
	TypeList
	TypeSet
	TypeHash
	TypeZSet
)

// Entity generic container for value
type Entity struct {
	Type  DataType
	Value interface{}
}

// HashField represents a single field inside a Hash with its own TTL
type HashField struct {
	Value    string
	ExpireAt int64 // Unix nanoseconds. 0 means no TTL
}
