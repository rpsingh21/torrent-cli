package piece

import (
	"crypto/sha1"
	"fmt"
	"sync"
	"time"

	"github.com/rpsingh21/torrent-cli/internal/storage"
)

type Piece struct {
	Index  int
	Offset int
	Length int
	HashV1 [20]byte
	Blocks []Block

	mu              sync.Mutex
	toatalBlock     int
	downloadedBlock int
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
	index := offset / BLOCK_SIZE
	if index < p.toatalBlock {
		return &p.Blocks[index]
	}

	return nil
}

func (p *Piece) Completed() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.toatalBlock == p.downloadedBlock
}

func (p *Piece) completeBlock(peerId string, offset int, data []byte) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	block := p.blockAt(offset)
	if block == nil || !block.Requested || block.RequestedBy != peerId || block.Completed || len(data) != block.Length {
		return false
	}

	block.Data = data
	block.Completed = true
	block.Requested = false
	block.RequestedBy = ""
	block.startedAt = time.Time{}

	p.downloadedBlock++

	return true
}

func (p *Piece) saveCompletePiece(store storage.Storage) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.Blocks == nil {
		return fmt.Errorf("piece %v does not have buffer data to write", p.Index)
	}

	data := make([]byte, 0, p.Length)
	for i := range p.Blocks {
		data = append(data, p.Blocks[i].Data...)
	}

	if sha1.Sum(data) != p.HashV1 {
		return fmt.Errorf("piece %d failed SHA-1 verification", p.Index)
	}

	if err := store.WritePiece(p.Index, data); err != nil {
		return err
	}

	p.Blocks = nil

	return nil
}

func (p *Piece) resetBlock(peerId string, offset int) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	block := p.blockAt(offset)
	if block == nil || !block.Requested || block.RequestedBy != peerId || block.Completed {
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

	p.downloadedBlock = 0
}
