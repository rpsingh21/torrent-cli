package bitfield

type Bitfield struct {
	bits []byte
	size int
}

func NewBitfield(size int) *Bitfield {
	return &Bitfield{
		bits: make([]byte, (size+7)/8),
		size: size,
	}
}

func (bf Bitfield) Have(index int) bool {
	if index < 0 || index >= bf.size {
		return false
	}

	byteIndex, offset := index>>3, index&7
	return bf.bits[byteIndex]>>(7-uint(offset))&1 != 0
}

func (bf Bitfield) SetIndex(index int) {
	if index < 0 || index >= bf.size {
		return
	}

	byteIndex, offset := index>>3, index&7
	bf.bits[byteIndex] |= 1 << (7 - uint(offset))
}

func (bf Bitfield) AllSet() bool {
	if bf.size == 0 {
		return true
	}

	fullBytes := bf.size / 8
	for _, v := range bf.bits[:fullBytes] {
		if v != 0xff {
			return false
		}
	}

	remaining := bf.size % 8
	if remaining == 0 {
		return true
	}

	// Valid bits are the most-significant `remaining` bits.
	mask := byte(0xff << (8 - remaining))
	return bf.bits[fullBytes]&mask == mask
}

func (bf *Bitfield) ClearIndex(index int) {
	if index < 0 || index >= bf.size {
		return
	}

	byteIndex := index / 8
	bitIndex := uint(7 - (index % 8))

	bf.bits[byteIndex] &^= 1 << bitIndex
}
