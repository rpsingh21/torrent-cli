package peer

import (
	"bytes"
	"fmt"
	"log"
)

type Handshake struct {
	Pstr     string
	InfoHash [20]byte
	PeerID   [20]byte
}

func NewHandshake(infoHash [20]byte, peerID [20]byte) *Handshake {
	return &Handshake{
		Pstr:     "BitTorrent protocol",
		InfoHash: infoHash,
		PeerID:   peerID,
	}
}

func (h *Handshake) Encode() []byte {
	buf := make([]byte, len(h.Pstr)+49)
	buf[0] = byte(len(h.Pstr))
	curr := 1
	curr += copy(buf[curr:], h.Pstr)
	curr += copy(buf[curr:], make([]byte, 8)) // 8 reserved bytes
	curr += copy(buf[curr:], h.InfoHash[:])
	curr += copy(buf[curr:], h.PeerID[:])
	return buf
}

func DecodeHandshakeMsg(msg []byte) (*Handshake, error) {
	if len(msg) != 68 {
		return nil, fmt.Errorf("invalid handshake length: %d", len(msg))
	}

	if msg[0] != 19 || !bytes.Equal(msg[1:20], []byte("BitTorrent protocol")) {
		log.Printf("invalid handshake protocol: %v", msg[1:20])
		return nil, fmt.Errorf("invalid handshake protocol")
	}

	return &Handshake{
		Pstr:     "BitTorrent protocol",
		InfoHash: [20]byte(msg[28:48]),
		PeerID:   [20]byte(msg[48:68]),
	}, nil
}
