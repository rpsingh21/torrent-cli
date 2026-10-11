package storage

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"os"
	"path"
	"path/filepath"
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

	storage, err := NewFileStorage(metaInfo, "./output"+"/"+metaInfo.Name)
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

func TestReadPiece(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working dir: %v", err)
	}

	torrentPath := filepath.Join(wd, "..", "..", "testdata", "torrents", "test.torrent")
	metaInfo, err := torrent.NewTorrentDetailFromFile(torrentPath)
	if err != nil {
		t.Fatalf("failed to parse torrent file: %v", err)
	}

	storage, err := NewFileStorage(metaInfo, "../../output"+"/"+metaInfo.Name)
	if err != nil {
		t.Fatalf("failed to initialize file storage: %v", err)
	}

	for i, f := range storage.files {
		fi, err := f.Stat()
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("File %d: name=%s, size=%d bytes", i, fi.Name(), fi.Size())
	}

	pieceLen := metaInfo.PieceLength
	totalSize := metaInfo.TotalSize

	for pID, expectedHash := range metaInfo.PieceHashes {
		t.Run(fmt.Sprintf("Piece_%d", pID), func(t *testing.T) {
			offset := int64(pID) * pieceLen
			length := min(pieceLen, totalSize-offset)

			buf := make([]byte, length)
			if err := storage.ReadPiece(pID, buf); err != nil {
				t.Fatalf("ReadPiece(%d) failed: %v", pID, err)
			}

			actualHash := sha1.Sum(buf)
			if !bytes.Equal(actualHash[:], expectedHash[:]) {
				t.Fatalf("hash mismatch for piece %d: got %x, want %x", pID, actualHash, expectedHash)
			}
		})
	}
}
