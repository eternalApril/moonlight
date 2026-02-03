package resp_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/eternalApril/moonlight/internal/resp"
)

func TestEncoder_Write(t *testing.T) {
	tests := []struct {
		name     string
		input    resp.Value
		expected string
	}{
		{
			name:     "Integer positive",
			input:    resp.Value{Type: resp.TypeInteger, Integer: 100},
			expected: ":100\r\n",
		},
		{
			name:     "Integer negative",
			input:    resp.Value{Type: resp.TypeInteger, Integer: -42},
			expected: ":-42\r\n",
		},
		{
			name:     "Simple String",
			input:    resp.Value{Type: resp.TypeSimpleString, String: []byte("OK")},
			expected: "+OK\r\n",
		},
		{
			name:     "Error",
			input:    resp.Value{Type: resp.TypeError, String: []byte("Error message")},
			expected: "-Error message\r\n",
		},
		{
			name:     "Bulk String",
			input:    resp.Value{Type: resp.TypeBulkString, String: []byte("hello")},
			expected: "$5\r\nhello\r\n",
		},
		{
			name:     "Bulk String Empty",
			input:    resp.Value{Type: resp.TypeBulkString, String: []byte("")},
			expected: "$0\r\n\r\n",
		},
		{
			name:     "Bulk String Null",
			input:    resp.Value{Type: resp.TypeBulkString, IsNull: true},
			expected: "$-1\r\n",
		},
		{
			name: "Array of Strings",
			input: resp.Value{
				Type: resp.TypeArray,
				Array: []resp.Value{
					{Type: resp.TypeBulkString, String: []byte("fff")},
					{Type: resp.TypeBulkString, String: []byte("ttt")},
				},
			},
			expected: "*2\r\n$3\r\nfff\r\n$3\r\nttt\r\n",
		},
		{
			name:     "Array Null",
			input:    resp.Value{Type: resp.TypeArray, IsNull: true},
			expected: "*-1\r\n",
		},
		{
			name:     "Array Empty",
			input:    resp.Value{Type: resp.TypeArray, Array: []resp.Value{}},
			expected: "*0\r\n",
		},
		{
			name: "Mixed Array",
			input: resp.Value{
				Type: resp.TypeArray,
				Array: []resp.Value{
					{Type: resp.TypeInteger, Integer: 1},
					{Type: resp.TypeArray, Array: []resp.Value{
						{Type: resp.TypeSimpleString, String: []byte("inner")},
					}},
				},
			},
			expected: "*2\r\n:1\r\n*1\r\n+inner\r\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			enc := resp.NewEncoder(&buf)

			err := enc.Write(tt.input)
			if err != nil {
				t.Fatalf("Write() failed: %v", err)
			}

			err = enc.Flush()
			if err != nil {
				t.Fatalf("Flush() failed: %v", err)
			}

			if buf.String() != tt.expected {
				t.Errorf("Write() got = %q, want %q", buf.String(), tt.expected)
			}
		})
	}
}

func TestEncoder_WriteError(t *testing.T) {
	errWriter := &errorWriter{}
	enc := resp.NewEncoder(errWriter)

	val := resp.Value{Type: resp.TypeSimpleString, String: []byte("test")}

	err := enc.Write(val)
	if err != nil {
		t.Fatalf("Write() failed: %v", err)
	}

	err = enc.Flush()
	if err == nil {
		t.Error("Expected error from Flush(), but got nil")
	}
}

type errorWriter struct{}

func (e *errorWriter) Write(_ []byte) (n int, err error) {
	return 0, io.ErrClosedPipe
}
