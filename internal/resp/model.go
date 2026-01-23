package resp

const (
	TypeSimpleString = '+'
	TypeError        = '-'
	TypeInteger      = ':'
	TypeBulkString   = '$'
	TypeArray        = '*'
)

type Value struct {
	String  []byte // SimpleString, Error, BulkString
	Array   []Value
	Integer int64 // Integer
	Type    byte
	IsNull  bool // For nil BulkString and nil Array
}
