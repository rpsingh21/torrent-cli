package peer

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync/atomic"
	"time"
)

type Peer struct {
	ID   string
	IP   string
	Port uint16
	// Metainfo         *torrent.MetaInfo
	InfoHash         [20]byte
	MyPeerId         [20]byte
	Connection       *Connection
	Choked           atomic.Bool
	Interested       atomic.Bool
	RemoteChoked     atomic.Bool
	RemoteInterested atomic.Bool
	stat             *Stat
	// pieceManager     *piece.Manager
	blockInprogres map[*Piece]struct{}
	ctx            context.Context
	// Will imp
	// Extensions PeerExtensions
}

// func NewPeer(id string, ip string, port uint16) *Peer {
// 	return &Peer{
// 		ID: ,
// 	}
// }

func NewPeer(myPeerId [20]byte, infoHash [20]byte, Id string, IP string, Port uint16) *Peer {

	peer := &Peer{
		ID:               string(myPeerId[:]),
		IP:               IP,
		Port:             Port,
		InfoHash:         infoHash,
		MyPeerId:         myPeerId,
		Connection:       nil,
		Choked:           atomic.Bool{},
		Interested:       atomic.Bool{},
		RemoteChoked:     atomic.Bool{},
		RemoteInterested: atomic.Bool{},
		stat:             &Stat{},
	}
	return peer
}

func (p *Peer) Start() error {
	ctx := context.WithoutCancel(context.TODO())
	address := net.JoinHostPort(p.IP, fmt.Sprintf("%d", p.Port))

	conn, err := net.DialTimeout("tcp", address, 15*time.Second)
	if err != nil {
		log.Printf("Failed to create connection: %v", err)
		return err
	}

	defer p.Close()
	p.Connection = NewConnection(ctx, conn, p.InfoHash, p.MyPeerId)
	if err := p.Connection.Handshake(); err != nil {
		return err
	}

	if err := p.Connection.WriteMessage(&Message{ID: MsgInterested}); err != nil {
		return err
	}

	log.Printf("Peer %v: Starting loop", address)
	go p.messageLoop()
	return nil
}

func (p *Peer) messageLoop() {

	// Imp p.ctx.done for grassfull stop
	for {
		// Todo: maxBlockProgress load from config
		// for !p.Choked.Load() && len(p.blockInprogres) < 1 {
		// 	block := p.pieceManager.NextBlock(p)
		// 	if block != nil {
		// 		requestBlock := Request{block.Piece, block.Offset, block.Length}
		// 		message := &Message{MsgRequest, requestBlock.Encode()}
		// 		p.Connection.WriteMessage(message)
		// 	}
		// }
		message, err := p.Connection.ReadMessage()
		if err != nil {
			log.Printf("%s: connection closed: %v", p.IP, err)
			return
		}
		if message == nil {
			log.Printf("%v Keep live", p.IP)
			continue
		}
		switch message.ID {
		case MsgChoke:
			p.Choked.Store(true)
		case MsgUnchoke:
			p.Choked.Store(false)
		case MsgBitfield:
			// Update piece manager to SET peer bitfield
			log.Printf("%v: Bitfiled %+v", p.IP, message)
		case MsgHave:
			// update have_piece
			log.Printf("%v: Have piece %+v", p.IP, message)
		case MsgPiece:
			// Call Resived piece
			log.Printf("%v: Message piece %+v", p.IP, message)
		case MsgRequest:
			log.Printf("%v: Get piece Request from peer %+v", p.IP, message)
		case MsgCancel:
			log.Printf("%v cancle", p.IP)
		default:
			log.Printf("%v, Default message %+v", p.IP, message)
		}
	}
}

func (p *Peer) Close() error {
	if p.Connection == nil {
		return nil
	}

	err := p.Connection.Close()
	p.Connection = nil
	return err
}
