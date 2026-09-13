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
	ID string
	// IP               string
	// Port             uint16
	// Addr             netip.AddrPort
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

func NewPeer(myPeerId [20]byte, infoHash [20]byte, ID string, IP string, Port uint16) (*Peer, error) {
	ctx := context.WithoutCancel(context.TODO())
	address := fmt.Sprintf("%s:%d", IP, Port)

	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		log.Printf("Failed to create connection: %v\n", err)
		return nil, err
	}
	peerConn := NewConnection(ctx, conn)
	peer := &Peer{
		ID:         ID,
		Connection: peerConn,
	}

	handshake := NewHandshake(infoHash, myPeerId)

	n, err := conn.Write(handshake.Encode())
	if err != nil {
		log.Printf("Handshake Failed: %v", err)
		return nil, err
	}
	log.Printf("Handshake Successfully: %v", n)

	return peer, err
}
