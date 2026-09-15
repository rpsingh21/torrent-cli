package piece

import (
	"bytes"
	"context"
	"crypto/sha1"
	"testing"
	"time"

	"github.com/rpsingh21/torrent-cli/internal/torrent"
	"github.com/rpsingh21/torrent-cli/pkg/bitfield"
)

var strategies = []struct {
	name     string
	strategy PickStrategy
}{
	{
		name:     "Sequential",
		strategy: StrategySequential,
	},
	{
		name:     "RarestFirst",
		strategy: StrategyRarestFirst,
	},
}

func testMetaInfo() *torrent.MetaInfo {
	hashes := make([][20]byte, 9)

	return &torrent.MetaInfo{
		PieceHashes: hashes,
		PieceLength: 128 * 1024,
		TotalSize:   1024*1024 + 10*1024,
		TotalPices:  9,
	}
}

func peerWithPiece(pieceIndex int) *bitfield.Bitfield {
	bf := bitfield.NewBitfield(9)
	bf.SetIndex(pieceIndex)
	return bf
}

func TestBuildBlocks(t *testing.T) {
	tests := []struct {
		name      string
		pieceID   int
		pieceSize int
		want      []Block
	}{
		{
			name:      "full piece",
			pieceID:   0,
			pieceSize: 128 * 1024,
			want: []Block{
				{Piece: 0, Offset: 0, Length: 16 * 1024},
				{Piece: 0, Offset: 16 * 1024, Length: 16 * 1024},
				{Piece: 0, Offset: 32 * 1024, Length: 16 * 1024},
				{Piece: 0, Offset: 48 * 1024, Length: 16 * 1024},
				{Piece: 0, Offset: 64 * 1024, Length: 16 * 1024},
				{Piece: 0, Offset: 80 * 1024, Length: 16 * 1024},
				{Piece: 0, Offset: 96 * 1024, Length: 16 * 1024},
				{Piece: 0, Offset: 112 * 1024, Length: 16 * 1024},
			},
		},
		{
			name:      "last partial piece",
			pieceID:   8,
			pieceSize: 10 * 1024,
			want: []Block{
				{Piece: 8, Offset: 0, Length: 10 * 1024},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildBlocks(tt.pieceID, tt.pieceSize)

			if len(got) != len(tt.want) {
				t.Fatalf("got %d blocks, want %d", len(got), len(tt.want))
			}

			for i := range got {
				if got[i].Piece != tt.want[i].Piece ||
					got[i].Offset != tt.want[i].Offset ||
					got[i].Length != tt.want[i].Length {
					t.Fatalf("block %d = %+v, want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestManagerNextBlock(t *testing.T) {
	for _, tt := range strategies {
		t.Run(tt.name, func(t *testing.T) {

			manager := NewManager(
				context.Background(),
				testMetaInfo(),
				tt.strategy,
			)
			defer manager.Close()

			manager.AddPeer("test1", peerWithPiece(3))

			b1 := manager.NextBlock("test1")
			if b1 == nil {
				t.Fatal("b1 should not be nil")
			}

			if b1.Piece != 3 {
				t.Fatalf("piece = %d, want 3", b1.Piece)
			}

			if !b1.Requested || b1.Completed {
				t.Fatalf("unexpected block state: %+v", b1)
			}

			// Unknown peer must not panic and must return nil.
			if b2 := manager.NextBlock("test2"); b2 != nil {
				t.Fatalf("unknown peer returned block: %+v", b2)
			}

			// Same peer gets the next block, not the same requested block.
			b2 := manager.NextBlock("test1")
			if b2 == nil {
				t.Fatal("second block should not be nil")
			}

			if b2.Offset == b1.Offset {
				t.Fatalf("same block returned twice: b1=%+v b2=%+v", b1, b2)
			}
		})
	}
}

func TestManagerCleanupExpiredBlocks(t *testing.T) {
	for _, tt := range strategies {
		t.Run(tt.name, func(t *testing.T) {

			manager := NewManager(
				context.Background(),
				testMetaInfo(), tt.strategy,
			)
			defer manager.Close()

			manager.AddPeer("peer", peerWithPiece(0))

			block := manager.NextBlock("peer")
			if block == nil {
				t.Fatal("expected block")
			}

			start := block.startedAt

			// Not expired yet.
			cleaned := manager.cleanupExpiredBlocks(start.Add(BLOCK_TIMEOUT - time.Nanosecond))
			if cleaned != 0 {
				t.Fatalf("cleaned %d blocks before timeout", cleaned)
			}

			if !block.Requested {
				t.Fatal("block was reset before timeout")
			}

			// Expired.
			cleaned = manager.cleanupExpiredBlocks(start.Add(BLOCK_TIMEOUT))
			if cleaned != 1 {
				t.Fatalf("cleaned %d blocks, want 1", cleaned)
			}

			if block.Requested {
				t.Fatal("expired block should no longer be requested")
			}
		})
	}
}

func TestManagerCompleteBlock(t *testing.T) {
	for _, tt := range strategies {
		t.Run(tt.name, func(t *testing.T) {

			manager := NewManager(
				context.Background(),
				testMetaInfo(), tt.strategy,
			)
			defer manager.Close()

			data := bytes.Repeat([]byte("A"), REQUEST_SIZE)
			hash := sha1.Sum(data)
			manager.Pieces[0].HashV1 = hash

			manager.AddPeer("peer", peerWithPiece(0))

			block := manager.NextBlock("peer")
			if block == nil {
				t.Fatal("expected block")
			}

			if !manager.CompleteBlock(0, block.Offset, data) {
				t.Fatal("CompleteBlock should succeed")
			}

			if !block.Completed {
				t.Fatal("block should be completed")
			}

			if block.Requested {
				t.Fatal("completed block should not remain requested")
			}

			if !bytes.Equal(block.Data, data) {
				t.Fatal("block data mismatch")
			}
		})
	}
}

func TestManagerCompletePiece(t *testing.T) {

	for _, tt := range strategies {
		t.Run(tt.name, func(t *testing.T) {

			manager := NewManager(
				context.Background(),
				testMetaInfo(), tt.strategy,
			)
			defer manager.Close()

			// Use a small custom piece so the test does not need 128 KiB of data.
			data := []byte("hello torrent")
			hash := sha1.Sum(data)

			manager.Pieces[0].Length = len(data)
			manager.Pieces[0].HashV1 = hash
			manager.Pieces[0].Blocks = []Block{
				{
					Piece:  0,
					Offset: 0,
					Length: len(data),
				},
			}

			manager.AddPeer("peer", peerWithPiece(0))

			block := manager.NextBlock("peer")
			if block == nil {
				t.Fatal("expected block")
			}

			if !manager.CompleteBlock(0, 0, data) {
				t.Fatal("CompleteBlock should succeed")
			}

			if !manager.CompletePiece(0) {
				t.Fatal("CompletePiece should succeed")
			}

			if !manager.IsComplete(0) {
				t.Fatal("piece should be complete")
			}

			if !manager.Have.Have(0) {
				t.Fatal("manager Have bitfield should contain piece 0")
			}
		})
	}
}

func TestManagerReDownloadPiece(t *testing.T) {
	for _, tt := range strategies {
		t.Run(tt.name, func(t *testing.T) {

			manager := NewManager(
				context.Background(),
				testMetaInfo(), tt.strategy,
			)
			defer manager.Close()

			manager.AddPeer("peer", peerWithPiece(0))

			block := manager.NextBlock("peer")
			if block == nil {
				t.Fatal("expected block")
			}

			block.Data = []byte("bad")
			block.Completed = true

			if !manager.ReDownloadPiece(0) {
				t.Fatal("ReDownloadPiece should succeed")
			}

			if block.Requested || block.Completed || block.Data != nil {
				t.Fatalf("block was not reset: %+v", block)
			}
		})
	}
}

func TestManagerConcurrentNextBlock(t *testing.T) {
	for _, tt := range strategies {
		t.Run(tt.name, func(t *testing.T) {

			manager := NewManager(
				context.Background(),
				testMetaInfo(), tt.strategy,
			)
			defer manager.Close()

			manager.AddPeer("peer", peerWithPiece(0))

			const callers = 8
			results := make(chan *Block, callers)

			for range callers {
				go func() {
					results <- manager.NextBlock("peer")
				}()
			}

			seen := make(map[int]bool)
			for range callers {
				block := <-results
				if block == nil {
					t.Fatal("unexpected nil block")
				}

				if seen[block.Offset] {
					t.Fatalf("duplicate block returned at offset %d", block.Offset)
				}
				seen[block.Offset] = true
			}
		})
	}
}
