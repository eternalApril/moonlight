package resp_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/eternalApril/moonlight/internal/resp"
)

func TestReadInt(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr error
	}{
		{
			name:    "Valid positive",
			input:   ":1000\r\n",
			want:    1000,
			wantErr: nil,
		},
		{
			name:    "Valid positive with +",
			input:   ":+1230\r\n",
			want:    1230,
			wantErr: nil,
		},
		{
			name:    "Valid negative",
			input:   ":-15\r\n",
			want:    -15,
			wantErr: nil,
		},
		{
			name:    "Valid zero",
			input:   ":0\r\n",
			want:    0,
			wantErr: nil,
		},
		{
			name:    "Invalid ending",
			input:   ":1000\n",
			want:    0,
			wantErr: resp.ErrInvalidEnding,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := resp.NewReader(strings.NewReader(tt.input))

			val, err := r.Read()

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Read() expected error %v, got nil", tt.wantErr)
				} else {
					return
				}
			}

			if err != nil {
				t.Errorf("Read() unexpected error %v", err)
			}

			if val.Type != resp.TypeInteger {
				t.Errorf("Read() type = %v, want %v", resp.TypeInteger, val.Type)
			}

			if val.Num != tt.want {
				t.Errorf("Read() num = %v, want %v", val.Num, tt.want)
			}
		})
	}
}
