package resp_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/eternalApril/moonlight/internal/resp"
)

func runTest(t *testing.T, name string, input string, want resp.Value, wantErr error) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		r := resp.NewDecoder(strings.NewReader(input))
		got, err := r.Read()

		if wantErr != nil {
			if !errors.Is(err, wantErr) {
				t.Fatalf("expected error %v, got %v", wantErr, err)
			}
			return
		}

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got.Type != want.Type {
			t.Errorf("got type %q, want %q", got.Type, want.Type)
		}

		if got.Num != want.Num {
			t.Errorf("got num %v, want %v", got.Num, want.Num)
		}

		if !reflect.DeepEqual(got.String, want.String) {
			t.Errorf("got string %q, want %q", got.String, want.String)
		}

		if !reflect.DeepEqual(got.Array, want.Array) {
			t.Errorf("got array %v, want %v", got.Array, want.Array)
		}
	})
}

func TestDecoder_ReadInt(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    resp.Value
		wantErr error
	}{
		{"Valid positive", ":1000\r\n", resp.Value{Type: resp.TypeInteger, Num: 1000}, nil},
		{"Valid negative", ":-15\r\n", resp.Value{Type: resp.TypeInteger, Num: -15}, nil},
		{"Invalid ending", ":1000\n", resp.Value{}, resp.ErrInvalidEnding},
	}

	for _, tt := range tests {
		runTest(t, tt.name, tt.input, tt.want, tt.wantErr)
	}
}

func TestDecoder_ReadSimpleString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    resp.Value
		wantErr error
	}{
		{"Valid string", "+OK\r\n", resp.Value{Type: resp.TypeSimpleString, String: []byte("OK")}, nil},
		{"Invalid ending", "+OK\n", resp.Value{}, resp.ErrInvalidEnding},
	}

	for _, tt := range tests {
		runTest(t, tt.name, tt.input, tt.want, tt.wantErr)
	}
}

func TestDecoder_ReadBulkString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    resp.Value
		wantErr error
	}{
		{name: "Valid Bulk String", input: "$6\r\nfoobar\r\n", want: resp.Value{Type: resp.TypeBulkString, String: []byte("foobar")}},
		{name: "Empty Bulk String", input: "$0\r\n\r\n", want: resp.Value{Type: resp.TypeBulkString, String: []byte("")}},
		{name: "Nil Bulk String", input: "$-1\r\n", want: resp.Value{Type: resp.TypeBulkString, String: nil, IsNull: true}},
		{name: "Invalid length", input: "$abc\r\n", wantErr: resp.ErrInvalidEnding},
		{name: "Mismatched length", input: "$10\r\nshort\r\n", wantErr: resp.ErrInvalidEnding},
		{name: "Missing trailing CRLF", input: "$6\r\nfoobar", wantErr: resp.ErrInvalidEnding},
		{name: "Unexpected EOF in header", input: "$6", wantErr: resp.ErrInvalidEnding},
	}

	for _, tt := range tests {
		runTest(t, tt.name, tt.input, tt.want, tt.wantErr)
	}
}

func TestDecoder_ReadError(t *testing.T) {
	runTest(t, "Basic Error", "-Err msg\r\n", resp.Value{Type: resp.TypeError, String: []byte("Err msg")}, nil)
}

func TestDecoder_ReadArray(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    resp.Value
		wantErr error
	}{
		{
			name:  "Valid array 1",
			input: "*2\r\n$5\r\nhello\r\n$5\r\nworld\r\n",
			want: resp.Value{Type: resp.TypeArray, Array: []resp.Value{
				{Type: resp.TypeBulkString, String: []byte("hello")},
				{Type: resp.TypeBulkString, String: []byte("world")},
			}},
		},
		{
			name:  "Empty array",
			input: "*0\r\n",
			want: resp.Value{
				Type:  resp.TypeArray,
				Array: []resp.Value{},
			},
		},
		{
			name:  "Nil array",
			input: "*-1\r\n",
			want: resp.Value{
				Type:   resp.TypeArray,
				Array:  nil,
				IsNull: true,
			},
		},
		{
			name:  "Mixed types array",
			input: "*3\r\n:1\r\n+OK\r\n$4\r\ntest\r\n",
			want: resp.Value{
				Type: resp.TypeArray,
				Array: []resp.Value{
					{Type: resp.TypeInteger, Num: 1},
					{Type: resp.TypeSimpleString, String: []byte("OK")},
					{Type: resp.TypeBulkString, String: []byte("test")},
				},
			},
		},
		{
			name:  "Nested array",
			input: "*2\r\n*1\r\n:5\r\n+inner\r\n",
			want: resp.Value{
				Type: resp.TypeArray,
				Array: []resp.Value{
					{
						Type: resp.TypeArray,
						Array: []resp.Value{
							{Type: resp.TypeInteger, Num: 5},
						},
					},
					{Type: resp.TypeSimpleString, String: []byte("inner")},
				},
			},
		},
		{
			name:    "Invalid array length",
			input:   "*abc\r\n",
			wantErr: resp.ErrInvalidEnding,
		},
		{
			name:    "Mismatched elements count",
			input:   "*3\r\n:1\r\n:2\r\n",
			wantErr: resp.ErrInvalidEnding,
		},
		{
			name:    "Corrupt element in array",
			input:   "*1\r\n+MissingCR\n",
			wantErr: resp.ErrInvalidEnding,
		},
	}

	for _, tt := range tests {
		runTest(t, tt.name, tt.input, tt.want, tt.wantErr)
	}
}
