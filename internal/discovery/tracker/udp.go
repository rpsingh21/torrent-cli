package tracker

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"time"

	"github.com/rpsingh21/torrent-cli/internal/torrent"
)

// https://www.bittorrent.org/beps/bep_0015.html

const protocolID = 0x41727101980

func AnnounceUPD(url string, metaInfo *torrent.MetaInfo) (*Response, error) {
	UDP_URL := strings.TrimPrefix(url, "udp://")

	if i := strings.Index(UDP_URL, "/"); i != -1 {
		UDP_URL = UDP_URL[:i]
	}

	conn, err := net.Dial("udp", UDP_URL)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	transactionID := int32(rand.Intn(1 << 31))

	connectReq := make([]byte, 16)
	binary.BigEndian.PutUint64(connectReq[:8], protocolID)
	binary.BigEndian.PutUint32(connectReq[8:12], 0) // Action connect
	binary.BigEndian.PutUint32(connectReq[12:16], uint32(transactionID))

	if _, err := conn.Write(connectReq); err != nil {
		return nil, err
	}

	buf := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(15 * time.Second))
	n, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}
	if n < 16 {
		return nil, fmt.Errorf("UDP: short response")
	}

	if binary.BigEndian.Uint32(buf[0:4]) != 0 {
		return nil, fmt.Errorf("UDP: unexpected action")
	}

	if binary.BigEndian.Uint32(buf[4:8]) != uint32(transactionID) {
		return nil, fmt.Errorf("UDP: transaction id mismatch")
	}

	connectionID := binary.BigEndian.Uint64(buf[8:16])

	// Create request to get peers
	announceReq := make([]byte, 98)
	binary.BigEndian.PutUint64(announceReq[0:8], connectionID)
	binary.BigEndian.PutUint32(announceReq[8:12], 1) // action = announce
	binary.BigEndian.PutUint32(announceReq[12:16], uint32(transactionID))
	copy(announceReq[16:36], metaInfo.InfoHash[:])
	copy(announceReq[36:56], metaInfo.AppId[:])
	// downloaded=0, left=0, uploaded=0 (offsets 56-80)
	binary.BigEndian.PutUint32(announceReq[80:84], 0)                        // event = none
	binary.BigEndian.PutUint32(announceReq[84:88], 0)                        // ip = 0 (use source)
	binary.BigEndian.PutUint32(announceReq[88:92], uint32(rand.Intn(1<<31))) // key
	binary.BigEndian.PutUint32(announceReq[92:96], 0xFFFFFFFF)               // num_want = -1
	binary.BigEndian.PutUint16(announceReq[96:98], metaInfo.AppPort)

	if _, err := conn.Write(announceReq); err != nil {
		return nil, err
	}

	conn.SetReadDeadline(time.Now().Add(15 * time.Second))
	n, err = conn.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("Announcr: %w", err)
	}

	if n < 20 {
		return nil, fmt.Errorf("announce: short response")
	}

	// Parse announce response
	action := binary.BigEndian.Uint32(buf[0:4])
	if action == 3 { // error
		msg := string(buf[16:])
		return nil, fmt.Errorf("tracker error: %s", msg)
	}
	if action != 1 {
		return nil, fmt.Errorf("announce: unexpected action %d", action)
	}

	interval := int(binary.BigEndian.Uint32(buf[8:12]))

	addrs := parseIpv4Bytes(buf[20:])

	response := &Response{
		Interval: interval,
		Addrs:    addrs,
	}

	return response, nil
}
