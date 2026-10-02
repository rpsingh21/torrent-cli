package peer

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"
)

func (p *Peer) Handshake() (*Handshake, error) {
	if _, err := p.writeAll(NewHandshake(p.metaInfo.InfoHash, p.metaInfo.AppId).Encode()); err != nil {
		return nil, fmt.Errorf("write handshake: %w", err)
	}

	buf := make([]byte, 68)
	if _, err := io.ReadFull(p.reader, buf); err != nil {
		return nil, fmt.Errorf("read handshake: %w", err)
	}

	remoteHs, err := DecodeHandshakeMsg(buf)
	if err != nil {
		return nil, err
	}

	if remoteHs.InfoHash != p.metaInfo.InfoHash {
		return nil, fmt.Errorf("info hash mismatch")
	}

	p.PeerId = string(remoteHs.PeerID[:])
	return remoteHs, nil
}

func (p *Peer) ReadMessage() (*Message, error) {
	if err := p.conn.SetReadDeadline(time.Now().Add(KEEPALIVE_TIMEOUT)); err != nil {
		return nil, err
	}

	var lengthBuf [4]byte
	if _, err := io.ReadFull(p.reader, lengthBuf[:]); err != nil {
		return nil, err
	}

	length := binary.BigEndian.Uint32(lengthBuf[:])
	if length == 0 {
		return nil, nil
	}
	if length > MAX_MESSAGE_LENGTH {
		return nil, fmt.Errorf("message too large: %d vs %d", length, MAX_MESSAGE_LENGTH)
	}

	messageBuf := make([]byte, int(length))
	if _, err := io.ReadFull(p.reader, messageBuf); err != nil {
		return nil, err
	}

	return ParseMessage(messageBuf)
}

func (p *Peer) WriteMessage(m *Message) (int, error) {
	n, err := p.writeAll(m.EncodeMessage())
	if err != nil {
		return 0, fmt.Errorf("write %s: %w", m.String(), err)
	}
	return n, nil
}

func (p *Peer) writeAll(data []byte) (int, error) {
	total := 0

	for len(data) > 0 {
		n, err := p.conn.Write(data)
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
