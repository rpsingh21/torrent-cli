package peer

import (
	"bufio"
	"context"
	"encoding/binary"
	"log"
	"net"
	"sync"
)

type PeerConnection struct {
	conn    net.Conn
	reader  *bufio.Reader
	writer  *bufio.Writer
	writeMu sync.Mutex
	ctx     context.Context
	cancel  context.CancelFunc
}

func NewPeerConnection(parent context.Context, conn net.Conn) *PeerConnection {
	ctx, cancel := context.WithCancel(parent)

	return &PeerConnection{
		conn:   conn,
		reader: bufio.NewReader(conn),
		writer: bufio.NewWriter(conn),
		ctx:    ctx,
		cancel: cancel,
	}
}

func (pc *PeerConnection) ReadMessage() (*Message, error) {
	lengthBuf := make([]byte, 4)
	if _, err := pc.reader.Read(lengthBuf); err != nil {
		log.Printf("Fail to read message length, Error: %v\n", err)
		return nil, err
	}
	length := binary.BigEndian.Uint32(lengthBuf)

	// keepLive
	if length == 0 {
		return nil, nil
	}

	messageBuf := make([]byte, length)
	if _, err := pc.reader.Read(messageBuf); err != nil {
		log.Printf("Fail to read message, Error: %v\n", err)
		return nil, err
	}
	messgae := &Message{
		ID:      MessageID(messageBuf[0]),
		Payload: messageBuf[1:],
	}
	return messgae, nil
}

func (pc *PeerConnection) WriteMessage(m *Message) error {
	if _, err := pc.conn.Write(m.EncodeMessage()); err != nil {
		log.Printf("Message Write Failed, Message type: %v\n", m.String())
		return err
	}
	return nil
}

func (pc *PeerConnection) Close() (*Message, error) {
	return nil, nil
}
