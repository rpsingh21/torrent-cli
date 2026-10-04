package bitmaskreservoir

import (
	"errors"
	"math/bits"
)

type BitmaskReservoir struct {
	size      int
	bitBlocks []uint64
	cursor    int
}

var ErrNoBlocksAvailable = errors.New("all piece blocks are already reserved")

func NewBitmaskReservoir(size int) *BitmaskReservoir {
	if size <= 0 {
		return nil
	}

	numInt64Block := (size + 63) >> 6
	bitBlocks := make([]uint64, numInt64Block)

	// Pre-fill unused trailing bits in the last bitMask with 1s
	// So trailingZeros64 will never select out-of-range indices.
	if rem := size % 64; rem != 0 {
		unusedBitsMask := ^uint64(0) << rem
		bitBlocks[numInt64Block-1] = unusedBitsMask
	}

	return &BitmaskReservoir{
		size:      size,
		bitBlocks: bitBlocks,
		cursor:    0,
	}
}

func (b *BitmaskReservoir) Reserve() (int, error) {
	for blockId := b.cursor; blockId < len(b.bitBlocks); blockId++ {
		bitblock := b.bitBlocks[blockId]
		if bitblock == ^uint64(0) {
			continue
		}

		bitId := bits.TrailingZeros64(^bitblock)
		index := (blockId << 6) + bitId

		if index >= b.size {
			break
		}

		b.bitBlocks[blockId] |= (uint64(1) << bitId)

		if b.bitBlocks[blockId] == ^uint64(0) {
			b.cursor = blockId + 1
		} else {
			b.cursor = blockId
		}

		return index, nil
	}

	b.cursor = len(b.bitBlocks)
	return -1, ErrNoBlocksAvailable
}

func (b *BitmaskReservoir) Unreserve(index int) error {
	if index < 0 || index >= b.size {
		return errors.New("block index out of bounds")
	}

	blockId := index >> 6
	bit := uint(index % 64)
	bitMask := uint64(1) << bit

	// Clear the bit (AND NOT) to make it 0(if 1)
	b.bitBlocks[blockId] &^= bitMask

	if blockId < b.cursor {
		b.cursor = blockId
	}

	return nil
}

func (b *BitmaskReservoir) IsReserved(index int) bool {
	if index < 0 || index >= b.size {
		return false
	}

	blockId := index >> 6
	bit := uint(index % 64)

	return (b.bitBlocks[blockId] & (uint64(1) << bit)) != 0
}
