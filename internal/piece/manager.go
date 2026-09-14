package piece

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/rpsingh21/torrent-cli/internal/torrent"
	"github.com/rpsingh21/torrent-cli/pkg/bitfield"
)

// Todo: Load from config
const (
	REQUEST_SIZE = 16 * 1024
)

type Manager struct {
	ctx            context.Context
	cancle         context.CancelFunc
	Metainfo       *torrent.MetaInfo
	Have           *bitfield.Bitfield
	PeerBitfield   map[string]*bitfield.Bitfield
	Pieces         []Piece
	nextPieceMutex sync.Mutex
}

func NewManager(ctx context.Context, meta *torrent.MetaInfo) *Manager {
	ctx, cancle := context.WithCancel(ctx)
	pieces := make([]Piece, len(meta.PieceHashes))
	for i, ph := range meta.PieceHashes {
		log.Printf("PieceSize : %v : %v", meta.PieceLength, meta.TotalSize-int64(i)*meta.PieceLength)
		pieceSize := int(min(meta.PieceLength, meta.TotalSize-int64(i)*meta.PieceLength))
		blocks := buildBuild(i, pieceSize)
		pieces[i] = *NewPiece(i, pieceSize, ph, blocks)
	}

	m := &Manager{
		Metainfo:     meta,
		Have:         bitfield.NewBitfield(len(pieces)),
		PeerBitfield: make(map[string]*bitfield.Bitfield),
		Pieces:       pieces,
		ctx:          ctx,
		cancle:       cancle,
	}
	go m.cleanBlockedBlock()
	return m
}

func buildBuild(pieceId int, pieceSize int) []Block {
	totalBlock := (pieceSize + REQUEST_SIZE - 1) / REQUEST_SIZE
	blocks := make([]Block, totalBlock)
	for i := range totalBlock {
		blocks[i] = Block{
			Piece:  pieceId,
			Offset: i * REQUEST_SIZE,
			Length: min(REQUEST_SIZE, pieceSize-i*REQUEST_SIZE),
		}
	}
	return blocks
}

func (m *Manager) cleanBlockedBlock() {
	ticker := time.NewTicker(5 * time.Second)

	for {
		select {
		case <-m.ctx.Done():
			log.Printf("Canclling manager block!")
			return
		case t := <-ticker.C:
			timeNow := time.Now().Unix()
			log.Printf("%v: Tick from new manager at: %v, Name: %v", timeNow, t, m.Metainfo.Name)
			for _, piece := range m.Pieces {
				for _, block := range piece.Blocks {
					if block.Data != nil &&
						block.Requested &&
						!block.Completed &&
						timeNow-block.startedAt >= 10 {
						// 10 seconds sufficient to download one block
						// Todo configer from config
						block.Requested = false
						log.Printf("Timed out block: %+v", block)
					}
				}
			}
		}
	}
}

func (m *Manager) AddPeerBitfield(peerId string, bf *bitfield.Bitfield) {
	m.PeerBitfield[peerId] = bf
}

func (m *Manager) NextBlock(peerId string) *Block {
	log.Printf("Accqured log for peedId %v", peerId)
	m.nextPieceMutex.Lock()
	defer m.nextPieceMutex.Unlock()

	peerBF, ok := m.PeerBitfield[peerId]
	if !ok || peerBF == nil {
		log.Printf("Unknown peer: %v", peerId)
		return nil
	}

	for i, piece := range m.Pieces {
		if m.PeerBitfield[peerId].Have(i) {
			block := piece.NextMissingBlock()
			if block != nil {
				block.Requested = true
				block.startedAt = time.Now().Unix()
				return block
			}
		}
	}
	return nil
}

func (m *Manager) CompletePiece(index int) {
	piece := m.Pieces[index]
	if piece.Verify() {
		m.Have.SetIndex(index)
		piece.write()
	} else {
		log.Printf("Piece verification failed, %v", index)
		m.reDownloadpiece(index)
	}

}

func (m *Manager) IsComplete(index int) bool {
	return m.Pieces[index].Completed()
}

func (m *Manager) Completed() bool {
	return m.Have.AllSet()
}

func (m *Manager) Close() {
	if m.cancle != nil {
		m.cancle()
	}
}

func (m *Manager) reDownloadpiece(index int) {
	for _, b := range m.Pieces[index].Blocks {
		b.Requested = false
		b.Completed = false
		b.Data = nil
	}
}
