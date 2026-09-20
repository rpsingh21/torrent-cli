package tracker

import (
	"encoding/binary"
	"fmt"
	"net"

	"github.com/rpsingh21/torrent-cli/internal/peer"
)

type Response struct {
	Interval int
	Peers    []*peer.Peer
}

func UnmarshalTrackerResponse(data any) (*Response, error) {
	root, ok := data.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid tracker response type %T", data)
	}

	if reason, ok := root["failure reason"].([]byte); ok {
		return nil, fmt.Errorf("tracker failure: %s", reason)
	}

	interval := 0
	if v, ok := root["interval"].(int64); ok && v > 0 {
		interval = int(v)
	}

	v, ok := root["peers"]
	if !ok {
		return nil, fmt.Errorf("tracker response has no peers field")
	}

	peers, err := parsePeers(v)
	if err != nil {
		return nil, err
	}
	return &Response{Interval: interval, Peers: peers}, nil
}

func parsePeers(v any) ([]*peer.Peer, error) {
	switch peers := v.(type) {
	case []byte:
		if len(peers)%6 != 0 {
			return nil, fmt.Errorf("invalid compact peers length %d", len(peers))
		}
		result := make([]*peer.Peer, 0, len(peers)/6)
		for i := 0; i < len(peers); i += 6 {
			ip := net.IPv4(peers[i], peers[i+1], peers[i+2], peers[i+3]).String()
			port := binary.BigEndian.Uint16(peers[i+4 : i+6])
			result = append(result, &peer.Peer{IP: ip, Port: port})
		}
		return result, nil

	case []any:
		result := make([]*peer.Peer, 0, len(peers))
		for _, item := range peers {
			m, ok := item.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("invalid peer entry type %T", item)
			}
			ipBytes, ok := m["ip"].([]byte)
			if !ok {
				return nil, fmt.Errorf("invalid peer ip %T", m["ip"])
			}
			port64, ok := m["port"].(int64)
			if !ok || port64 < 1 || port64 > 65535 {
				return nil, fmt.Errorf("invalid peer port %v", m["port"])
			}
			var id string
			if idBytes, ok := m["peer id"].([]byte); ok {
				id = string(idBytes)
			}
			result = append(
				result,
				&peer.Peer{ID: id, IP: string(ipBytes), Port: uint16(port64)})
		}
		return result, nil
	default:
		return nil, fmt.Errorf("invalid peers field type %T", v)
	}
}

// func peerKey(p *peer.Peer) string {
// 	return net.JoinHostPort(p.IP, strconv.Itoa(int(p.Port)))
// }
