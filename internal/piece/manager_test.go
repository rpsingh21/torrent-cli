package piece

import (
	"crypto/rand"
	"log"
	"testing"
	"time"

	"github.com/rpsingh21/torrent-cli/internal/torrent"
	"github.com/rpsingh21/torrent-cli/pkg/bitfield"
)

func createMetainfo() *torrent.MetaInfo {
	hashes := make([][20]byte, 9)
	for i := range 9 {
		hashes[i] = calculate_peer_id()
	}

	return &torrent.MetaInfo{
		PieceHashes: hashes,
		PieceLength: 128 * 1024,
		TotalSize:   1024*1024 + 10*1024,
		TotalPices:  9,
	}
}

func CreatePeerBitfield() *bitfield.Bitfield {
	bf := bitfield.NewBitfield(9)
	bf.SetIndex(3)
	return bf
}

func calculate_peer_id() [20]byte {
	var peerID [20]byte
	rand.Read(peerID[:])
	return peerID
}

func TestManager(t *testing.T) {
	cxt := t.Context()
	metainfo := createMetainfo()
	manager := NewManager(cxt, metainfo)

	manager.AddPeerBitfield("test1", CreatePeerBitfield())
	b1, b2 := manager.NextBlock("test1"), manager.NextBlock("test2")

	if b1 == nil {
		t.Fatal("b1 Should not be nil.")
	}
	if b2 != nil {
		t.Fatal("b2 Shoud be nil.")
	}
	if b1.Requested != true || b1.Completed != false {
		t.Fatalf("b1 status wrong: %+v", b1)
	}

	time.Sleep(11 * time.Second)
	if b1.Requested != false {
		t.Fatal("b1 Should be reset to false")
	}
	log.Println("----------------------------")
}
