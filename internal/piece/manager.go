package piece

import (
	"log"

	"github.com/rpsingh21/torrent-cli/internal/peer"
	"github.com/rpsingh21/torrent-cli/internal/torrent"
)

const (
	REQUEST_SIZE      = 16 * 1024
	MAX_PEERS         = 80
	REQUESTS_PER_PEER = 32
	REQUEST_TIMEOUT   = 15
)

type Manager struct {
	Metainfo *torrent.MetaInfo
	Pieces   []Piece
	Picker   *Picker
}

func NewManager(meta *torrent.MetaInfo) *Manager {
	pieces := make([]Piece, len(meta.PieceHashes))
	for i, ph := range meta.PieceHashes {
		pieceSize := uint32(min(meta.PieceLength, meta.TotalSize-int64(i)*meta.PieceLength))
		blocks := buildBuild(uint32(i), pieceSize)
		pieces[i] = *NewPiece(uint32(i), uint32(pieceSize), ph, blocks)
	}
	picker := NewPicker(len(meta.PieceHashes))

	return &Manager{
		Metainfo: meta,
		Pieces:   pieces,
		Picker:   picker,
	}
}

func buildBuild(pieceId uint32, pieceSize uint32) []Block {
	totalBlock := (pieceSize + REQUEST_SIZE - 1) / REQUEST_SIZE
	blocks := make([]Block, totalBlock)
	for i := range totalBlock {
		blocks[i] = Block{
			Piece:  pieceId,
			Offset: uint32(i) * REQUEST_SIZE,
			Length: min(REQUEST_SIZE, pieceSize-i*REQUEST_SIZE),
		}
	}
	return blocks
}

// func (m *Manager) NextPiece(peer *peer.Peer) (int, bool) {

// }

func (m *Manager) NextBlock(peer *peer.Peer) *Block {
	for i, piece := range m.Pieces {
		if peer.Bitfield.Have(i) {
			block := piece.MissingBlock()
			if block != nil {
				block.requested = true
				return block
			}
		}
	}
	return nil
}

func (m *Manager) ReceivedBlock(block *Block, data []byte) {
	if block == nil || len(data) != block.Length {
		return
	}

	piece := &m.Pieces[block.Piece]
	if piece.Blocks[block.offset/REQUEST_SIZE].completed {
		log.Printf("Duplicate download %+v\n", block)
		return
	}

	piece.Blocks[block.offset/REQUEST_SIZE].data = data
	piece.Blocks[block.offset/REQUEST_SIZE].completed = true

	if piece.Completed() {
		if piece.Verify() {
			m.writePiece(piece)
		}
	}
}

func (m *Manager) writePiece(piece *Piece) {
}

// func (m *Manager) RequestBlock() {

// }

// func (m *Manager) CompleteBlock()

// func (m *Manager) CompletePiece(index int)

// func (m *Manager) IsComplete(index int) bool

func (m *Manager) Completed() bool {
	return false
}

// func (m *Manager) Progress() float64
