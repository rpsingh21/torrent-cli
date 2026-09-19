package bitfield

type Bitfield struct {
	bits []byte
	size int
}

func NewBitfield(size int) *Bitfield {
	return &Bitfield{
		bits: make([]byte, (size+7)>>3),
		size: size,
	}
}

func NewBitfieldFromBytes(bits []byte, size int) *Bitfield {
	if (size+7)/8 != len(bits) {
		return nil // or return an error
	}

	return &Bitfield{
		bits: bits,
		size: size,
	}
}

func (b *Bitfield) Have(index int) bool {
	if uint(index) >= uint(b.size) {
		return false
	}

	return b.bits[index>>3]&(1<<uint(7-(index&7))) != 0
}

func (b *Bitfield) SetIndex(index int) {
	if uint(index) >= uint(b.size) {
		return
	}

	b.bits[index>>3] |= byte(1 << (7 - (index & 7)))
}

func (b *Bitfield) ClearIndex(index int) {
	if uint(index) >= uint(b.size) {
		return
	}

	b.bits[index>>3] &^= 1 << uint(7-(index&7))
}

func (b *Bitfield) AllSet() bool {
	if b.size == 0 {
		return true
	}

	fullBytes := b.size >> 3

	for _, v := range b.bits[:fullBytes] {
		if v != 0xff {
			return false
		}
	}

	remaining := b.size & 7
	if remaining == 0 {
		return true
	}

	mask := byte(0xff << uint(8-remaining))
	return b.bits[fullBytes]&mask == mask
}
