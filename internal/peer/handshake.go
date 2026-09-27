package peer

import (
	"bytes"
	"fmt"
)

const (
	BitTorrentProtocol = "BitTorrent protocol"

	// BEP 10: Extension Protocol
	ExtensionProtocolByte = 5
	ExtensionProtocolBit  = 0x10
)

type Handshake struct {
	Pstr     string
	Reserved [8]byte
	InfoHash [20]byte
	PeerID   [20]byte
}

func NewHandshake(infoHash [20]byte, peerID [20]byte) *Handshake {
	h := &Handshake{
		Pstr:     BitTorrentProtocol,
		InfoHash: infoHash,
		PeerID:   peerID,
	}

	h.EnableExtensions()

	return h
}

func (h *Handshake) Encode() []byte {
	buf := make([]byte, 68)
	buf[0] = byte(len(h.Pstr))
	curr := 1
	curr += copy(buf[curr:], h.Pstr)
	curr += copy(buf[curr:], h.Reserved[:])
	curr += copy(buf[curr:], h.InfoHash[:])
	curr += copy(buf[curr:], h.PeerID[:])
	return buf
}

func (h *Handshake) EnableExtensions() {
	h.Reserved[ExtensionProtocolByte] |= ExtensionProtocolBit
}

func (h *Handshake) SupportsExtensions() bool {
	return h.Reserved[ExtensionProtocolByte]&ExtensionProtocolBit != 0
}

func DecodeHandshakeMsg(msg []byte) (*Handshake, error) {
	if len(msg) != 68 {
		return nil, fmt.Errorf("invalid handshake length: %d", len(msg))
	}

	if msg[0] != 19 || !bytes.Equal(msg[1:20], []byte(BitTorrentProtocol)) {
		return nil, fmt.Errorf("invalid handshake protocol: %v", msg[1:20])
	}

	var reserved [8]byte
	copy(reserved[:], msg[20:28])

	var infoHash [20]byte
	copy(infoHash[:], msg[28:48])

	var peerID [20]byte
	copy(peerID[:], msg[48:68])

	return &Handshake{
		Pstr:     BitTorrentProtocol,
		Reserved: reserved,
		InfoHash: infoHash,
		PeerID:   peerID,
	}, nil
}
