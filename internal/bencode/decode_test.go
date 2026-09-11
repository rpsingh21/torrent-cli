package bencode

import (
	"reflect"
	"testing"
)

func TestDecoderDecode(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  any
	}{
		{
			name:  "integer",
			input: "i42e",
			want:  int64(42),
		},
		{
			name:  "negative integer",
			input: "i-42e",
			want:  int64(-42),
		},
		{
			name:  "zero integer",
			input: "i0e",
			want:  int64(0),
		},
		{
			name:  "string",
			input: "4:spam",
			want:  []byte("spam"),
		},
		{
			name:  "empty string",
			input: "0:",
			want:  []byte{},
		},
		{
			name:  "empty list",
			input: "le",
			want:  []any{},
		},
		{
			name:  "list",
			input: "l4:spam4:eggse",
			want: []any{
				[]byte("spam"),
				[]byte("eggs"),
			},
		},
		{
			name:  "nested list",
			input: "lli1ei2eee",
			want: []any{
				[]any{
					int64(1),
					int64(2),
				},
			},
		},
		{
			name:  "empty dictionary",
			input: "de",
			want:  map[string]any{},
		},
		{
			name:  "dictionary",
			input: "d3:cow3:moo4:spam4:eggse",
			want: map[string]any{
				"cow":  []byte("moo"),
				"spam": []byte("eggs"),
			},
		},
		{
			name:  "nested dictionary",
			input: "d4:spamd3:foo3:baree",
			want: map[string]any{
				"spam": map[string]any{
					"foo": []byte("bar"),
				},
			},
		},
		{
			name:  "mixed nested values",
			input: "d4:listli1e4:teste3:numi42ee",
			want: map[string]any{
				"list": []any{
					int64(1),
					[]byte("test"),
				},
				"num": int64(42),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoder := NewDecoder([]byte(tt.input))

			got, err := decoder.Decode()
			if err != nil {
				t.Fatalf("Decode() unexpected error: %v", err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Decode() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestDecoderDecodeErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "empty input",
			input: "",
		},
		{
			name:  "invalid token",
			input: "x",
		},
		{
			name:  "unterminated integer",
			input: "i42",
		},
		{
			name:  "invalid integer",
			input: "iabc e",
		},
		{
			name:  "unterminated list",
			input: "l4:spam",
		},
		{
			name:  "unterminated dictionary",
			input: "d3:key5:value",
		},
		{
			name:  "missing string separator",
			input: "4spam",
		},
		{
			name:  "invalid string length",
			input: "x:spam",
		},
		{
			name:  "string exceeds input",
			input: "10:spam",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoder := NewDecoder([]byte(tt.input))

			_, err := decoder.Decode()
			if err == nil {
				t.Fatal("Decode() expected error, got nil")
			}
		})
	}
}
