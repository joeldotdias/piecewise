package bencode_test

import (
	"piecewise/internal/bencode"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected any
		wantErr  bool
	}{
		{
			name:     "string",
			input:    "4:spam",
			expected: "spam",
			wantErr:  false,
		},
		{
			name:     "integer",
			input:    "i42e",
			expected: int64(42),
			wantErr:  false,
		},
		{
			name:     "negative_integer",
			input:    "i-42e",
			expected: int64(-42),
			wantErr:  false,
		},
		{
			name:     "List",
			input:    "l4:spam4:eggse",
			expected: []any{"spam", "eggs"},
			wantErr:  false,
		},
		{
			name:     "dictionary",
			input:    "d3:cow3:moo4:spam4:eggse",
			expected: map[string]any{"cow": "moo", "spam": "eggs"},
			wantErr:  false,
		},
		{
			name:     "nested types",
			input:    "d4:spaml1:a1:bee",
			expected: map[string]any{"spam": []any{"a", "b"}},
			wantErr:  false,
		},
		{
			name:     "malformed_integer",
			input:    "i42",
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := bencode.Decode(strings.NewReader(tt.input))

			if tt.wantErr {
				require.Error(t, err, "expected an error but got none")
				require.Nil(t, got)
			} else {
				require.NoError(t, err, "whoops we didn't want an error here")
				require.Equal(t, tt.expected, got, "decoded value doesn't match expected")
			}
		})
	}
}
