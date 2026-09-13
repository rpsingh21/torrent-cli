package tracker

import (
	"fmt"
)

type Response struct {
	Interval int
	Peers    []peer
}

type peer struct {
	ID [20]byte
	// ID   string
	IP   string
	Port uint16
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

	peers := make([]peer, len(peersArr))
	// log.Printf("Got %d totals peers\n", len(peers))
	for i, peerel := range peersArr {
		peerMap, ok := peerel.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("Failed to convert peer tp map[string]: %+v", peerel)
		}
		// fmt.Printf("%d, peer data %+v\n", i, peerMap)
		ipBytes, ok := peerMap["ip"].([]byte)
		if !ok {
			return nil, fmt.Errorf("Failed to convert peer ip: %+v", peerMap)
		}
		peers[i].IP = string(ipBytes)
		// peers[i].IP = unsafe.String(unsafe.SliceData(ipBytes), len(ipBytes))
		// Don't use unfase because ip should be immutable

		port, ok := peerMap["port"].(int64)
		if !ok {
			return nil, fmt.Errorf("Failed to convert port unit16: %+v", peerMap)
		}
		peers[i].Port = uint16(port)

		idBytes, ok := peerMap["peer id"].([]byte)
		// if !ok {
		// 	return nil, fmt.Errorf("Failed to convert peer id: %+v", peerMap["peer id"])
		// }
		copy(peers[i].ID[:], idBytes)
		// peers[i].ID = unsafe.String(unsafe.SliceData(idBytes), len(idBytes))
	}

	return &Response{int(interval), peers}, nil
}
