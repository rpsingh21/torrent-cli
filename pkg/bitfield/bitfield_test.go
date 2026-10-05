package bitfield

import (
	"errors"
	"math"
	"testing"
)

func TestNewBitfield(t *testing.T) {
	tests := []struct {
		name          string
		size          int
		expectedWords int
		expectErr     error
	}{
		{name: "negative size", size: -1, expectedWords: 0, expectErr: ErrSizeCanNotNegative},
		{name: "zero size", size: 0, expectedWords: 0, expectErr: nil},
		{name: "single bit", size: 1, expectedWords: 1, expectErr: nil},
		{name: "boundary 63 bits", size: 63, expectedWords: 1, expectErr: nil},
		{name: "exact 64 bits", size: 64, expectedWords: 1, expectErr: nil},
		{name: "boundary 65 bits", size: 65, expectedWords: 2, expectErr: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bf, err := NewBitfield(tc.size)
			if tc.expectErr != nil {
				if !errors.Is(err, tc.expectErr) {
					t.Fatalf("expected error %v, got %v", tc.expectErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if bf.Size() != tc.size {
				t.Errorf("expected size %d, got %d", tc.size, bf.Size())
			}
			if len(bf.words) != tc.expectedWords {
				t.Errorf("expected %d words, got %d", tc.expectedWords, len(bf.words))
			}
		})
	}
}

func TestNewBitfieldFromBytes(t *testing.T) {
	tests := []struct {
		name      string
		data      []byte
		size      int
		expectErr error
	}{
		{
			name:      "negative size",
			data:      []byte{0x01},
			size:      -1,
			expectErr: ErrSizeCanNotNegative,
		},
		{
			name:      "zero size with empty slice",
			data:      []byte{},
			size:      0,
			expectErr: nil,
		},
		{
			name:      "zero size with non-empty slice",
			data:      []byte{0x01},
			size:      0,
			expectErr: ErrInvalidByteLength,
		},
		{
			name:      "sub-byte size (3 bits, requires 1 byte)",
			data:      []byte{0x05}, // bits 0 and 2 set
			size:      3,
			expectErr: nil,
		},
		{
			name:      "too few bytes for size (8 bits requires 1 byte, got 0)",
			data:      []byte{},
			size:      8,
			expectErr: ErrInvalidByteLength,
		},
		{
			name:      "too many bytes for size (8 bits requires 1 byte, got 2)",
			data:      []byte{0x00, 0x00},
			size:      8,
			expectErr: ErrInvalidByteLength,
		},
		{
			name:      "multi-byte non-word-boundary (10 bits requires 2 bytes)",
			data:      []byte{0xFF, 0x03},
			size:      10,
			expectErr: nil,
		},
		{
			name:      "exact 64 bits (requires 8 bytes)",
			data:      make([]byte, 8),
			size:      64,
			expectErr: nil,
		},
		{
			name:      "boundary 65 bits (requires 9 bytes)",
			data:      make([]byte, 9),
			size:      65,
			expectErr: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bf, err := NewBitfieldFromBytes(tc.data, tc.size)
			if tc.expectErr != nil {
				if !errors.Is(err, tc.expectErr) {
					t.Fatalf("expected error %v, got %v", tc.expectErr, err)
				}
				if bf != nil {
					t.Fatal("expected nil Bitfield on error")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if bf.Size() != tc.size {
				t.Errorf("expected size %d, got %d", tc.size, bf.Size())
			}
		})
	}
}

func TestNewBitfieldFromBytes_ValueIntegrity(t *testing.T) {
	// byte 0 = 0b00000101 (bits 0, 2)
	// byte 1 = 0b00000010 (bit 9 -> index 1 of byte 1)
	// byte 8 = 0b00000001 (bit 64 -> index 0 of byte 8, crosses into word 1)
	data := make([]byte, 9)
	data[0] = 0b00000101
	data[1] = 0b00000010
	data[8] = 0b00000001

	bf, err := NewBitfieldFromBytes(data, 65)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedSet := []int{0, 2, 9, 64}
	for _, idx := range expectedSet {
		if !bf.Have(idx) {
			t.Errorf("expected bit %d to be set", idx)
		}
	}

	expectedUnset := []int{1, 3, 8, 10, 63}
	for _, idx := range expectedUnset {
		if bf.Have(idx) {
			t.Errorf("expected bit %d to be unset", idx)
		}
	}
}

func TestSetHaveAndClear(t *testing.T) {
	const size = 130
	bf, _ := NewBitfield(size)

	// Out of bounds
	oob := []int{-1, -50, size, size + 1, math.MaxInt}
	for _, idx := range oob {
		if bf.Have(idx) {
			t.Errorf("Have(%d) should be false", idx)
		}
		if bf.SetIndex(idx) {
			t.Errorf("SetIndex(%d) should return false", idx)
		}
		if bf.ClearIndex(idx) {
			t.Errorf("ClearIndex(%d) should return false", idx)
		}
	}

	// Cross-word boundaries
	indices := []int{0, 63, 64, 65, 127, 128, 129}
	for _, idx := range indices {
		if !bf.SetIndex(idx) {
			t.Fatalf("SetIndex(%d) failed", idx)
		}
		if !bf.Have(idx) {
			t.Errorf("Have(%d) should be true", idx)
		}
	}

	for _, idx := range indices {
		if !bf.ClearIndex(idx) {
			t.Fatalf("ClearIndex(%d) failed", idx)
		}
		if bf.Have(idx) {
			t.Errorf("Have(%d) should be false after clear", idx)
		}
	}
}

func TestAllSet(t *testing.T) {
	t.Run("empty bitfield", func(t *testing.T) {
		bf, _ := NewBitfield(0)
		if !bf.AllSet() {
			t.Error("expected AllSet() to be true for size 0")
		}
	})

	t.Run("exact word boundary 64", func(t *testing.T) {
		bf, _ := NewBitfield(64)
		if bf.AllSet() {
			t.Error("expected AllSet() to be false when empty")
		}
		for i := range 64 {
			bf.SetIndex(i)
		}
		if !bf.AllSet() {
			t.Error("expected AllSet() to be true when all 64 bits are set")
		}
		bf.ClearIndex(63)
		if bf.AllSet() {
			t.Error("expected AllSet() to be false when 1 bit is cleared")
		}
	})

	t.Run("non-multiple of 64", func(t *testing.T) {
		const size = 70
		bf, _ := NewBitfield(size)

		for i := range size {
			bf.SetIndex(i)
		}
		if !bf.AllSet() {
			t.Error("expected AllSet() to be true when all 70 bits are set")
		}

		// Bits above size in the last word must not invalidate AllSet
		// if they remain 0, or if they get modified outside the bounds
		bf.ClearIndex(69)
		if bf.AllSet() {
			t.Error("expected AllSet() to be false when bit 69 is cleared")
		}
	})

	t.Run("bits set beyond size in the same word", func(t *testing.T) {
		// Verify AllSet ignores unused bits in the last uint64 word
		bf, _ := NewBitfield(3)
		bf.SetIndex(0)
		bf.SetIndex(1)
		bf.SetIndex(2)
		// Manually tamper higher unused bits in the word
		bf.words[0] |= (uint64(1) << 10)

		if !bf.AllSet() {
			t.Error("AllSet should ignore bits beyond size in the last word")
		}
	})
}

func TestCount(t *testing.T) {
	t.Run("empty bitfield", func(t *testing.T) {
		bf, _ := NewBitfield(0)
		if bf.Count() != 0 {
			t.Errorf("expected count 0, got %d", bf.Count())
		}
	})

	t.Run("arbitrary bits count", func(t *testing.T) {
		bf, _ := NewBitfield(150)
		indices := []int{0, 10, 63, 64, 100, 149}

		for _, idx := range indices {
			bf.SetIndex(idx)
		}

		if count := bf.Count(); count != len(indices) {
			t.Errorf("expected count %d, got %d", len(indices), count)
		}

		bf.ClearIndex(10)
		if count := bf.Count(); count != len(indices)-1 {
			t.Errorf("expected count %d, got %d", len(indices)-1, count)
		}
	})

	t.Run("ignores unmanaged bits in last word", func(t *testing.T) {
		bf, _ := NewBitfield(5)
		bf.SetIndex(0)
		bf.SetIndex(4)
		// Tamper an unused bit beyond size
		bf.words[0] |= (uint64(1) << 30)

		if count := bf.Count(); count != 2 {
			t.Errorf("expected count to be 2 ignoring out-of-range bits, got %d", count)
		}
	})
}
