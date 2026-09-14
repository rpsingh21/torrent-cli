package bitfield

import (
	"math/rand/v2"
	"testing"
)

func getSetBit(size int, b *testing.B) map[int]any {
	b.Helper()
	setIdx := make(map[int]any, 10000)
	for range 10000 {
		n := rand.IntN(size) // [0, 100000]
		setIdx[n] = struct{}{}
	}
	return setIdx
}

func BenchmarkBitField(b *testing.B) {
	size := 1000_00
	setBit := getSetBit(size, b)

	b.ReportAllocs()
	b.ReportMetric(float64(len(setBit)), "set_ops")

	for b.Loop() {
		bitfield := NewBitfield(size)
		for i := range setBit {
			bitfield.SetIndex(i)
		}
		for range 1000 {
			n := rand.IntN(size)
			have := bitfield.Have(n)
			_, ok := setBit[n]

			if have != ok {
				b.Fatalf("Value in bitfiled: %v, in set %v", have, ok)
			}
		}
	}

}

func BenchmarkBitfieldSet(b *testing.B) {
	size := 100_000
	bf := NewBitfield(size)

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		for i := range size {
			bf.SetIndex(i)
		}
	}
}

func BenchmarkBitfieldHave(b *testing.B) {
	size := 100_000
	bf := NewBitfield(size)

	for i := range size {
		bf.SetIndex(i)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		for i := range size {
			_ = bf.Have(i)
		}
	}
}

func TestBitfieldAllSet(t *testing.T) {
	tests := []struct {
		name     string
		size     int
		set      []int
		expected bool
	}{
		{
			name:     "empty bitfield",
			size:     0,
			expected: true,
		},
		{
			name:     "no bits set",
			size:     8,
			expected: false,
		},
		{
			name:     "partially set",
			size:     8,
			set:      []int{0, 1, 2, 3},
			expected: false,
		},
		{
			name:     "all bits set",
			size:     8,
			set:      []int{0, 1, 2, 3, 4, 5, 6, 7},
			expected: true,
		},
		{
			name:     "multiple bytes all set",
			size:     16,
			set:      []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
			expected: true,
		},
		{
			name:     "multiple bytes partially set",
			size:     16,
			set:      []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14},
			expected: false,
		},
		{
			name:     "non byte aligned all set",
			size:     10,
			set:      []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
			expected: true,
		},
		{
			name:     "non byte aligned partially set",
			size:     10,
			set:      []int{0, 1, 2, 3, 4, 5, 6, 7, 8},
			expected: false,
		},
		{
			name:     "single bit all set",
			size:     1,
			set:      []int{0},
			expected: true,
		},
		{
			name:     "single bit not set",
			size:     1,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bf := NewBitfield(tt.size)

			for _, index := range tt.set {
				bf.SetIndex(index)
			}

			if got := bf.AllSet(); got != tt.expected {
				t.Errorf("AllSet() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestBitfieldSetAndHave(t *testing.T) {
	bf := NewBitfield(10)

	// Initially all bits should be unset.
	for i := range 10 {
		if bf.Have(i) {
			t.Errorf("Have(%d) = true, want false", i)
		}
	}

	// Set a few bits.
	for _, index := range []int{0, 3, 7, 9} {
		bf.SetIndex(index)
	}

	for i := range 10 {
		want := i == 0 || i == 3 || i == 7 || i == 9
		if got := bf.Have(i); got != want {
			t.Errorf("Have(%d) = %v, want %v", i, got, want)
		}
	}
}

func TestBitfieldSetIndexBounds(t *testing.T) {
	bf := NewBitfield(8)

	bf.SetIndex(-1)
	bf.SetIndex(8)

	if bf.Have(-1) {
		t.Error("Have(-1) = true, want false")
	}

	if bf.Have(8) {
		t.Error("Have(8) = true, want false")
	}
}
