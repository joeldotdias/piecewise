package bencode_test

import (
	"piecewise/internal/bencode"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncode(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected []byte
		wantErr  bool
	}{
		{
			name:     "String",
			input:    "spam",
			expected: []byte("4:spam"),
			wantErr:  false,
		},
		{
			name:     "Integer (int64)",
			input:    int64(42),
			expected: []byte("i42e"),
			wantErr:  false,
		},
		{
			name:     "Negative Integer (int)",
			input:    -42,
			expected: []byte("i-42e"),
			wantErr:  false,
		},
		{
			name:     "List",
			input:    []any{"spam", "eggs"},
			expected: []byte("l4:spam4:eggse"),
			wantErr:  false,
		},
		{
			name: "Dictionary (Tests Sorting)",
			// Even though "spam" is defined first in Go, "cow" must encode first alphabetically
			input:    map[string]any{"spam": "eggs", "cow": "moo"},
			expected: []byte("d3:cow3:moo4:spam4:eggse"),
			wantErr:  false,
		},
		{
			name: "Nested Structure",
			input: map[string]any{
				"list": []any{"a", "b"},
				"num":  int64(123),
			},
			expected: []byte("d4:listl1:a1:be3:numi123ee"),
			wantErr:  false,
		},
		{
			name:     "Unsupported Type (Float)",
			input:    3.14,
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := bencode.Encode(tt.input)

			if tt.wantErr {
				require.Error(t, err, "expected an error for unsupported type")
				require.Nil(t, got)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, got, "encoded bytes did not match expected")
			}
		})
	}
}
