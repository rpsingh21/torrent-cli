package peer

import (
	"log"
	"testing"

	"github.com/rpsingh21/torrent-cli/internal/torrent"
)

func TestCompleteHandshake(t *testing.T) {
	myPeerId := [20]byte{73, 158, 15, 108, 202, 74, 147, 78, 115, 126, 145, 49, 204, 11, 11, 39, 41, 16, 136, 183}
	infoHash := [20]byte{217, 132, 246, 122, 249, 145, 123, 33, 76, 216, 182, 4, 138, 181, 98, 76, 125, 246, 160, 122}
	metaInfo := &torrent.MetaInfo{
		AppId:    myPeerId,
		InfoHash: infoHash,
	}

	peers := []struct {
		ip       string
		port     uint16
		isFailed bool
	}{

		{"87.179.14.132", 57434, true},
		// {"147.90.209.90", 33622, true},
		// {"147.90.227.68", 49987, true},
		// {"147.90.209.164", 30363, true},
		// {"146.70.100.165", 51413, false},
		// {"103.108.231.238", 47416, false},
	}

	for _, tf := range peers {
		t.Run(tf.ip, func(t *testing.T) {
			peer := NewPeer(tf.ip, tf.ip, tf.port, metaInfo, nil)

			err := peer.Start()
			// time.Sleep(5 * time.Second)

			if !tf.isFailed && err != nil {
				t.Fatal(err)
			}
			if !tf.isFailed {
				log.Printf("%v Connected successfully", peer.IP)
			}
			peer.Close()
		})
	}
}
