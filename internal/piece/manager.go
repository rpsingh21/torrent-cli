package piece

import (
	"context"
	"sync"
	"time"

	"github.com/rpsingh21/torrent-cli/internal/torrent"
	"github.com/rpsingh21/torrent-cli/pkg/bitfield"
)

const (
	REQUEST_SIZE     = 16 * 1024
	BLOCK_TIMEOUT    = 10 * time.Second
	CLEANUP_INTERVAL = 10 * time.Second
)

type Manager struct {
	ctx          context.Context
	cancel       context.CancelFunc
	Metainfo     *torrent.MetaInfo
	Have         *bitfield.Bitfield
	Pieces       []*Piece
	Availability []uint32
	PeerPieces   map[string]*bitfield.Bitfield
	Strategy     PickStrategy
	next         int
	mu           sync.Mutex
}

func NewManager(
	ctx context.Context,
	meta *torrent.MetaInfo,
	strategy PickStrategy,
) *Manager {
	ctx, cancel := context.WithCancel(ctx)

	pieces := make([]*Piece, len(meta.PieceHashes))

	for i, hash := range meta.PieceHashes {
		remaining := meta.TotalSize - int64(i)*meta.PieceLength
		pieceSize := int(min(meta.PieceLength, remaining))

		pieces[i] = &Piece{
			Index:  i,
			Length: pieceSize,
			HashV1: hash,
			Blocks: buildBlocks(i, pieceSize),
		}
	}

	m := &Manager{
		ctx:          ctx,
		cancel:       cancel,
		Metainfo:     meta,
		Have:         bitfield.NewBitfield(len(pieces)),
		PeerPieces:   make(map[string]*bitfield.Bitfield),
		Pieces:       pieces,
		Strategy:     strategy,
		Availability: make([]uint32, len(meta.PieceHashes)),
	}

	go m.cleanBlockedBlocks()
	return m
}

func buildBlocks(pieceID, pieceSize int) []Block {
	if pieceSize <= 0 {
		return nil
	}

	totalBlocks := (pieceSize + REQUEST_SIZE - 1) / REQUEST_SIZE
	blocks := make([]Block, totalBlocks)

	for i := range blocks {
		offset := i * REQUEST_SIZE
		blocks[i] = Block{
			Piece:  pieceID,
			Offset: offset,
			Length: min(REQUEST_SIZE, pieceSize-offset),
		}
	}

	return blocks
}

func (m *Manager) cleanBlockedBlocks() {
	ticker := time.NewTicker(CLEANUP_INTERVAL)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			// log.Printf("Cancelling manager block cleanup")
			return

		case now := <-ticker.C:
			m.mu.Lock()
			m.cleanupExpiredBlocksLocked(now)
			m.mu.Unlock()
		}
	}
}

// cleanupExpiredBlocksLocked resets requests that have been outstanding for
// Todo implement via queue
func (m *Manager) cleanupExpiredBlocksLocked(now time.Time) int {
	cleaned := 0

	for _, piece := range m.Pieces {
		for i := range piece.Blocks {
			block := &piece.Blocks[i]

			if !block.Requested || block.Completed {
				continue
			}

			if now.Sub(block.startedAt) < BLOCK_TIMEOUT {
				continue
			}

			block.Requested = false
			block.startedAt = time.Time{}
			cleaned++
		}
	}

	return cleaned
}

func (m *Manager) cleanupExpiredBlocks(now time.Time) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.cleanupExpiredBlocksLocked(now)
}

// NextBlock returns the next block that this peer can download.
//
// The peer must advertise that it has the piece. A returned block is marked
// Requested before returning, so two concurrent callers cannot select the
// same block.
func (m *Manager) NextBlock(peerID string) *Block {
	m.mu.Lock()
	defer m.mu.Unlock()

	pieceIndex := m.Pick(peerID)

	// Pick return -1 if index out of range
	if pieceIndex < 0 {
		return nil
	}

	block := m.Pieces[pieceIndex].NextMissingBlock()

	block.Requested = true
	block.startedAt = time.Now()

	return block

}

// CompleteBlock stores data received for a block.
//
// A block must have been requested. The data is copied so the caller may
// safely reuse its network read buffer.
func (m *Manager) CompleteBlock(pieceIndex, offset int, data []byte) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if pieceIndex < 0 || pieceIndex >= len(m.Pieces) {
		return false
	}

	block := m.Pieces[pieceIndex].blockAt(offset)
	if block == nil || !block.Requested || block.Completed {
		return false
	}

	if len(data) != block.Length {
		return false
	}

	block.Data = append(block.Data[:0], data...)
	block.Completed = true
	block.Requested = false
	block.startedAt = time.Time{}

	return true
}

func (m *Manager) CompletePiece(index int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if index < 0 || index >= len(m.Pieces) {
		return false
	}

	piece := m.Pieces[index]

	if !piece.Completed() {
		return false
	}

	if !piece.Verify() {
		m.resetPieceLocked(piece)
		return false
	}

	m.Have.SetIndex(index)
	return true
}

func (m *Manager) IsComplete(index int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if index < 0 || index >= len(m.Pieces) {
		return false
	}

	return m.Pieces[index].Completed()
}

func (m *Manager) Completed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.Have.AllSet()
}

func (m *Manager) resetPieceLocked(piece *Piece) {
	for i := range piece.Blocks {
		block := &piece.Blocks[i]
		block.Requested = false
		block.Completed = false
		block.startedAt = time.Time{}
		block.Data = nil
	}
	m.Have.ClearIndex(piece.Index)
}

func (m *Manager) ReDownloadPiece(index int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if index < 0 || index >= len(m.Pieces) {
		return false
	}

	m.resetPieceLocked(m.Pieces[index])
	return true
}

func (m *Manager) Close() {
	if m.cancel != nil {
		m.cancel()
	}
}
