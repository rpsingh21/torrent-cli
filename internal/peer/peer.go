package peer

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"sync/atomic"
	"time"

	"github.com/rpsingh21/torrent-cli/internal/piece"
	"github.com/rpsingh21/torrent-cli/pkg/bitfield"
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
	blockInProgres   atomic.Int32
	stat             *Stat
	pieceManager     *piece.Manager
	removeChan       chan *Peer
	bitfield         *bitfield.Bitfield
	// ctx            context.Context
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
		blockInProgres:   atomic.Int32{},
		stat:             &Stat{},
	}
	return peer
}

func (p *Peer) Start() error {
	ctx := context.WithoutCancel(context.TODO())
	address := net.JoinHostPort(p.IP, fmt.Sprintf("%d", p.Port))

	conn, err := net.DialTimeout("tcp", address, REQUEST_TIMEOUT*time.Second)
	if err != nil {
		log.Printf("Failed to create connection: %v", err)
		p.removeChan <- p
		return err
	}

	defer p.Close()
	p.Connection = NewConnection(ctx, conn, p.InfoHash, p.MyPeerId)
	if err := p.Connection.Handshake(); err != nil {
		p.removeChan <- p
		return err
	}

	if err := p.Connection.WriteMessage(&Message{ID: MsgInterested}); err != nil {
		p.removeChan <- p
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
		if !p.Choked.Load() && p.blockInProgres.Load() < REQUESTS_PER_PEER {
			block := p.pieceManager.NextBlock(p.ID)
			if block != nil {
				requestBlock := Request{block.Piece, block.Offset, block.Length}
				message := &Message{MsgRequest, requestBlock.Encode()}
				if err := p.Connection.WriteMessage(message); err != nil {
					log.Printf("%v: Error while sending block request: %v", p.ID, err)
				}
			}
			if p.blockInProgres.Load() == 0 {
				break
			}
		}

		message, err := p.Connection.ReadMessage()
		if err != nil {
			log.Printf("%s: connection closed: %v", p.IP, err)
			break
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
			// Todo: Set bit size from metainfo
			log.Printf("%v: Bitfiled %+v", p.IP, message)
			p.bitfield = bitfield.NewBitfieldFromBytes(message.Payload, p.pieceManager.Metainfo.TotalPices)
			p.pieceManager.AddPeer(p.ID, p.bitfield)
		case MsgHave:
			// update have_piece
			p.pieceManager.UpdatePeer(p.ID, p.bitfield)
			pieceIndex := binary.LittleEndian.Uint32(message.Payload)
			p.bitfield.SetIndex(int(pieceIndex))
			log.Printf("%v: Have piece %+v", p.IP, message)
		case MsgPiece:
			// Call Resived piece
			log.Printf("%v: Message piece %+v", p.IP, message)
			block, err := ParsePiece(message.Payload)
			if err != nil {
				continue
			}
			p.pieceManager.CompleteBlock(int(block.Index), int(block.Begin), block.Data)
		case MsgRequest:
			// Todo: imp Later
			log.Printf("%v: Get piece Request from peer %+v", p.IP, message)
		case MsgCancel:
			// Todo: imp Later
			log.Printf("%v cancle", p.IP)
		default:
			log.Printf("%v, Default message %+v", p.IP, message)
		}
	}
	p.Close()
}

func (p *Peer) Close() error {
	if p.Connection == nil {
		return nil
	}

	err := p.Connection.Close()
	p.Connection = nil
	return err
}
