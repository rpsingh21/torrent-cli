package bitfield

import (
	"encoding/binary"
	"errors"
	"math/bits"
)

type Bitfield struct {
	size  int
	words []uint64
}

var (
	ErrSizeCanNotNegative = errors.New("size cannot be negative")
	ErrInvalidByteLength  = errors.New("invalid byte slice length for given size")
)

func NewBitfield(size int) (*Bitfield, error) {
	if size < 0 {
		return nil, ErrSizeCanNotNegative
	}

	wordCount := int((uint(size) + 63) >> 6)
	return &Bitfield{
		size:  size,
		words: make([]uint64, wordCount),
	}, nil
}

func NewBitfieldFromBytes(data []byte, size int) (*Bitfield, error) {
	if size < 0 {
		return nil, ErrSizeCanNotNegative
	}

	expectedBytes := int((uint(size) + 7) >> 3)
	if len(data) != expectedBytes {
		return nil, ErrInvalidByteLength
	}

	wordCount := int((uint(size) + 63) >> 6)
	words := make([]uint64, wordCount)

	// Copy full 8-byte chunks (len(data) >> 3)
	fullWords := len(data) >> 3
	for i := range fullWords {
		words[i] = binary.LittleEndian.Uint64(data[i<<3 : (i+1)<<3])
	}

	// Handle remaining trailing bytes (1 to 7 bytes)
	if rem := len(data) & 7; rem > 0 {
		var lastWord uint64
		offset := fullWords << 3
		for i := range rem {
			lastWord |= uint64(data[offset+i]) << (i << 3)
		}
		words[fullWords] = lastWord
	}

	return &Bitfield{
		size:  size,
		words: words,
	}, nil
}

func (b *Bitfield) Size() int {
	return b.size
}

func (b *Bitfield) Have(index int) bool {
	if uint(index) >= uint(b.size) {
		return false
	}

	wordId := index >> 6
	bitMask := uint64(1) << (index & 63) // equal index%64

	return b.words[wordId]&bitMask != 0
}

func (b *Bitfield) SetIndex(index int) bool {
	if uint(index) >= uint(b.size) {
		return false
	}

	wordId := index >> 6
	bitMask := uint64(1) << (index & 63)

	b.words[wordId] |= bitMask
	return true
}

func (b *Bitfield) ClearIndex(index int) bool {
	if uint(index) >= uint(b.size) {
		return false
	}

	wordId := index >> 6
	bitMask := uint64(1) << (index & 63)

	b.words[wordId] &^= bitMask
	return true
}

func (b *Bitfield) AllSet() bool {
	if b.size == 0 {
		return true
	}

	fullWords := b.size >> 6

	for _, v := range b.words[:fullWords] {
		if v != ^uint64(0) {
			return false
		}
	}

	remaining := b.size & 63
	if remaining == 0 {
		return true
	}

	mask := (uint64(1) << remaining) - 1
	return (b.words[fullWords] & mask) == mask
}

func (b *Bitfield) Count() int {
	var total int
	fullWords := b.size >> 6

	for _, v := range b.words[:fullWords] {
		total += bits.OnesCount64(v)
	}

	if remaining := b.size & 63; remaining > 0 {
		mask := (uint64(1) << remaining) - 1
		total += bits.OnesCount64(b.words[fullWords] & mask)
	}

	return total
}
