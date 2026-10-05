package bitmaskreservoir

import (
	"errors"
	"math/bits"
)

type BitmaskReservoir struct {
	size   int
	words  []uint64
	cursor int
}

var ErrNoBitAvailable = errors.New("all bit are already reserved")
var ErrIndexOutOfRange = errors.New("block index out of range")

func NewBitmaskReservoir(size int) *BitmaskReservoir {
	if size <= 0 {
		return nil
	}

	numInt64Block := (size + 63) >> 6
	words := make([]uint64, numInt64Block)

	// Pre-fill unused trailing bits in the last bitMask with 1s
	// So trailingZeros64 will never select out-of-range indices.
	// size & 63 = size%64 for all size >= 0
	if rem := size & 63; rem != 0 {
		unusedBitsMask := ^uint64(0) << rem
		words[numInt64Block-1] = unusedBitsMask
	}

	return &BitmaskReservoir{
		size:   size,
		words:  words,
		cursor: 0,
	}
}

func (b *BitmaskReservoir) Reserve() (int, error) {
	for wordId := b.cursor; wordId < len(b.words); wordId++ {
		bitblock := b.words[wordId]
		if bitblock == ^uint64(0) {
			continue
		}

		bitId := bits.TrailingZeros64(^bitblock)
		index := (wordId << 6) + bitId

		if index >= b.size {
			break
		}

		b.words[wordId] |= (uint64(1) << bitId)

		if b.words[wordId] == ^uint64(0) {
			b.cursor = wordId + 1
		} else {
			b.cursor = wordId
		}

		return index, nil
	}

	b.cursor = len(b.words)
	return -1, ErrNoBitAvailable
}

func (b *BitmaskReservoir) Unreserve(index int) error {
	if uint(index) >= uint(b.size) {
		return ErrIndexOutOfRange
	}

	wordId := index >> 6
	bit := uint(index & 63) // or index % 64
	bitMask := uint64(1) << bit

	// Clear the bit (AND NOT) to make it 0(if 1)
	b.words[wordId] &^= bitMask

	if wordId < b.cursor {
		b.cursor = wordId
	}

	return nil
}

func (b *BitmaskReservoir) IsReserved(index int) bool {
	if uint(index) >= uint(b.size) {
		return false
	}

	wordId := index >> 6
	bit := uint(index & 63)

	return (b.words[wordId] & (uint64(1) << bit)) != 0
}

func (b *BitmaskReservoir) CanReserve() bool {
	return b.cursor < len(b.words)
}

func (b *BitmaskReservoir) CountReserved() int {
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
