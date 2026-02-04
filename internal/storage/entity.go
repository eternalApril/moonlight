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
