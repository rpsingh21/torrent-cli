package peer

import (
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

const KEEPALIVE_TIMEOUT = 2 * time.Minute

type requestKey struct {
	piece  int
	offset int
}

type Peer struct {
	ID               string
	IP               string
	Port             uint16
	metaInfo         *torrent.MetaInfo
	Connection       *Connection
	Choked           bool
	Interested       bool
	RemoteChoked     bool
	RemoteInterested bool
	stat             *Stat
	pieceManager     *piece.Manager
	removeChan       chan *Peer
	bitfield         *bitfield.Bitfield
	pending          map[requestKey]time.Time
	pendingMu        sync.Mutex
	ctx              context.Context
	cancel           context.CancelFunc
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

func (p *Peer) Start(parent context.Context) error {
	if p.metaInfo == nil || p.pieceManager == nil {
		return fmt.Errorf("peer is missing metainfo or piece manager")
	}

	if p.pending == nil {
		p.pending = make(map[requestKey]time.Time)
		p.stat = &Stat{}
		p.Choked = true
		p.RemoteChoked = true
	}

	ctx, cancel := context.WithCancel(parent)
	p.ctx, p.cancel = ctx, cancel
	defer cancel()

	address := p.Key()
	dialer := net.Dialer{Timeout: REQUEST_TIMEOUT * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", address)

	if err != nil {
		return err
	}

	p.Connection = NewConnection(ctx, conn, p.metaInfo.InfoHash, p.metaInfo.AppId)
	defer p.Connection.Close()

	if err := p.Connection.Handshake(); err != nil {
		return err
	}

	if _, err := p.Connection.WriteMessage(&Message{ID: MsgInterested}); err != nil {
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
		p.Connection.SetReadDeadline(p.nextReadDeadline())

		message, err := p.Connection.ReadMessage()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				p.expireRequests()
				continue
			}
			return err
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

	for p.pendingCount() < REQUESTS_PER_PEER {
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

		if n, err := p.Connection.WriteMessage(message); err != nil {
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
		wantBytes := (p.pieceManager.Metainfo.TotalPices + 7) / 8
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
		_, pending := p.pending[key]
		if pending {
			delete(p.pending, key)
		}
		p.pendingMu.Unlock()
		if !pending {
			return fmt.Errorf("unsolicited piece block %d/%d from %s", block.Index, block.Begin, p.Key())
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
	case MsgRequest, MsgCancel, MsgPort, MsgSuggest, MsgRejectRequest:
		log.Println("Upload-side messages are intentionally ignored in this download-only beta.")
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

func (p *Peer) nextReadDeadline() time.Time {
	deadline := time.Now().Add(KEEPALIVE_TIMEOUT)
	p.pendingMu.Lock()
	for _, started := range p.pending {
		d := started.Add(REQUEST_TIMEOUT * time.Second)
		if d.Before(deadline) {
			deadline = d
		}
	}
	p.pendingMu.Unlock()
	return deadline
}

func (p *Peer) expireRequests() {
	now := time.Now()
	var expired []requestKey
	p.pendingMu.Lock()
	for key, started := range p.pending {
		if now.Sub(started) >= REQUEST_TIMEOUT*time.Second {
			expired = append(expired, key)
			delete(p.pending, key)
		}
	}
	p.pendingMu.Unlock()
	for _, key := range expired {
		p.pieceManager.ReleaseBlock(p.schedulerID(), key.piece, key.offset)
		p.stat.IncTimeouts()
	}
}

func (p *Peer) releaseAllPending() {
	p.pendingMu.Lock()
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
	if p.cancel != nil {
		p.cancel()
	}
	p.releaseAllPending()
	if p.Connection != nil {
		return p.Connection.Close()
	}
	return nil
}
