package bitfield

import (
	"errors"
	"math/bits"
)

type Bitfield struct {
	size  int
	words []byte
}

var (
	ErrSizeCanNotNegative = errors.New("size cannot be negative")
	ErrInvalidByteLength  = errors.New("invalid byte slice length for given size")
)

func NewBitfield(size int) (*Bitfield, error) {
	if size < 0 {
		return nil, ErrSizeCanNotNegative
	}

	return &Bitfield{
		words: make([]byte, (size+7)>>3),
		size:  size,
	}, nil
}

func NewBitfieldFromBytes(words []byte, size int) (*Bitfield, error) {
	if size < 0 {
		return nil, ErrSizeCanNotNegative
	}

	if (size+7)>>3 != len(words) {
		return nil, ErrInvalidByteLength
	}

	return &Bitfield{
		words: words,
		size:  size,
	}, nil
}

func (b *Bitfield) Size() int {
	return b.size
}

func (b *Bitfield) Have(index int) bool {
	if uint(index) >= uint(b.size) {
		return false
	}

	return b.words[index>>3]&(1<<uint(7-(index&7))) != 0
}

func (b *Bitfield) SetIndex(index int) bool {
	if uint(index) >= uint(b.size) {
		return false
	}

	b.words[index>>3] |= byte(1 << (7 - (index & 7)))
	return true
}

func (b *Bitfield) ClearIndex(index int) bool {
	if uint(index) >= uint(b.size) {
		return false
	}

	b.words[index>>3] &^= 1 << uint(7-(index&7))
	return true
}

func (b *Bitfield) AllSet() bool {
	if b.size == 0 {
		return true
	}

	fullBytes := b.size >> 3

	for _, v := range b.words[:fullBytes] {
		if v != ^uint8(0) {
			return false
		}
	}

	remaining := b.size & 7
	if remaining == 0 {
		return true
	}

	mask := byte(^uint8(0) << uint(8-remaining))
	return b.words[fullBytes]&mask == mask
}

func (b *Bitfield) Count() int {
	var total int
	fullWords := b.size >> 3

	for _, v := range b.words[:fullWords] {
		total += bits.OnesCount8(v)
	}

	if remaining := b.size & 7; remaining > 0 {
		mask := (uint8(1) << remaining) - 1
		total += bits.OnesCount8(b.words[fullWords] & mask)
	}

	return total
}
