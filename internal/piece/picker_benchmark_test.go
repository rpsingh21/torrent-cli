package piece

// import (
// 	"testing"

// 	"github.com/rpsingh21/torrent-cli/pkg/bitfield"
// )

// func newBenchmarkPicker(n int, strategy PickStrategy) *Picker {
// 	p := NewPicker(n, bitfield.NewBitfield(n))
// 	p.Strategy = strategy
// 	bf := bitfield.NewBitfield(n)
// 	for i := range n {
// 		bf.SetIndex(i)
// 	}
// 	p.AddPeer("peer", bf)
// 	return p
// }

// func BenchmarkPickerSequential(b *testing.B) {
// 	for _, n := range []int{100, 1000, 10000} {
// 		b.Run("pieces_"+itoa(n), func(b *testing.B) {
// 			p := newBenchmarkPicker(n, StrategySequential)
// 			b.ReportAllocs()

// 			for b.Loop() {
// 				if p.Pick("peer") < 0 {
// 					b.Fatal("Pick returned -1")
// 				}
// 			}
// 		})
// 	}
// }

// func BenchmarkPickerRarestFirst(b *testing.B) {
// 	for _, n := range []int{100, 1000, 10000} {
// 		b.Run("pieces_"+itoa(n), func(b *testing.B) {
// 			p := newBenchmarkPicker(n, StrategyRarestFirst)
// 			b.ReportAllocs()

// 			for b.Loop() {
// 				if p.Pick("peer") < 0 {
// 					b.Fatal("Pick returned -1")
// 				}
// 			}
// 		})
// 	}
// }

// func BenchmarkPickerSequentialWorstCase(b *testing.B) {
// 	for _, n := range []int{100, 1000, 10000} {
// 		b.Run("pieces_"+itoa(n), func(b *testing.B) {
// 			p := NewPicker(n, bitfield.NewBitfield(n))

// 			peer := bitfield.NewBitfield(n)
// 			peer.SetIndex(n - 1)

// 			p.AddPeer("peer", peer)

// 			b.ReportAllocs()

// 			for b.Loop() {
// 				p.next = 0

// 				if p.Pick("peer") != n-1 {
// 					b.Fatal("unexpected piece")
// 				}
// 			}
// 		})
// 	}
// }

// // func itoa(n int) string {
// // 	switch n {
// // 	case 100:
// // 		return "100"
// // 	case 1000:
// // 		return "1000"
// // 	case 10000:
// // 		return "10000"
// // 	default:
// // 		return "unknown"
// // 	}
// // }
