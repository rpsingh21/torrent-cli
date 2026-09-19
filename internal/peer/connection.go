package peer

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
)

type Connection struct {
	remoteId string
	conn     net.Conn
	infoHash [20]byte
	appId    [20]byte
	reader   *bufio.Reader
	ctx      context.Context
	cancel   context.CancelFunc
}

func NewConnection(parent context.Context, conn net.Conn, infoHash [20]byte, appId [20]byte) *Connection {
	ctx, cancel := context.WithCancel(parent)

	return &Connection{
		conn:     conn,
		reader:   bufio.NewReader(conn),
		ctx:      ctx,
		cancel:   cancel,
		infoHash: infoHash,
		appId:    appId,
	}
}

func (pc *Connection) Handshake() error {
	handshake := NewHandshake(pc.infoHash, pc.appId)

	// Send our handshake.
	if _, err := pc.conn.Write(handshake.Encode()); err != nil {
		log.Printf("Handshake write failed: %v", err)
		return err
	}

	// Read peer handshake.
	buf := make([]byte, 68)

	if _, err := io.ReadFull(pc.reader, buf); err != nil {
		log.Printf("Handshake read failed: %v", err)
		return err
	}

	remote, err := DecodeHandshakeMsg(buf)
	if err != nil {
		log.Printf("Invalid peer handshake: %v", err)
		return err
	}

	// Make sure we're talking about the same torrent.
	if remote.InfoHash != pc.infoHash {
		return fmt.Errorf("info hash mismatch")
	}

	pc.remoteId = string(remote.PeerID[:])
	log.Printf(
		"Handshake successful with %v, peerID=%x",
		pc.conn.RemoteAddr(),
		remote.PeerID,
	)

	return nil
}

func (pc *Connection) ReadMessage() (*Message, error) {
	var lengthBuf [4]byte
	if _, err := io.ReadFull(pc.reader, lengthBuf[:]); err != nil {
		return nil, err
	}

	length := binary.BigEndian.Uint32(lengthBuf[:])

	// Keep-alive.
	if length == 0 {
		return nil, nil
	}

	// A peer message must at least contain the message ID.
	if length < 1 {
		return nil, fmt.Errorf("invalid message length: %d", length)
	}

	// TODO: configurable maximum.
	if length > 4*1024*1024 {
		return nil, fmt.Errorf("message too large: %d", length)
	}

	messageBuf := make([]byte, length)
	if _, err := io.ReadFull(pc.reader, messageBuf); err != nil {
		return nil, err
	}

	return ParseMessage(messageBuf)
}

func (pc *Connection) WriteMessage(m *Message) error {
	data := m.EncodeMessage()

	_, err := pc.conn.Write(data)
	if err != nil {
		log.Printf("Message write failed, Message type: %v", m.String())
	}

	return err
}

func (pc *Connection) Close() error {
	if pc.cancel != nil {
		pc.cancel()
	}
	if pc.conn != nil {
		return pc.conn.Close()
	}
	return nil
}
