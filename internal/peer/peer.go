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
	Bitfield         *Bitfield
	stat             *Stat
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
		Bitfield:         nil,
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

	p.Connection = NewConnection(ctx, conn, p.InfoHash, p.MyPeerId)
	if err := p.Connection.Handshake(); err != nil {
		return err
	}
	log.Printf("Ending without error %v", p.IP)
	return nil
}

func (p *Peer) Close() {
	if p.Connection != nil {
		p.Connection.Close()
	}
}
