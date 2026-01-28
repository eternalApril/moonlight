package resp

// RESP type identifiers as defined by the Redis Serialization Protocol
const (
	TypeSimpleString = '+' // +<data>\r\n
	TypeError        = '-' // -<data>\r\n
	TypeInteger      = ':' // :<data>\r\n
	TypeBulkString   = '$' // $<length>\r\n<data>\r\n
	TypeArray        = '*' // *<len>\r\n<elements>
)

// Value represents a single RESP entity
type Value struct {
	// String holds the raw bytes for SimpleStrings, Errors, and BulkStrings
	String []byte

	// Array contains a slice of Value objects if the Type is TypeArray
	Array []Value

	// Integer holds the numeric value if the Type is TypeInteger
	Integer int64

	// Type determines which field String, Array, or Integer is populated
	Type byte

	// IsNull indicates if the value represents a Null Bulk String or Null Array
	IsNull bool
}
