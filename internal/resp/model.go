package resp

const (
	TypeSimpleString = '+'
	TypeError        = '-'
	TypeInteger      = ':'
	TypeBulkString   = '$'
	TypeArray        = '*'
)

type Value struct {
	Type   byte
	Num    int    // Integer
	String []byte // SimpleString, Error, BulkString
	Array  []Value
	IsNull bool // For nil BulkString and nil Array
}
