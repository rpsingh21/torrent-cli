package peer

type Bitfield []byte

func (bf Bitfield) Has(index int) bool {
	if index < 0 || index >= len(bf) {
		return false
	}
	byteIndex, offset := index/8, index%8
	return bf[byteIndex]>>uint(7-offset)&1 != 0
}

func (bf Bitfield) SetIndex(index int) {
	if index < 0 || index > len(bf) {
		return
	}
	byteIndex, offset := index/8, index%8
	bf[byteIndex] |= 1 << uint(7-offset)
}
