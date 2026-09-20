package piece

import (
	"fmt"
	"sync"
	"time"

	"github.com/rpsingh21/torrent-cli/internal/storage"
	"github.com/rpsingh21/torrent-cli/internal/torrent"
	"github.com/rpsingh21/torrent-cli/pkg/bitfield"
)

const (
	REQUEST_SIZE  = 16 * 1024
	BLOCK_TIMEOUT = 10 * time.Second
)

type Manager struct {
	Metainfo     *torrent.MetaInfo
	Have         *bitfield.Bitfield
	Pieces       []*Piece
	Availability []uint32
	PeerPieces   map[string]*bitfield.Bitfield
	Strategy     PickStrategy
	next         int
	mu           sync.Mutex
	storage      storage.Storage
}

func NewManager(meta *torrent.MetaInfo, strategy PickStrategy, store storage.Storage) *Manager {
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

	return &Manager{
		Metainfo:     meta,
		Have:         bitfield.NewBitfield(len(pieces)),
		Pieces:       pieces,
		Availability: make([]uint32, len(pieces)),
		PeerPieces:   make(map[string]*bitfield.Bitfield),
		Strategy:     strategy,
		storage:      store,
	}

}

func buildBlocks(pieceID, pieceSize int) []Block {
	if pieceSize <= 0 {
		return nil
	}

	totalBlocks := (pieceSize + REQUEST_SIZE - 1) / REQUEST_SIZE
	blocks := make([]Block, totalBlocks)

	for i := range blocks {
		offset := i * REQUEST_SIZE
		blocks[i] = Block{Piece: pieceID, Offset: offset, Length: min(REQUEST_SIZE, pieceSize-offset)}
	}

	return blocks
}

// NextBlock atomically reserves a block for a peer.
func (m *Manager) NextBlock(peerID string) *Block {
	m.mu.Lock()
	defer m.mu.Unlock()

	pieceIndex := m.Pick(peerID)
	if pieceIndex < 0 || pieceIndex >= len(m.Pieces) {
		return nil
	}

	block := m.Pieces[pieceIndex].NextMissingBlock()
	if block == nil {
		return nil
	}

	block.Requested = true
	block.RequestedBy = peerID
	block.startedAt = time.Now()

	return block
}

func (m *Manager) ReleaseBlock(peerID string, pieceIndex, offset int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if pieceIndex < 0 || pieceIndex >= len(m.Pieces) {
		return false
	}

	block := m.Pieces[pieceIndex].blockAt(offset)
	if block == nil || !block.Requested || block.RequestedBy != peerID || block.Completed {
		return false
	}

	block.Requested = false
	block.RequestedBy = ""
	block.startedAt = time.Time{}
	return true
}

// RemovePeer releases all blocks owned by a disconnected peer.
// Todo: Will implement via queue.
func (m *Manager) RemovePeer(peerID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.removeWithoutLock(peerID)
	for _, p := range m.Pieces {
		for i := range p.Blocks {
			b := &p.Blocks[i]
			if b.Requested && b.RequestedBy == peerID {
				b.Requested = false
				b.RequestedBy = ""
				b.startedAt = time.Time{}
			}
		}
	}
}

func (m *Manager) CompleteBlock(peerID string, pieceIndex, offset int, data []byte) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if pieceIndex < 0 || pieceIndex >= len(m.Pieces) {
		return false
	}

	block := m.Pieces[pieceIndex].blockAt(offset)
	if block == nil || !block.Requested || block.RequestedBy != peerID || block.Completed || len(data) != block.Length {
		return false
	}

	block.Data = data
	block.Completed = true
	block.Requested = false
	block.RequestedBy = ""
	block.startedAt = time.Time{}

	return true
}

func (m *Manager) IsPieceReady(index int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if index < 0 || index >= len(m.Pieces) {
		return false
	}

	p := m.Pieces[index]
	if p.Verifying || len(p.Blocks) == 0 {
		return false
	}

	return p.Completed()
}

// CompletePiece verifies a fully received piece and persists it. The manager
// mutex is deliberately not held while storage I/O occurs.
func (m *Manager) CompletePiece(index int) error {
	m.mu.Lock()
	if index < 0 || index >= len(m.Pieces) {
		m.mu.Unlock()
		return fmt.Errorf("invalid piece index %d", index)
	}

	p := m.Pieces[index]
	if p.Verifying || !p.Completed() {
		m.mu.Unlock()
		return nil
	}

	p.Verifying = true
	data := make([]byte, 0, p.Length)
	for i := range p.Blocks {
		data = append(data, p.Blocks[i].Data...)
	}
	hashOK := p.Verify()
	m.mu.Unlock()

	if !hashOK {
		m.mu.Lock()
		m.resetPieceLocked(p)
		m.mu.Unlock()
		return fmt.Errorf("piece %d failed SHA-1 verification", index)
	}

	if m.storage != nil {
		if err := m.storage.WritePiece(index, data); err != nil {
			m.mu.Lock()
			m.resetPieceLocked(p)
			m.mu.Unlock()
			return fmt.Errorf("write piece %d: %w", index, err)
		}
	}

	m.mu.Lock()
	p.Verifying = false
	m.Have.SetIndex(index)

	p.Blocks = nil // persisted successfully; release the piece buffer
	m.mu.Unlock()
	return nil
}

func (m *Manager) IsComplete(index int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return index >= 0 && index < len(m.Pieces) && m.Have.Have(index)
}

func (m *Manager) Completed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.Have.AllSet()
}

func (m *Manager) resetPieceLocked(p *Piece) {
	p.Verifying = false
	for i := range p.Blocks {
		b := &p.Blocks[i]
		b.Requested = false
		b.RequestedBy = ""
		b.Completed = false
		b.startedAt = time.Time{}
		b.Data = nil
	}
	m.Have.ClearIndex(p.Index)
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
