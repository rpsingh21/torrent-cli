package peer

import (
	"bufio"
	"context"
	"encoding/binary"
	"log"
	"net"
	"sync"
)

type Connection struct {
	conn    net.Conn
	reader  *bufio.Reader
	writer  *bufio.Writer
	writeMu sync.Mutex
	ctx     context.Context
	cancel  context.CancelFunc

	remoteId string
	infoHash [20]byte
	myPeerId [20]byte
}

func NewConnection(parent context.Context, conn net.Conn, infoHash [20]byte, myPeerId [20]byte) *Connection {
	ctx, cancel := context.WithCancel(parent)

	return &Connection{
		conn:     conn,
		reader:   bufio.NewReader(conn),
		writer:   bufio.NewWriter(conn),
		ctx:      ctx,
		cancel:   cancel,
		infoHash: infoHash,
		myPeerId: myPeerId,
	}
}

func (pc *Connection) Handshake() error {
	handshake := NewHandshake(pc.infoHash, pc.myPeerId)

	if _, err := pc.conn.Write(handshake.Encode()); err != nil {
		log.Printf("Handshake Failed: %v", err)
		return err
	}
	log.Printf("Handshake successfully %v", pc.conn.RemoteAddr())
	return nil
}

func (pc *Connection) ReadMessage() (*Message, error) {
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

func (pc *Connection) WriteMessage(m *Message) error {
	if _, err := pc.conn.Write(m.EncodeMessage()); err != nil {
		log.Printf("Message Write Failed, Message type: %v\n", m.String())
		return err
	}
	return nil
}

func (pc *Connection) Close() {
	if pc.cancel != nil {
		pc.cancel()
	}
	if pc.conn != nil {
		pc.conn.Close()
	}
}
