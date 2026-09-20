package piece

import "crypto/sha1"

type Piece struct {
	Index     int
	Length    int
	HashV1    [20]byte
	Blocks    []Block
	Verifying bool
}

func (p *Piece) NextMissingBlock() *Block {
	for i := range p.Blocks {
		block := &p.Blocks[i]
		if block.Completed || block.Requested {
			continue
		}

		return block
	}

	return nil
}

func (p *Piece) blockAt(offset int) *Block {
	for i := range p.Blocks {
		if p.Blocks[i].Offset == offset {
			return &p.Blocks[i]
		}
	}

	return nil
}

func (p *Piece) Completed() bool {
	if p.Verifying || len(p.Blocks) == 0 {
		return false
	}
	for i := range p.Blocks {
		if !p.Blocks[i].Completed {
			return false
		}
	}
	return true
}

func (p *Piece) Verify() bool {
	if len(p.Blocks) == 0 {
		return false
	}

	data := make([]byte, 0, p.Length)
	for i := range p.Blocks {
		if !p.Blocks[i].Completed || len(p.Blocks[i].Data) != p.Blocks[i].Length {
			return false
		}
		data = append(data, p.Blocks[i].Data...)
	}
	return sha1.Sum(data) == p.HashV1
}
