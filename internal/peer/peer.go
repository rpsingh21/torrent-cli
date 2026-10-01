package peer

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/rpsingh21/torrent-cli/internal/piece"
	"github.com/rpsingh21/torrent-cli/internal/torrent"
	"github.com/rpsingh21/torrent-cli/pkg/bitfield"
)

type requestKey struct {
	piece  int
	offset int
}

type Peer struct {
	ID               string
	IP               string
	Port             uint16
	metaInfo         *torrent.MetaInfo
	conn             net.Conn
	reader           *bufio.Reader
	Choked           bool
	Interested       bool
	RemoteChoked     bool
	RemoteInterested bool
	stat             *Stat
	pieceManager     *piece.Manager
	bitfield         *bitfield.Bitfield
	pending          map[requestKey]time.Time
	pendingMu        sync.Mutex
	maxBlockRequest  int
	pendingMessage   chan Message
}

func NewPeer(id, ip string, port uint16, metaInfo *torrent.MetaInfo, pieceManager *piece.Manager) *Peer {
	return &Peer{
		ID:           id,
		IP:           ip,
		Port:         port,
		metaInfo:     metaInfo,
		pieceManager: pieceManager,
		stat:         &Stat{},
		Choked:       true,
		RemoteChoked: true,
		pending:      make(map[requestKey]time.Time),
	}
}

func (p *Peer) Key() string {
	return net.JoinHostPort(p.IP, strconv.Itoa(int(p.Port)))
}

func (p *Peer) schedulerID() string {
	return p.Key()
}

func (p *Peer) Start(pctx context.Context) error {
	// init while create
	p.maxBlockRequest = 16

	if p.metaInfo == nil || p.pieceManager == nil {
		return fmt.Errorf("peer is missing metainfo or piece manager")
	}

	if p.pending == nil {
		p.pending = make(map[requestKey]time.Time)
		p.stat = &Stat{}
		p.Choked = true
		p.RemoteChoked = true
	}

	ctx, cancel := context.WithCancel(pctx)
	defer cancel()

	address := p.Key()
	dialer := net.Dialer{Timeout: REQUEST_TIMEOUT}
	conn, err := dialer.DialContext(ctx, "tcp", address)

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
				return fmt.Errorf("peer %s timed out waiting for message: %w", p.ID, err)
			}

			return fmt.Errorf("read message from peer %s: %w", p.ID, err)
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

	// Todo Limit dynamic (back pressure based on dowload limit)
	for p.pendingCount() < p.maxBlockRequest {
		block := p.pieceManager.NextBlock(p.schedulerID())
		if block == nil {
			break
		}
		key := requestKey{piece: block.Piece, offset: block.Offset}

		// ToDo: It can create inline?
		message := &Message{
			ID:      MsgRequest,
			Payload: (&Request{Index: uint32(block.Piece), Begin: uint32(block.Offset), Length: uint32(block.Length)}).Encode(),
		}

		if n, err := p.WriteMessage(message); err != nil {
			p.pieceManager.ReleaseBlock(p.schedulerID(), block.Piece, block.Offset)
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
		p.releaseAllPending()

	case MsgUnchoke:
		p.Choked = false

	case MsgBitfield:
		wantBytes := (p.pieceManager.Metainfo.TotalPices + 7) >> 3
		if len(message.Payload) != wantBytes {
			return fmt.Errorf("invalid bitfield length %d, want %d %+v", len(message.Payload), wantBytes, p.pieceManager.Metainfo)
		}
		bf := bitfield.NewBitfieldFromBytes(message.Payload, p.pieceManager.Metainfo.TotalPices)
		if bf == nil {
			return fmt.Errorf("invalid bitfield from %s", p.Key())
		}
		p.bitfield = bf
		p.pieceManager.AddPeer(p.schedulerID(), bf)

	case MsgHave:
		if len(message.Payload) != 4 || p.bitfield == nil {
			return fmt.Errorf("invalid have message from %s", p.Key())
		}
		pieceIndex := binary.BigEndian.Uint32(message.Payload)
		if pieceIndex >= uint32(p.pieceManager.Metainfo.TotalPices) {
			return fmt.Errorf("invalid have piece index %d", pieceIndex)
		}
		if !p.bitfield.Have(int(pieceIndex)) {
			p.bitfield.SetIndex(int(pieceIndex))
			p.pieceManager.PeerHasPiece(p.schedulerID(), int(pieceIndex))
		}

	case MsgHaveAll:
		p.bitfield = bitfield.NewBitfield(p.pieceManager.Metainfo.TotalPices)
		for i := 0; i < p.pieceManager.Metainfo.TotalPices; i++ {
			p.bitfield.SetIndex(i)
		}
		p.pieceManager.AddPeer(p.schedulerID(), p.bitfield)

	case MsgHaveNone:
		p.bitfield = bitfield.NewBitfield(p.pieceManager.Metainfo.TotalPices)
		p.pieceManager.AddPeer(p.schedulerID(), p.bitfield)

	case MsgPiece:
		block, err := ParsePiece(message.Payload)
		if err != nil {
			return err
		}
		key := requestKey{piece: int(block.Index), offset: int(block.Begin)}

		p.pendingMu.Lock()
		startTime, pending := p.pending[key]
		if pending {
			delete(p.pending, key)

			if time.Since(startTime) > REQUEST_TIMEOUT {
				// arrange based on dowload letancy(stat).
				p.maxBlockRequest >>= 1
				log.Printf("peer: %v Decrease max request %v", p.IP, p.maxBlockRequest)

				// REFRESH TIMEOUTS: The peer is actively sending data.
				now := time.Now()
				for k := range p.pending {
					p.pending[k] = now.Add(10 * time.Second)
				}
			} else if p.maxBlockRequest < MAX_REQUESTS_PER_PEER && len(p.pending) <= p.maxBlockRequest && time.Since(startTime) < (3*time.Second) {
				p.maxBlockRequest = min(p.maxBlockRequest+8, MAX_REQUESTS_PER_PEER)
				log.Printf("peer: %v Increase max request %v", p.IP, p.maxBlockRequest)
			}
		}
		p.pendingMu.Unlock()

		if !pending {
			log.Printf("ignoring unsolicited/timed-out piece block %d/%d from %s", block.Index, block.Begin, p.Key())
			return nil
		}

		if p.maxBlockRequest < 8 {
			return fmt.Errorf("peer: %v is too slow and sent too many expired blocks, dropping connection", p.IP)
		}

		if !p.pieceManager.CompleteBlock(p.schedulerID(), int(block.Index), int(block.Begin), block.Data) {
			return fmt.Errorf("invalid piece block %d/%d from %s", block.Index, block.Begin, p.Key())
		}

		p.stat.IncRequestsCompleted()
		p.stat.AddDownloaded(len(block.Data))

		if p.pieceManager.IsPieceReady(int(block.Index)) {
			if err := p.pieceManager.CompletePiece(int(block.Index)); err != nil {
				p.dropPendingPiece(int(block.Index))
				log.Printf("piece %d rejected: %v", block.Index, err)
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

func (p *Peer) dropPendingPiece(pieceIndex int) {
	p.pendingMu.Lock()
	var keys []requestKey
	for key := range p.pending {
		if key.piece == pieceIndex {
			keys = append(keys, key)
			delete(p.pending, key)
		}
	}
	p.pendingMu.Unlock()

	for _, key := range keys {
		p.pieceManager.ReleaseBlock(p.schedulerID(), key.piece, key.offset)
	}
}

func (p *Peer) pendingCount() int {
	p.pendingMu.Lock()
	defer p.pendingMu.Unlock()
	return len(p.pending)
}

func (p *Peer) releaseAllPending() {
	p.pendingMu.Lock()
	log.Printf("===================== %v: releaseAllPending (%v)====================", p.IP, len(p.pending))
	pending := make([]requestKey, 0, len(p.pending))
	for key := range p.pending {
		pending = append(pending, key)
	}
	clear(p.pending)
	p.pendingMu.Unlock()

	for _, key := range pending {
		p.pieceManager.ReleaseBlock(p.schedulerID(), key.piece, key.offset)
	}
}

func (p *Peer) Close() error {
	p.releaseAllPending()
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}
