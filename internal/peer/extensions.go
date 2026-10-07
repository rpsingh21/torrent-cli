package peer

import (
	"bufio"
	"context"
	"crypto/sha1"
	"fmt"
	"log"
	"net"

	"github.com/rpsingh21/torrent-cli/internal/bencode"
	"github.com/rpsingh21/torrent-cli/internal/torrent"
)

// BEP-9: Extension Protocol for BitTorrent metadata.
//
// The metadata protocol itself is defined by:
// BEP-9: https://www.bittorrent.org/beps/bep_0009.html
// BEP-10: https://www.bittorrent.org/beps/bep_0010.html

const metadataPieceSize = 16 * 1024
const maxMetadataSize = 10 * 1024 * 1024

func (p *Peer) DownloadMetadata(ctx context.Context, metaInfo *torrent.MetaInfo) ([]byte, error) {
	p.metaInfo = metaInfo

	dialer := net.Dialer{Timeout: REQUEST_TIMEOUT}

	conn, err := dialer.DialContext(ctx, "tcp", p.Addr)
	if err != nil {
		return nil, err
	}

	p.conn = conn
	p.reader = bufio.NewReader(conn)

	done := make(chan struct{})
	defer close(done)

	go func() {
		select {
		case <-ctx.Done():
			p.conn.Close()
		case <-done:
		}
	}()

	peerHs, err := p.Handshake()
	if err != nil {
		return nil, fmt.Errorf("BitTorrent handshake failed: %w", err)
	}

	if !peerHs.SupportsExtensions() {
		return nil, fmt.Errorf("peer %s doesn't support extension protocol", p.Addr)
	}

	if err := p.ExtendedHandshake(); err != nil {
		return nil, err
	}

	data, err := p.metadataLoop(ctx)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (p *Peer) metadataLoop(ctx context.Context) ([]byte, error) {
	var (
		peerMetadataExtID uint8
		metadataSize      int
		pieceCount        int

		metadataBuf   []byte
		received      []bool
		receivedCount int
	)

	handshakeReceived := false

	for {
		message, err := p.ReadMessage()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				return nil, fmt.Errorf("metadata download timeout")
			}

			return nil, err
		}

		if message == nil || message.ID != MsgExtended {
			continue
		}

		extMessage, err := ParseExtendedMessage(message.Payload)
		if err != nil {
			return nil, fmt.Errorf("parse extended message: %w", err)
		}

		// Extended message ID 0 is always the extended handshake.
		if extMessage.ID == 0 {
			// Prevent duplicate handshakes from resetting the download buffers
			if handshakeReceived {
				continue
			}

			peerMetadataExtID, metadataSize, err = handleExtendedHandshakeMessage(extMessage)
			if err != nil {
				return nil, err
			}

			if peerMetadataExtID == 0 {
				return nil, fmt.Errorf("peer: %v does not support ut_metadata", p.Addr)
			}

			pieceCount = (metadataSize + metadataPieceSize - 1) / metadataPieceSize
			metadataBuf = make([]byte, metadataSize)
			received = make([]bool, pieceCount)

			// log.Printf("peer metadata: size=%d pieces=%d ut_metadata=%d", metadataSize, pieceCount, peerMetadataExtID)

			// Request ONLY the first piece to avoid flood protection disconnects.
			if pieceCount > 0 {
				if err := p.sendMetaRequest(peerMetadataExtID, 0); err != nil {
					return nil, err
				}
			}

			handshakeReceived = true
			continue
		}

		// Ignore extension messages until we receive
		// the peer's extended handshake.
		if !handshakeReceived {
			continue
		}

		// Ignore extensions that aren't ut_metadata.
		if extMessage.ID != peerMetadataExtID {
			continue
		}

		pieceIdx, data, msgType, err := handleExtendedMessage(extMessage)
		if err != nil {
			return nil, err
		}

		switch msgType {
		case 1: // piece
			if pieceIdx < 0 || pieceIdx >= pieceCount {
				return nil, fmt.Errorf("invalid metadata piece index: %d", pieceIdx)
			}

			if received[pieceIdx] {
				// Duplicate piece.
				continue
			}

			start := pieceIdx * metadataPieceSize
			end := min(start+len(data), metadataSize)

			if start >= metadataSize {
				return nil, fmt.Errorf("metadata piece %d starts outside metadata", pieceIdx)
			}

			copy(metadataBuf[start:end], data[:end-start])

			received[pieceIdx] = true
			receivedCount++

			log.Printf("peer: %v received metadata piece %d/%d", p.Addr, receivedCount, pieceCount)

			if receivedCount != pieceCount {
				// Find and request the next unreceived piece sequentially
				for i := 0; i < pieceCount; i++ {
					if !received[i] {
						if err := ctx.Err(); err != nil {
							return nil, err
						}

						if err := p.sendMetaRequest(peerMetadataExtID, i); err != nil {
							return nil, err
						}
						break // Only request one piece at a time
					}
				}
				continue
			}

			// All pieces have been received.
			hash := sha1.Sum(metadataBuf)

			if hash != p.metaInfo.InfoHash {
				return nil, fmt.Errorf("metadata hash mismatch: expected %x, got %x", p.metaInfo.InfoHash, hash)
			}

			return metadataBuf, nil

		case 2: // reject
			return nil, fmt.Errorf("peer rejected metadata piece %d", pieceIdx)

		default:
			log.Printf("unknown ut_metadata msg_type=%d", msgType)
		}
	}
}

func (p *Peer) ExtendedHandshake() error {
	extHandshakePayload, err := bencode.Encode(map[string]any{
		"m": map[string]any{
			// This is OUR ID for ut_metadata messages.
			"ut_metadata": int64(2),
		},
		"p": int64(p.metaInfo.AppPort),
	})
	if err != nil {
		return fmt.Errorf("extended handshake encoding failed: %w", err)
	}

	// Prefix the payload with Extended Message ID 0 (Handshake)
	extMsg := ExtendedMessage{
		ID:      0,
		Payload: extHandshakePayload,
	}

	msg := Message{
		ID:      MsgExtended,
		Payload: extMsg.Encode(),
	}

	if _, err := p.writeAll(msg.EncodeMessage()); err != nil {
		return fmt.Errorf("extended handshake failed: %w", err)
	}

	// log.Printf("%s: sent extended handshake", p.IP)
	return nil
}

func (p *Peer) sendMetaRequest(peerMetadataExtID uint8, index int) error {
	req := map[string]any{
		"msg_type": int64(0),
		"piece":    int64(index),
	}

	reqEncoded, err := bencode.Encode(req)
	if err != nil {
		return fmt.Errorf("encode metadata request: %w", err)
	}

	extMessage := ExtendedMessage{
		ID:      peerMetadataExtID,
		Payload: reqEncoded,
	}

	msg := Message{
		ID:      MsgExtended,
		Payload: extMessage.Encode(),
	}

	if _, err := p.writeAll(msg.EncodeMessage()); err != nil {
		return fmt.Errorf("metadata piece request failed: %w", err)
	}

	// log.Printf("metadata piece %d request sent", index)

	return nil
}

func handleExtendedHandshakeMessage(extMessage *ExtendedMessage) (uint8, int, error) {

	decodedMsg, err := bencode.NewDecoder(extMessage.Payload).Decode()
	if err != nil {
		return 0, 0, fmt.Errorf("decode extended handshake: %w", err)
	}

	data, ok := decodedMsg.(map[string]any)
	if !ok {
		return 0, 0, fmt.Errorf("extended handshake is not a dictionary: %T", decodedMsg)
	}

	metadataSizeValue, ok := data["metadata_size"].(int64)
	if !ok {
		return 0, 0, fmt.Errorf("extended handshake missing metadata_size: %v", data)
	}

	if metadataSizeValue <= 0 || metadataSizeValue > maxMetadataSize {
		return 0, 0, fmt.Errorf("invalid metadata_size: %d", metadataSizeValue)
	}

	m, ok := data["m"].(map[string]any)
	if !ok {
		return 0, 0, fmt.Errorf("extended handshake missing m: %v", data)
	}

	utMetadataValue, ok := m["ut_metadata"].(int64)
	if !ok {
		return 0, 0, fmt.Errorf("peer doesn't advertise ut_metadata: %v", m)
	}

	if utMetadataValue <= 0 || utMetadataValue > 255 {
		return 0, 0, fmt.Errorf("invalid ut_metadata extension ID: %d", utMetadataValue)
	}

	// log.Printf("peer ut_metadata=%d metadata_size=%d", utMetadataValue, metadataSizeValue)

	return uint8(utMetadataValue), int(metadataSizeValue), nil
}

func handleExtendedMessage(extMessage *ExtendedMessage) (pieceIdx int, data []byte, msgType int, err error) {

	bencodeEnd := findBencodedEnd(extMessage.Payload)
	if bencodeEnd < 0 {
		return 0, nil, 0, fmt.Errorf("couldn't find bencoded metadata header")
	}

	decodedMsg, err := bencode.NewDecoder(extMessage.Payload[:bencodeEnd]).Decode()
	if err != nil {
		return 0, nil, 0, fmt.Errorf("decode metadata message: %w", err)
	}

	d, ok := decodedMsg.(map[string]any)
	if !ok {
		return 0, nil, 0, fmt.Errorf("metadata message is not dictionary: %T", decodedMsg)
	}

	msgTypeValue, ok := d["msg_type"].(int64)
	if !ok {
		return 0, nil, 0, fmt.Errorf("metadata message missing msg_type: %v", d)
	}

	pieceValue, ok := d["piece"].(int64)
	if !ok {
		return 0, nil, 0, fmt.Errorf("metadata message missing piece: %v", d)
	}

	if pieceValue < 0 || pieceValue > int64(^uint(0)>>1) {
		return 0, nil, 0, fmt.Errorf("invalid piece index: %d", pieceValue)
	}

	// Everything after the bencoded dictionary is the raw metadata
	// piece data.
	data = extMessage.Payload[bencodeEnd:]

	return int(pieceValue), data, int(msgTypeValue), nil
}

func findBencodedEnd(data []byte) int {
	if len(data) == 0 || data[0] != 'd' {
		return -1
	}

	depth := 0
	i := 0

	for i < len(data) {
		switch data[i] {
		case 'd', 'l':
			depth++
			i++
		case 'i':
			// Skip over integers (i...e)
			i++
			for i < len(data) && data[i] != 'e' {
				i++
			}
			if i < len(data) && data[i] == 'e' {
				i++ // Skip the 'e'
			} else {
				return -1 // Malformed integer
			}
		case 'e':
			depth--
			i++
			if depth == 0 {
				return i
			}
			if depth < 0 {
				return -1
			}
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			// Read the string length
			strLen := 0
			for i < len(data) && data[i] >= '0' && data[i] <= '9' {
				strLen = strLen*10 + int(data[i]-'0')
				i++
			}

			// Validate colon separator
			if i >= len(data) || data[i] != ':' {
				return -1
			}
			i++ // Skip the colon

			// Skip the string bytes entirely
			i += strLen
		default:
			return -1 // Invalid bencode type
		}
	}

	return -1
}
