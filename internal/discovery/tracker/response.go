package tracker

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strconv"
)

type Response struct {
	Interval int
	Addrs    []string
}

var ErrorInvalidPeersData = errors.New("invalid peers data")

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

	addrs, err := parsePeers(v)
	if err != nil {
		return nil, err
	}

	return &Response{Interval: interval, Addrs: addrs}, nil
}

func parsePeers(v any) ([]string, error) {
	switch peers := v.(type) {
	case []byte:
		if len(peers)%6 == 0 {
			return parseIpv4Bytes(peers), nil
		} else if len(peers)%18 == 0 {
			return parseIpv6Bytes(peers), nil
		}
		return nil, ErrorInvalidPeersData

	case []any:
		return parseBpeers(peers)

	default:
		return nil, fmt.Errorf("invalid peers field type %T", v)
	}
}

func parseIpv4Bytes(buf []byte) []string {
	size := len(buf) / 6
	result := make([]string, 0, size)

	for i := range size {
		off := i * 6
		ip := net.IPv4(buf[off], buf[off+1], buf[off+2], buf[off+3]).String()
		port := binary.BigEndian.Uint16(buf[off+4 : off+6])
		result = append(result, net.JoinHostPort(ip, strconv.Itoa(int(port))))
	}

	return result
}

func parseIpv6Bytes(buf []byte) []string {
	size := len(buf) / 18
	result := make([]string, 0, size)

	for i := range size {
		off := i * 18
		ip := net.IP(buf[off : off+16]).String()
		port := uint16(binary.BigEndian.Uint16(buf[off+16 : off+18]))
		result = append(result, net.JoinHostPort(ip, strconv.Itoa(int(port))))
	}

	return result
}

func parseBpeers(peers []any) ([]string, error) {
	result := make([]string, 0, len(peers))
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
		port := strconv.Itoa(int(port64))

		result = append(result, net.JoinHostPort(string(ipBytes), port))
	}
	return result, nil

}
