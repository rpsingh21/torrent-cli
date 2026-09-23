package peer

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"
)

const maxPeerMessageLength = 2 * 1024 * 1024

type Connection struct {
	remoteID string
	conn     net.Conn
	infoHash [20]byte
	appID    [20]byte
	reader   *bufio.Reader
	buff     []byte
	ctx      context.Context
	cancel   context.CancelFunc
}

func NewConnection(parent context.Context, conn net.Conn, infoHash [20]byte, appID [20]byte) *Connection {
	ctx, cancel := context.WithCancel(parent)

	return &Connection{
		conn:     conn,
		reader:   bufio.NewReader(conn),
		ctx:      ctx,
		cancel:   cancel,
		infoHash: infoHash,
		appID:    appID,
	}
}

func (c *Connection) Handshake() error {
	if _, err := c.writeAll(NewHandshake(c.infoHash, c.appID).Encode()); err != nil {
		return fmt.Errorf("write handshake: %w", err)
	}

	buf := make([]byte, 68)
	if _, err := io.ReadFull(c.reader, buf); err != nil {
		return fmt.Errorf("read handshake: %w", err)
	}

	remote, err := DecodeHandshakeMsg(buf)
	if err != nil {
		return err
	}

	if remote.InfoHash != c.infoHash {
		return fmt.Errorf("info hash mismatch")
	}

	c.remoteID = string(remote.PeerID[:])
	return nil
}

func (c *Connection) ReadMessage() (*Message, error) {
	var lengthBuf [4]byte
	if _, err := io.ReadFull(c.reader, lengthBuf[:]); err != nil {
		return nil, err
	}

	length := binary.BigEndian.Uint32(lengthBuf[:])
	if length == 0 {
		return nil, nil
	}
	if length > maxPeerMessageLength {
		return nil, fmt.Errorf("message too large: %d vs %d", length, maxPeerMessageLength)
	}

	messageBuf := make([]byte, int(length))
	if _, err := io.ReadFull(c.reader, messageBuf); err != nil {
		return nil, err
	}

	return ParseMessage(messageBuf)
}

func (c *Connection) WriteMessage(m *Message) (int, error) {
	n, err := c.writeAll(m.EncodeMessage())
	if err != nil {
		return 0, fmt.Errorf("write %s: %w", m.String(), err)
	}
	return n, nil
}

func (c *Connection) writeAll(data []byte) (int, error) {
	total := 0

	for len(data) > 0 {
		n, err := c.conn.Write(data)
		total += n

		if err != nil {
			return total, err
		}

		if n == 0 {
			return total, io.ErrShortWrite
		}

		data = data[n:]
	}

	return total, nil
}

func (c *Connection) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

func (c *Connection) Close() error {
	if c.cancel != nil {
		c.cancel()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
