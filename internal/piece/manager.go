package piece

import (
	"fmt"
	"log"
	"sync"

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
	Availability []uint16
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

		pieces[i] = &Piece{
			Index:           i,
			Length:          pieceSize,
			HashV1:          hash,
			downloadedBlock: 0,
		}
	}

	haveBitfield, err := bitfield.NewBitfield(len(pieces))
	if err != nil {
		panic(err)
	}

	return &Manager{
		Metainfo:     meta,
		Have:         haveBitfield,
		Pieces:       pieces,
		Availability: make([]uint16, len(pieces)),
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

	return m.Pieces[pieceIndex].reserveBlock(peerId)
}

func (m *Manager) ReleaseBlock(peerId string, pieceIndex, offset int) bool {
	if pieceIndex < 0 || pieceIndex >= len(m.Pieces) {
		return false
	}

	m.Pieces[pieceIndex].resetBlock(peerId, offset)

	m.mu.Lock()
	defer m.mu.Unlock()

	m.ReleaseQue[pieceIndex] = 0
	return true
}

func (m *Manager) RemovePeer(peerId string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.removePeerWithoutLock(peerId)
}

func (m *Manager) CompleteBlock(peerId string, pieceIndex, offset int, data []byte) bool {
	if pieceIndex < 0 || pieceIndex >= len(m.Pieces) {
		return false
	}

	piece := m.Pieces[pieceIndex]
	return piece.completeBlock(peerId, offset, data)
}

func (m *Manager) IsPieceReady(index int) bool {
	if index < 0 || index >= len(m.Pieces) {
		return false
	}

	p := m.Pieces[index]
	if len(p.Blocks) == 0 {
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
		if m.Have.Have(index) {
			log.Printf("piece already store in storage. hence raiseing, %v", err)
			return nil
		}
		return err
	}

	m.Have.SetIndex(index)
	m.completed++

	return nil
}

func (m *Manager) IsPieceComplete(index int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return index >= 0 && index < len(m.Pieces) && m.Have.Have(index)
}

func (m *Manager) Completed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.Have.AllSet()
}

func (m *Manager) ReDownloadPiece(index int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if index < 0 || index >= len(m.Pieces) || m.Have.Have(index) {
		return false
	}

	m.Pieces[index].resetAllBlock()
	return true
}
