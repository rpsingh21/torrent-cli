package piece

import (
	"context"
	"testing"

	"github.com/rpsingh21/torrent-cli/internal/torrent"
	"github.com/rpsingh21/torrent-cli/pkg/bitfield"
)

func benchmarkMetaInfo(pieceCount int) *torrent.MetaInfo {
	hashes := make([][20]byte, pieceCount)

	return &torrent.MetaInfo{
		PieceHashes: hashes,
		PieceLength: 128 * 1024,
		TotalSize:   int64(pieceCount * 128 * 1024),
		TotalPices:  pieceCount,
	}
}

func BenchmarkManagerNextBlock(b *testing.B) {
	for _, pieceCount := range []int{100, 1000, 10000} {
		b.Run("pieces_"+itoa(pieceCount), func(b *testing.B) {
			manager := NewManager(context.Background(), benchmarkMetaInfo(pieceCount))
			defer manager.Close()

			bf := bitfield.NewBitfield(pieceCount)
			bf.SetIndex(pieceCount - 1)
			manager.AddPeerBitfield("peer", bf)

			b.ReportAllocs()
			b.ResetTimer()

			for range b.N {
				block := manager.NextBlock("peer")
				if block == nil {
					// Reset the selected piece so the benchmark can continue.
					manager.ReDownloadPiece(pieceCount - 1)
					block = manager.NextBlock("peer")
				}

				if block == nil {
					b.Fatal("NextBlock returned nil")
				}

				// Reuse the same piece for the next iteration.
				block.Requested = false
			}
		})
	}
}

func BenchmarkPieceNextMissingBlock(b *testing.B) {
	piece := &Piece{
		Index:  0,
		Length: 128 * 1024,
		Blocks: buildBlocks(0, 128*1024),
	}

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		block := piece.NextMissingBlock()
		if block == nil {
			for i := range piece.Blocks {
				piece.Blocks[i].Requested = false
				piece.Blocks[i].Completed = false
			}
			block = piece.NextMissingBlock()
		}

		if block == nil {
			b.Fatal("NextMissingBlock returned nil")
		}

		block.Requested = true
	}
}

// Avoid strconv.FormatInt in the benchmark loop/name construction dependency.
func itoa(v int) string {
	switch v {
	case 100:
		return "100"
	case 1000:
		return "1000"
	case 10000:
		return "10000"
	default:
		return "unknown"
	}
}
