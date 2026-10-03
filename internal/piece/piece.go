package piece

import (
	"crypto/sha1"
	"fmt"
	"sync"

	"github.com/rpsingh21/torrent-cli/internal/storage"
)

type Piece struct {
	Index  int
	Length int
	HashV1 [20]byte
	Blocks []Block

	buffer          []byte
	mu              sync.Mutex
	downloadedBlock int
}

func (p *Piece) NextMissingBlock() *Block {
	if p.Blocks == nil {
		p.Blocks = buildBlocks(p.Index, p.Length)
	}

	for i := range p.Blocks {
		block := &p.Blocks[i]
		if block.Completed || block.RequestedBy != "" {
			continue
		}

		return block
	}

	return nil
}

func buildBlocks(pieceId, pieceSize int) []Block {
	if pieceSize <= 0 {
		return nil
	}

	totalBlocks := (pieceSize + BLOCK_SIZE - 1) / BLOCK_SIZE
	blocks := make([]Block, totalBlocks)

	for i := range blocks {
		offset := i * BLOCK_SIZE
		blocks[i] = Block{Piece: pieceId, Offset: offset, Length: min(BLOCK_SIZE, pieceSize-offset)}
	}

	return blocks
}

func (p *Piece) blockAt(offset int) *Block {
	index := offset / BLOCK_SIZE
	if index < len(p.Blocks) {
		return &p.Blocks[index]
	}

	return nil
}

func (p *Piece) Completed() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	return len(p.Blocks) == p.downloadedBlock
}

func (p *Piece) completeBlock(peerId string, offset int, data []byte) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	block := p.blockAt(offset)
	if block == nil || block.RequestedBy != peerId || block.Completed || len(data) != block.Length {
		return false
	}

	if p.buffer == nil {
		p.buffer = make([]byte, 0, p.Length)
	}

	end := offset + len(data)
	if end > len(p.buffer) {
		p.buffer = p.buffer[:end]
	}
	copy(p.buffer[offset:end], data)

	block.Completed = true
	block.RequestedBy = ""
	p.downloadedBlock++

	return true
}

func (p *Piece) saveCompletePiece(store storage.Storage) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.Blocks == nil {
		return fmt.Errorf("piece %v does not have buffer data to write", p.Index)
	}

	if sha1.Sum(p.buffer) != p.HashV1 {
		return fmt.Errorf("piece %d failed SHA-1 verification", p.Index)
	}

	if err := store.WritePiece(p.Index, p.buffer); err != nil {
		return err
	}

	p.buffer = nil
	p.Blocks = nil

	return nil
}

func (p *Piece) resetBlock(peerId string, offset int) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	block := p.blockAt(offset)
	if block == nil || block.RequestedBy != peerId || block.Completed {
		return false
	}

	if block.Completed {
		p.downloadedBlock--
	}
	block.resetDownload()

	return true
}

func (p *Piece) resetAllBlock() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, block := range p.Blocks {
		block.resetDownload()
	}

	p.buffer = nil
	p.downloadedBlock = 0
}
