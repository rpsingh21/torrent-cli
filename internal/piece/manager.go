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
	BLOCK_SIZE = 16 * 1024
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
	ReleaseQue   map[int]any

	completed  int
	inprogress int
}

func NewManager(meta *torrent.MetaInfo, strategy PickStrategy, store storage.Storage) *Manager {
	pieces := make([]*Piece, len(meta.PieceHashes))

	for i, hash := range meta.PieceHashes {
		remaining := meta.TotalSize - int64(i)*meta.PieceLength
		pieceSize := int(min(meta.PieceLength, remaining))
		blocks, size := buildBlocks(i, pieceSize)

		pieces[i] = &Piece{
			Index:           i,
			Offset:          i * int(meta.PieceLength),
			Length:          pieceSize,
			HashV1:          hash,
			Blocks:          blocks,
			toatalBlock:     size,
			downloadedBlock: 0,
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
		ReleaseQue:   make(map[int]any),
	}

}

func (m *Manager) GetStat() (int, int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.completed, m.inprogress
}

func buildBlocks(pieceId, pieceSize int) ([]Block, int) {
	if pieceSize <= 0 {
		return nil, 0
	}

	totalBlocks := (pieceSize + BLOCK_SIZE - 1) / BLOCK_SIZE
	blocks := make([]Block, totalBlocks)

	for i := range blocks {
		offset := i * BLOCK_SIZE
		blocks[i] = Block{Piece: pieceId, Offset: offset, Length: min(BLOCK_SIZE, pieceSize-offset)}
	}

	return blocks, totalBlocks
}

// NextBlock atomically reserves a block for a peer.
func (m *Manager) NextBlock(peerId string) *Block {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.nextNewBlock(peerId)
}

func (m *Manager) nextNewBlock(peerId string) *Block {
	pieceIndex := m.Pick(peerId)
	if pieceIndex < 0 || pieceIndex >= len(m.Pieces) {
		return nil
	}

	block := m.Pieces[pieceIndex].NextMissingBlock()
	if block == nil {
		return nil
	}

	block.Requested = true
	block.RequestedBy = peerId
	block.startedAt = time.Now()

	return block
}

func (m *Manager) ReleaseBlock(peerId string, pieceIndex, offset int) bool {
	if pieceIndex < 0 || pieceIndex >= len(m.Pieces) {
		return false
	}

	piece := m.Pieces[pieceIndex]
	piece.resetBlock(peerId, offset)

	m.mu.Lock()
	defer m.mu.Unlock()

	m.ReleaseQue[pieceIndex] = 0
	return true
}

// RemovePeer releases all blocks owned by a disconnected peer.
// Todo: Will implement via queue.
func (m *Manager) RemovePeer(peerId string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.removeWithoutLock(peerId)
	for _, p := range m.Pieces {
		for i := range p.Blocks {
			b := &p.Blocks[i]
			if b.Requested && b.RequestedBy == peerId {
				log.Printf("==================invalid block found for peer %v, (%v | %v)", b.RequestedBy, p.Index, b.Offset)
				b.Requested = false
				b.RequestedBy = ""
				b.startedAt = time.Time{}

				m.ReleaseQue[b.Piece] = 0
			}
		}
	}
}

func (m *Manager) CompleteBlock(peerId string, pieceIndex, offset int, data []byte) bool {
	if pieceIndex < 0 || pieceIndex >= len(m.Pieces) {
		return false
	}

	piece := m.Pieces[pieceIndex]
	return piece.completeBlock(peerId, offset, data)
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

func (m *Manager) SaveCompletePiece(index int) error {
	if index < 0 || index >= len(m.Pieces) {
		return fmt.Errorf("invalid piece index %d", index)
	}

	piece := m.Pieces[index]

	if err := piece.saveCompletePiece(m.storage); err != nil {
		return err
	}

	m.Have.SetIndex(index)
	m.completed++

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
