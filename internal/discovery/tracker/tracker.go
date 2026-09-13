package tracker

import (
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/rpsingh21/torrent-cli/internal/bencode"
	"github.com/rpsingh21/torrent-cli/internal/torrent"
)

type Tracker struct {
	PeerId   [20]byte
	MetaInfo *torrent.MetaInfo
	client   *http.Client
}

func NewTracker(metaInfo *torrent.MetaInfo) *Tracker {
	return &Tracker{
		MetaInfo: metaInfo,
		PeerId:   calculate_peer_id(),
	}
}

func (t *Tracker) RequestPeers(event string) (*Response, error) {
	url, err := t.MetaInfo.BuildTrackerURL(t.PeerId, event)
	if err != nil {
		return nil, err
	}

	if t.client == nil {
		log.Printf("Creating new client %v,", t.MetaInfo.Announce)
		t.client = &http.Client{Timeout: 15 * time.Second}
	}
	resp, err := t.client.Get(url)
	if err != nil {
		log.Fatalf("Error: While calling url = %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("HTTP status: %d", resp.StatusCode)
		return nil, fmt.Errorf("Http status: %v (%v)", resp.StatusCode, t.MetaInfo.Announce)
	}
	log.Printf("HTTP status: %d", resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("HTTP status: %d", resp.StatusCode)

	respData, err := bencode.NewDecoder(body).Decode()
	if err != nil {
		fmt.Printf("Error: resp convering %v = %+v \n", err, body)
		return nil, err
	}

	tresp, err := UnmarshalTrackerResponse(respData)
	if err != nil {
		return nil, err
	}
	return tresp, nil
}

func calculate_peer_id() [20]byte {
	var peerID [20]byte
	rand.Read(peerID[:])
	return peerID
}
