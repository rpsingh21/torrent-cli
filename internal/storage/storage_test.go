package storage

import (
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/rpsingh21/torrent-cli/internal/torrent"
)

func TestFileStorage(t *testing.T) {
	epath, _ := os.Getwd()
	path := path.Join(epath, "../../testdata/torrents/test.torrent")
	metaInfo, err := torrent.NewTorrentDetailFromFile(path)
	if err != nil {
		t.Fatal(err)
	}

	storage, err := NewFileStorage(metaInfo, "./output")
	if err != nil {
		t.Fatal(err)
	}

	for i := range storage.offsets {
		var src []byte
		src = fmt.Appendf(src, "This is offset %v", i*int(metaInfo.PieceLength))
		storage.WritePiece(i, src)
	}

	storage.Close()
	os.RemoveAll("./output")
}
