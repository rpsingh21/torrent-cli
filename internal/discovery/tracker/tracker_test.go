package tracker

import (
	"fmt"
	"log"
	"os"
	"path"
	"testing"

	"github.com/rpsingh21/torrent-cli/internal/torrent"
)

func TestTracker(t *testing.T) {
	epath, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		path string
	}{
		{
			name: "MultiFiles",
			path: path.Join(epath, "../../../testdata/torrents/test.torrent"),
		},
		// {
		// 	name: "Ubuntu",
		// 	path: path.Join(epath, "../../../testdata/torrents/ubuntu-26.04.1.torrent"),
		// },
		// {
		// 	name: "SingleFile",
		// 	path: path.Join(epath, "../../../testdata/torrents/singlefile.torrent"),
		// },
		// {
		// 	name: "Arch",
		// 	path: path.Join(epath, "../../../testdata/torrents/archlinux-2026.09.01.torrent"),
		// },
	}

	for _, tf := range tests {
		t.Run(tf.name, func(t *testing.T) {
			metaInfo, err := torrent.NewTorrentDetailFromFile(tf.path)
			if err != nil {
				t.Fatal(err)
			}
			tracker := NewTracker(metaInfo)
			tarckerResp, err := tracker.RequestPeers("started")
			if err != nil {
				fmt.Printf("Error => %+v\n", err)
				t.Fatal(err)
			}
			log.Printf("My PeerId %+v", tracker.PeerId)
			log.Printf("InfoHash %+v", metaInfo.InfoHash)
			log.Printf("TrackerResponse = %+v", tarckerResp)
		})
	}
}
