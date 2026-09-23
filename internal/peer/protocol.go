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
		return nil, fmt.Errorf("empty peer message")
	}
	id := MessageID(data[0])
	if !id.valid() {
		return nil, fmt.Errorf("unknown message ID: %d", id)
	}

	payload := data[1:]
	switch id {
	case MsgChoke, MsgUnchoke, MsgInterested, MsgNotInterested, MsgHaveAll, MsgHaveNone:
		if len(payload) != 0 {
			return nil, fmt.Errorf("message %d has invalid payload length %d", id, len(payload))
		}
	case MsgHave:
		if len(payload) != 4 {
			return nil, fmt.Errorf("have payload length %d, want 4", len(payload))
		}
	case MsgBitfield:
		// Variable length; the receiver validates the bitfield size.
	case MsgRequest, MsgCancel, MsgRejectRequest:
		if len(payload) != 12 {
			return nil, fmt.Errorf("message %d payload length %d, want 12", id, len(payload))
		}
	case MsgPiece:
		if len(payload) < 8 {
			return nil, fmt.Errorf("piece payload length %d, want >= 8", len(payload))
		}
	case MsgPort:
		if len(payload) != 2 {
			return nil, fmt.Errorf("port payload length %d, want 2", len(payload))
		}
	case MsgSuggest:
		if len(payload) != 4 {
			return nil, fmt.Errorf("suggest payload length %d, want 4", len(payload))
		}
	}

	return &Message{
		ID:      id,
		Payload: payload,
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

func (r *Request) Encode() []byte {
	buffer := make([]byte, 12)
	binary.BigEndian.PutUint32(buffer[:4], r.Index)
	binary.BigEndian.PutUint32(buffer[4:8], r.Begin)
	binary.BigEndian.PutUint32(buffer[8:], r.Length)
	return buffer
}

func (m *Message) EncodeMessage() []byte {
	if m == nil {
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
	names := [...]string{"Choke", "Unchoke", "Interested", "NotInterested", "Have", "Bitfield", "Request", "Piece", "Cancel", "Port", "Suggest", "HaveAll", "HaveNone", "RejectRequest"}
	if int(m.ID) < len(names) {
		return names[m.ID]
	}
	return fmt.Sprintf("UnknownMessageId#%d", m.ID)
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
