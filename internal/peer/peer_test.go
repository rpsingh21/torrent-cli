package peer

import (
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
		addr     string
		isFailed bool
	}{
		{"87.179.14.132:57434", true},
		{"147.90.209.90:33622", true},
	}

	for _, tf := range peers {
		t.Run(tf.addr, func(t *testing.T) {
			peer := NewPeer(tf.addr, tf.addr, metaInfo)
			if _, err := peer.Handshake(); err != nil {
				t.Fatalf("peer: %v handshake failed", peer.Addr)
			}
			peer.Close()
		})
	}
}
