package torrent

import (
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"testing"
)

func TestTorrentFile(t *testing.T) {
	epath, _ := os.Getwd()
	path := path.Join(epath, "../../testdata/torrents/test.torrent")
	nt, err := NewTorrentDetailFromFile(path)
	if err != nil || nt.Announce == "" {
		t.Errorf("Ancounter error while decoding torrent file %v", err)
	}
	fmt.Println(hex.EncodeToString(nt.InfoHash[:]))
}

func TestSingleTorrentFile(t *testing.T) {
	myPeerId := [20]byte{73, 158, 15, 108, 202, 74, 147, 78, 115, 126, 145, 49, 204, 11, 11, 39, 41, 16, 136, 183}
	epath, _ := os.Getwd()
	path := path.Join(epath, "../../testdata/torrents/singlefile.torrent")
	nt, err := NewTorrentDetailFromFile(path)
	if err != nil || nt.Announce == "" {
		t.Errorf("Ancounter error while decoding torrent file %v", err)
	}
	if _, err := nt.BuildTrackerURL(myPeerId, "start"); err != nil {
		t.Error("Fail to build tracker url", err)
	}
	if _, err := nt.BuildTrackerURL(myPeerId, ""); err != nil {
		t.Error("Faild to build URL", err)
	}
}

var benchmarkResult *TorrentMetaInfo

func BenchmarkTorrentMetaInfoFromFile(b *testing.B) {
	epath, err := os.Getwd()
	if err != nil {
		b.Fatal(err)
	}

	tests := []struct {
		name string
		path string
	}{
		{
			name: "MultiFiles",
			path: path.Join(epath, "../../testdata/torrents/test.torrent"),
		},
		{
			name: "Ubuntu",
			path: path.Join(epath, "../../testdata/torrents/ubuntu-26.04.1.torrent"),
		},
		{
			name: "SingleFile",
			path: path.Join(epath, "../../testdata/torrents/singlefile.torrent"),
		},
		{
			name: "Arch",
			path: path.Join(epath, "../../testdata/torrents/archlinux-2026.09.01.torrent"),
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			info, err := os.Stat(tt.path)
			if err != nil {
				b.Fatal(err)
			}

			b.SetBytes(info.Size())
			b.ReportAllocs()

			for b.Loop() {
				result, err := NewTorrentDetailFromFile(tt.path)
				if err != nil {
					b.Fatal(err)
				}

				benchmarkResult = result
			}
		})
	}
}
