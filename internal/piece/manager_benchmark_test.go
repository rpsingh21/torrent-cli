package piece

import (
	"fmt"
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

func peerWith10CentBits(pieceCount int) *bitfield.Bitfield {
	peerBf := bitfield.NewBitfield(pieceCount)
	peerBf.SetIndex(0)
	for i := 9; i < pieceCount; i += 10 {
		peerBf.SetIndex(i)
	}
	return peerBf
}

func BenchmarkManagerNextBlock(b *testing.B) {
	for _, st := range strategies {
		for _, pieceCount := range []int{100, 1000, 10000, 100000} {
			testName := fmt.Sprintf("pieces_%v_%d", st.name, pieceCount)

			b.Run(testName, func(b *testing.B) {

				manager := NewManager(
					benchmarkMetaInfo(pieceCount),
					st.strategy,
					nil,
				)

				bf := bitfield.NewBitfield(pieceCount)
				bf.SetIndex(pieceCount - 1)
				manager.AddPeer("peer", bf)

				b.ReportAllocs()

				for b.Loop() {
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
}

func BenchmarkManager10Cent(b *testing.B) {
	for _, st := range strategies {

		for _, pieceCount := range []int{100, 1000, 10000, 100000} {
			testName := fmt.Sprintf("pieces_10C_%v_%d", st.name, pieceCount*10)

			b.Run(testName, func(b *testing.B) {
				manager := NewManager(
					benchmarkMetaInfo(pieceCount),
					st.strategy,
					nil,
				)

				bf := peerWith10CentBits(pieceCount * 10)
				manager.AddPeer("peer", bf)

				b.ReportAllocs()

				for b.Loop() {
					block := manager.NextBlock("peer")

					if block == nil {
						b.Fatal("NextBlock returned nil")
					}

					// Reuse the same piece for the next iteration.
					block.Requested = false
				}
			})
		}
	}
}

func BenchmarkPieceNextMissingBlock(b *testing.B) {
	piece := &Piece{
		Index:  0,
		Length: 128 * 1024,
		Blocks: buildBlocks(0, 128*1024),
	}

	b.ReportAllocs()

	for b.Loop() {
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
