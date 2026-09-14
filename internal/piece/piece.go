package piece

import (
	"bytes"
	"crypto/sha1"
	"log"
)

type Piece struct {
	Index     int
	Length    int
	HashV1    [20]byte
	Blocks    []Block
	startedAt int64
}

func NewPiece(index int, length int, hashv1 [20]byte, blocks []Block) *Piece {
	return &Piece{
		Index:  index,
		Length: length,
		HashV1: hashv1,
		Blocks: blocks,
	}
}

func (p *Piece) NextMissingBlock() *Block {
	for i := range p.Blocks {
		if !p.Blocks[i].Requested && !p.Blocks[i].Completed {
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
		if !block.Completed {
			return false
		}
		h.Write(block.Data)
	}

	return bytes.Equal(h.Sum(nil), p.HashV1[:])
}

func (p *Piece) Completed() bool {
	for i := range p.Blocks {
		if !p.Blocks[i].Completed {
			return false
		}
	}
	return true
}

func (p *Piece) write() {
	log.Printf("Data write in storage %v", p.Index)
	// storage.write

	// Removed data from memory
	p.Blocks = nil
}
