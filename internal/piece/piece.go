package piece

import (
	"bytes"
	"crypto/sha1"
)

type Piece struct {
	Index  uint32
	Length uint32
	HashV1 [20]byte
	Blocks []Block
}

func NewPiece(index uint32, length uint32, hashv1 [20]byte, blocks []Block) *Piece {
	return &Piece{
		Index:  index,
		Length: length,
		HashV1: hashv1,
		Blocks: blocks,
	}
}

func (p *Piece) MissingBlock() *Block {
	for i := range p.Blocks {
		if !p.Blocks[i].requested && !p.Blocks[i].completed {
			return &p.Blocks[i]
		}
	}
	return nil
}

// func (p *Piece) Received(begin int, data byte)

func (p *Piece) Verify() bool {
	h := sha1.New()
	for i := range p.Blocks {
		block := &p.Blocks[i]
		if !block.completed {
			return false
		}
		h.Write(block.data)
	}

	return bytes.Equal(h.Sum(nil), p.HashV1[:])
}

func (p *Piece) Completed() bool {
	for i := range p.Blocks {
		if !p.Blocks[i].completed {
			return false
		}
	}
	return true
}
