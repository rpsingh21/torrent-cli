package peer

import (
	"encoding/binary"
	"fmt"
)

type MessageID uint8

const (
	MsgChoke MessageID = iota
	MsgUnchoke
	MsgInterested
	MsgNotInterested
	MsgHave
	MsgBitfield
	MsgRequest
	MsgPiece
	MsgCancel
	MsgPort
	MsgSuggest
	MsgHaveAll
	MsgHaveNone
	MsgRejectRequest
)

type Message struct {
	ID      MessageID
	Payload []byte
}

type Request struct {
	Index  uint32
	Begin  uint32
	Length uint32
}

type Piece struct {
	Index uint32
	Begin uint32
	Data  []byte
}

// func ParseMessage(data []byte) (Message, error)

func (m *Message) EncodeMessage() []byte {
	// KeepLive message
	if m == nil {
		return make([]byte, 4)
	}
	payloadLen := len(m.Payload)
	buffer := make([]byte, 5+payloadLen)
	binary.BigEndian.PutUint32(buffer[:4], uint32(payloadLen+1)) // 1 for message ID
	buffer[4] = byte(m.ID)
	return buffer

	// For zero copy do like this
	// w.Write(header)
	// w.Write(payload)
}

// func ParseRequest(payload []byte) (Request, error)

// func ParsePiece(payload []byte) (Piece, error)

func (m *Message) name() string {
	if m == nil {
		return "KeepAlive"
	}
	switch m.ID {
	case MsgChoke:
		return "Choke"
	case MsgUnchoke:
		return "Unchoke"
	case MsgInterested:
		return "Interested"
	case MsgNotInterested:
		return "NotInterested"
	case MsgHave:
		return "Have"
	case MsgBitfield:
		return "Bitfield"
	case MsgRequest:
		return "Request"
	case MsgPiece:
		return "Piece"
	case MsgCancel:
		return "Cancel"
	default:
		return fmt.Sprintf("UnknownMessageId#%d", m.ID)
	}
}

func (m *Message) String() string {
	if m == nil {
		return m.name()
	}
	return fmt.Sprintf("%s [%d]", m.name(), len(m.Payload))
}
