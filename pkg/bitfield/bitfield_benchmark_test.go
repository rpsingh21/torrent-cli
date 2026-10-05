package bitfield

import (
	"crypto/rand"
	"testing"
)

func BenchmarkNewBitfieldFromBytes(b *testing.B) {
	const bitSize = 1000000
	byteSize := (bitSize + 7) >> 3
	data := make([]byte, byteSize)
	_, _ = rand.Read(data)

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		_, _ = NewBitfieldFromBytes(data, bitSize)
	}
}

func BenchmarkSetIndex(b *testing.B) {
	bf, _ := NewBitfield(1000000)
	target := 500000

	b.ResetTimer()
	for b.Loop() {
		bf.SetIndex(target)
	}
}

func BenchmarkHave(b *testing.B) {
	bf, _ := NewBitfield(1000000)
	bf.SetIndex(500000)
	target := 500000

	b.ResetTimer()
	for b.Loop() {
		_ = bf.Have(target)
	}
}

func BenchmarkClearIndex(b *testing.B) {
	bf, _ := NewBitfield(1000000)
	target := 500000

	b.ResetTimer()
	for b.Loop() {
		bf.ClearIndex(target)
	}
}

func BenchmarkAllSet(b *testing.B) {
	const size = 100000
	bf, _ := NewBitfield(size)
	for i := range size {
		bf.SetIndex(i)
	}

	b.ResetTimer()
	for b.Loop() {
		_ = bf.AllSet()
	}
}

func BenchmarkCount(b *testing.B) {
	const size = 100000
	bf, _ := NewBitfield(size)
	for i := 0; i < size; i += 2 {
		bf.SetIndex(i)
	}

	b.ResetTimer()
	for b.Loop() {
		_ = bf.Count()
	}
}
