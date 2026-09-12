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
	MetaInfo *torrent.TorrentMetaInfo
}

func NewTracker(metaInfo *torrent.TorrentMetaInfo) *Tracker {
	// peer_id := calculate_peer_id()

	return &Tracker{
		MetaInfo: metaInfo,
		PeerId:   calculate_peer_id(),
	}
}

func (t *Tracker) RequestPeers(event string) (*TrackerResponse, error) {
	url, err := t.MetaInfo.BuildTrackerURL(t.PeerId, event)
	if err != nil {
		return nil, err
	}
	fmt.Printf("URL Build by tracker = %v\n", url)
	c := &http.Client{Timeout: 30 * time.Second}
	resp, err := c.Get(url)
	if err != nil {
		fmt.Printf("Error: While calling url = %v\n", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("HTTP status: %d", resp.StatusCode)
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
	// for debug
	// if err := os.WriteFile("httpbody.test.data", body, 0664); err != nil {
	// 	log.Println(err)
	// }

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
