package bencode

import (
	"testing"
)

func TestEncode(t *testing.T) {
	tests := []struct {
		name    string
		input   any
		want    string
		wantErr bool
	}{
		{
			name:  "bytes",
			input: []byte("hello"),
			want:  "5:hello",
		},
		{
			name:  "empty bytes",
			input: []byte(""),
			want:  "0:",
		},
		{
			name:  "int",
			input: 42,
			want:  "i42e",
		},
		{
			name:  "negative int",
			input: -42,
			want:  "i-42e",
		},
		{
			name:  "int8",
			input: int8(10),
			want:  "i10e",
		},
		{
			name:  "int16",
			input: int16(100),
			want:  "i100e",
		},
		{
			name:  "int32",
			input: int32(1000),
			want:  "i1000e",
		},
		{
			name:  "int64",
			input: int64(10000),
			want:  "i10000e",
		},
		{
			name:  "empty list",
			input: []any{},
			want:  "le",
		},
		{
			name: "list",
			input: []any{
				[]byte("spam"),
				42,
				[]byte("eggs"),
			},
			want: "l4:spami42e4:eggse",
		},
		{
			name:  "empty dictionary",
			input: map[string]any{},
			want:  "de",
		},
		{
			name: "dictionary",
			input: map[string]any{
				"foo": []byte("bar"),
				"num": 42,
			},
			want: "d3:foo3:bar3:numi42ee",
		},
		{
			name: "nested dictionary",
			input: map[string]any{
				"foo": map[string]any{
					"bar": []byte("baz"),
				},
			},
			want: "d3:food3:bar3:bazee",
		},
		{
			name: "nested list",
			input: []any{
				[]any{
					[]byte("foo"),
					[]byte("bar"),
				},
			},
			want: "ll3:foo3:baree",
		},
		{
			name:    "unsupported string",
			input:   "hello",
			wantErr: true,
		},
		{
			name:    "unsupported float",
			input:   1.5,
			wantErr: true,
		},
		{
			name: "nested unsupported value",
			input: map[string]any{
				"foo": "bar",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Encode(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if string(got) != tt.want {
				t.Errorf("Encode() = %q, want %q", got, tt.want)
			}
		})
	}
}
