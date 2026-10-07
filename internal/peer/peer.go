package peer

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/rpsingh21/torrent-cli/internal/piece"
	"github.com/rpsingh21/torrent-cli/internal/torrent"
	"github.com/rpsingh21/torrent-cli/pkg/bitfield"
)

// Todo: Will remove once cleanup piece package
// Will use block. block without data
// Data store on piece level
type requestKey struct {
	piece  int
	offset int
}

type Peer struct {
	PeerId string
	Addr   string

	Choked           bool
	Interested       bool
	RemoteChoked     bool
	RemoteInterested bool

	maxBlockRequest int

	pending   map[requestKey]time.Time
	pendingMu sync.Mutex

	conn   net.Conn
	reader *bufio.Reader

	metaInfo     *torrent.MetaInfo
	pieceManager *piece.Manager

	bitfield *bitfield.Bitfield
	stat     *Stat

	fibrillation int64

	utPex uint8
}

func NewPeer(id, addr string, metaInfo *torrent.MetaInfo) *Peer {
	return &Peer{
		PeerId:       id,
		Addr:         addr,
		metaInfo:     metaInfo,
		stat:         &Stat{},
		Choked:       true,
		RemoteChoked: true,

		pending:         make(map[requestKey]time.Time),
		maxBlockRequest: MIN_REQUESTS_PER_PEER,
		fibrillation:    -1,
	}
}

func (p *Peer) Start(pctx context.Context) error {
	if p.metaInfo == nil || p.pieceManager == nil {
		return fmt.Errorf("peer is missing metainfo or piece manager")
	}

	// init while create
	// p.maxBlockRequest = MIN_REQUESTS_PER_PEER
	// p.fibrillation = -1

	// if p.pending == nil {
	// 	p.pending = make(map[requestKey]time.Time)
	// 	p.stat = &Stat{}
	// 	p.Choked = true
	// 	p.RemoteChoked = true
	// }

	ctx, cancel := context.WithCancel(pctx)
	defer cancel()

	dialer := net.Dialer{Timeout: REQUEST_TIMEOUT}
	conn, err := dialer.DialContext(ctx, "tcp", p.Addr)

	if err != nil {
		return err
	}
	p.conn = conn
	p.reader = bufio.NewReader(conn)

	if _, err := p.Handshake(); err != nil {
		return err
	}

	if _, err := p.WriteMessage(&Message{ID: MsgInterested}); err != nil {
		return err
	}

	p.Interested = true
	p.Choked = true

	go p.controllerLoop(ctx)
	return p.messageLoop()
}

func (p *Peer) messageLoop() error {
	for {
		if err := p.fillRequests(); err != nil {
			return err
		}

		message, err := p.ReadMessage()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				return fmt.Errorf("peer %s timed out waiting for message: %w", p.Addr, err)
			}

			return fmt.Errorf("read message from peer %s: %w", p.Addr, err)
		}

		if message == nil {
			continue
		}

		if err := p.handleMessage(message); err != nil {
			return err
		}
	}
}

func (p *Peer) fillRequests() error {
	if p.Choked || p.bitfield == nil {
		return nil
	}

	for p.pendingCount() < p.maxBlockRequest {
		block := p.pieceManager.NextBlock(p.Addr)
		if block == nil {
			break
		}
		key := requestKey{piece: block.Piece, offset: block.Offset}

		message := &Message{
			ID:      MsgRequest,
			Payload: (&Request{Index: uint32(block.Piece), Begin: uint32(block.Offset), Length: uint32(block.Length)}).Encode(),
		}

		if n, err := p.WriteMessage(message); err != nil {
			p.pieceManager.ReleaseBlock(p.Addr, block.Piece, block.Offset)
			return err
		} else {
			p.stat.AddUploaded(n)
		}

		p.pendingMu.Lock()
		p.pending[key] = time.Now()
		p.pendingMu.Unlock()
		p.stat.IncRequestsSent()
	}
	return nil
}

func (p *Peer) handleMessage(message *Message) error {
	switch message.ID {
	case MsgChoke:
		p.Choked = true
		snapshot := p.stat.Snapshot()
		if p.fibrillation != -1 && snapshot.RequestsCompleted-p.fibrillation == 0 {
			return fmt.Errorf("bad peer %v, frequently fibrillation.", p.Addr)
		}
		p.fibrillation = snapshot.RequestsCompleted
		p.conn.SetReadDeadline(time.Now().Add(KEEPALIVE_TIMEOUT))

	case MsgUnchoke:
		p.Choked = false

	case MsgBitfield:
		wantBytes := (p.pieceManager.Metainfo.TotalPices + 7) >> 3
		if len(message.Payload) != wantBytes {
			return fmt.Errorf("invalid bitfield length %d, want %d %+v", len(message.Payload), wantBytes, p.pieceManager.Metainfo)
		}

		bf, err := bitfield.NewBitfieldFromBytes(message.Payload, p.pieceManager.Metainfo.TotalPices)
		if err != nil {
			return err
		}

		if bf == nil {
			return fmt.Errorf("invalid bitfield from %s", p.Addr)
		}
		p.bitfield = bf
		p.pieceManager.AddPeer(p.Addr, bf)

	case MsgHave:
		if len(message.Payload) != 4 || p.bitfield == nil {
			return fmt.Errorf("invalid have message from %s", p.Addr)
		}
		pieceIndex := binary.BigEndian.Uint32(message.Payload)
		if pieceIndex >= uint32(p.pieceManager.Metainfo.TotalPices) {
			return fmt.Errorf("invalid have piece index %d", pieceIndex)
		}
		if !p.bitfield.Have(int(pieceIndex)) {
			p.bitfield.SetIndex(int(pieceIndex))
			p.pieceManager.PeerHasPiece(p.Addr, int(pieceIndex))
		}

	case MsgHaveAll:
		p.bitfield, _ = bitfield.NewBitfield(p.pieceManager.Metainfo.TotalPices)
		for i := 0; i < p.pieceManager.Metainfo.TotalPices; i++ {
			p.bitfield.SetIndex(i)
		}
		p.pieceManager.AddPeer(p.Addr, p.bitfield)

	case MsgHaveNone:
		p.bitfield, _ = bitfield.NewBitfield(p.pieceManager.Metainfo.TotalPices)
		p.pieceManager.AddPeer(p.Addr, p.bitfield)

	case MsgPiece:
		block, err := ParsePiece(message.Payload)
		if err != nil {
			return err
		}
		key := requestKey{piece: int(block.Index), offset: int(block.Begin)}

		p.pendingMu.Lock()
		_, pending := p.pending[key]
		if pending {
			delete(p.pending, key)
		}
		p.stat.AddDownloaded(len(block.Data))
		p.pendingMu.Unlock()

		// Todo: Check is valid case
		if !pending {
			log.Printf("ignoring unsolicited/timed-out piece block %d/%d from %s", block.Index, block.Begin, p.Addr)
			return nil
		}

		if !p.pieceManager.CompleteBlock(p.Addr, int(block.Index), int(block.Begin), block.Data) {
			return fmt.Errorf("invalid piece block %d/%d from %s", block.Index, block.Begin, p.Addr)
		}

		p.stat.IncRequestsCompleted()

		if p.pieceManager.IsPieceReady(int(block.Index)) {
			if err := p.pieceManager.SaveCompletePiece(int(block.Index)); err != nil {
				p.pieceManager.ReDownloadPiece(int(block.Index))
				log.Printf("peer: %v piece %d rejected: %v", p.Addr, block.Index, err)
			}
		}

	case MsgExtended:
		// if err := p.handleExtendedMessage(message); err != nil {
		// 	return err
		// }
		log.Printf("extened message %v", message.name())

	case MsgRequest, MsgCancel, MsgPort, MsgSuggest, MsgRejectRequest:
		log.Println("upload-side messages are intentionally ignored in this download-only beta.")
		// Upload-side messages are intentionally ignored in this download-only beta.
	}

	return nil
}

func (p *Peer) pendingCount() int {
	p.pendingMu.Lock()
	defer p.pendingMu.Unlock()
	return len(p.pending)
}

func (p *Peer) releaseAllPending() {
	p.pendingMu.Lock()
	log.Printf("peer %v releaseAllPending_called (%v)", p.Addr, len(p.pending))
	pending := make([]requestKey, 0, len(p.pending))
	for key := range p.pending {
		pending = append(pending, key)
	}
	clear(p.pending)
	p.pendingMu.Unlock()

	for _, key := range pending {
		p.pieceManager.ReleaseBlock(p.Addr, key.piece, key.offset)
	}
}

func (p *Peer) controllerLoop(ctx context.Context) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	lastCompletedBlock := 0

	for {
		select {
		case <-ticker.C:
			if !p.Choked {
				lastCompletedBlock = p.updateRequestWindow(lastCompletedBlock)
			}

		case <-ctx.Done():
			return
		}
	}
}

func (p *Peer) updateRequestWindow(lastCompletedBlock int) int {
	blockCompleted := int(p.stat.GetRequestsCompleted())
	completedInWC := blockCompleted - lastCompletedBlock

	// if p.maxBlockRequest != min(MAX_REQUESTS_PER_PEER, max(8, completedInWC)) {
	// 	log.Printf("peer %v Change Request window previous: %v Now: %v, Total change %v",
	// 		p.Addr, p.maxBlockRequest, min(MAX_REQUESTS_PER_PEER, max(8, completedInWC)),
	// 		completedInWC-p.maxBlockRequest)
	// }
	p.maxBlockRequest = min(MAX_REQUESTS_PER_PEER, max(MIN_REQUESTS_PER_PEER, completedInWC))

	return blockCompleted
}

func (p *Peer) Close() error {
	p.releaseAllPending()
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}
