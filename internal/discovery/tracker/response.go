package tracker

import (
	"fmt"

	"github.com/rpsingh21/torrent-cli/internal/peer"
)

type Response struct {
	Interval int
	Peers    []*peer.Peer
}

func UnmarshalTrackerResponse(data any) (*Response, error) {
	root, ok := data.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("Recived invalid data %T", root)
	}
	interval, ok := root["interval"].(int64)
	if !ok {
		return nil, fmt.Errorf("Recived invalid data %T", root["interval"])
	}
	peersArr, ok := root["peers"].([]any)
	if !ok {
		return nil, fmt.Errorf("Recived invalid peers data %T, %+v", root, root["peers"])
	}

	peers := make([]*peer.Peer, len(peersArr))

	for i, peerel := range peersArr {
		peerMap, ok := peerel.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("failed to convert peer to map[string]any: %+v", peerel)
		}

		ipBytes, ok := peerMap["ip"].([]byte)
		if !ok {
			return nil, fmt.Errorf("failed to convert peer ip: %+v", peerMap)
		}

		port, ok := peerMap["port"].(int64)
		if !ok {
			return nil, fmt.Errorf("failed to convert port: %+v", peerMap["port"])
		}

		idBytes, ok := peerMap["peer id"].([]byte)
		if !ok {
			return nil, fmt.Errorf("failed to convert peer id: %+v", peerMap["peer id"])
		}

		peers[i] = &peer.Peer{
			ID:   string(idBytes),
			IP:   string(ipBytes),
			Port: uint16(port),
		}
	}

	return &Response{int(interval), peers}, nil
}
