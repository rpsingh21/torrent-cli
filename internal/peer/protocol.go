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

func ParseMessage(data []byte) (*Message, error) {
	if len(data) == 0 {
		return nil, nil
	}
	id := MessageID(data[0])
	if !id.valid() {
		return nil, fmt.Errorf("unknown message ID: %d", id)
	}
	return &Message{
		ID:      id,
		Payload: data[1:],
	}, nil
}

func ParseRequest(payload []byte) (*Request, error) {
	if len(payload) != 12 {
		return nil, fmt.Errorf("invalid request payload length: got %d, want 12", len(payload))
	}
	return &Request{
		Index:  binary.BigEndian.Uint32(payload[:4]),
		Begin:  binary.BigEndian.Uint32(payload[4:8]),
		Length: binary.BigEndian.Uint32(payload[8:12]),
	}, nil
}

func ParsePiece(payload []byte) (*Piece, error) {
	if len(payload) < 8 {
		return nil, fmt.Errorf("invalid piece payload: length %d", len(payload))
	}
	return &Piece{
		Index: binary.BigEndian.Uint32(payload[:4]),
		Begin: binary.BigEndian.Uint32(payload[4:8]),
		Data:  payload[8:],
	}, nil
}

func (m *Message) EncodeMessage() []byte {
	if m == nil {
		// Keep-alive message.
		return make([]byte, 4)
	}

	payloadLen := len(m.Payload)
	buffer := make([]byte, 5+payloadLen)

	binary.BigEndian.PutUint32(buffer[:4], uint32(payloadLen+1))
	buffer[4] = byte(m.ID)
	copy(buffer[5:], m.Payload)

	return buffer
}

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
	case MsgPort:
		return "Port"
	case MsgSuggest:
		return "Suggest"
	case MsgHaveAll:
		return "HaveAll"
	case MsgHaveNone:
		return "HaveNone"
	case MsgRejectRequest:
		return "RejectRequest"
	default:
		return fmt.Sprintf("UnknownMessageId#%d", m.ID)
	}
}

func (m *Message) String() string {
	if m == nil {
		return "KeepAlive"
	}

	return fmt.Sprintf("%s [%d]", m.name(), len(m.Payload))
}

func (id MessageID) valid() bool {
	return id <= MsgRejectRequest
}
