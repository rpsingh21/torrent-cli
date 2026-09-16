package piece

// import (
// 	"testing"

// 	"github.com/rpsingh21/torrent-cli/pkg/bitfield"
// )

// func TestPickerSequential(t *testing.T) {
// 	p := NewPicker(6, bitfield.NewBitfield(6))
// 	bf := bitfield.NewBitfield(6)
// 	bf.SetIndex(1)
// 	bf.SetIndex(3)
// 	bf.SetIndex(5)
// 	p.AddPeer("peer", bf)

// 	for _, want := range []int{1, 3, 5, 1} {
// 		if got := p.Pick("peer"); got != want {
// 			t.Fatalf("got %d, want %d", got, want)
// 		}
// 	}
// }

// func TestPickerSkipsHave(t *testing.T) {
// 	have := bitfield.NewBitfield(5)
// 	have.SetIndex(1)
// 	p := NewPicker(5, have)

// 	bf := bitfield.NewBitfield(5)
// 	bf.SetIndex(0)
// 	bf.SetIndex(1)
// 	bf.SetIndex(2)
// 	p.AddPeer("peer", bf)

// 	if got := p.Pick("peer"); got != 0 {
// 		t.Fatalf("got %d, want 0", got)
// 	}
// 	have.SetIndex(0)
// 	if got := p.Pick("peer"); got != 2 {
// 		t.Fatalf("got %d, want 2", got)
// 	}
// }

// func TestPickerUnknownPeer(t *testing.T) {
// 	p := NewPicker(5, bitfield.NewBitfield(5))
// 	if got := p.Pick("unknown"); got != -1 {
// 		t.Fatalf("got %d, want -1", got)
// 	}
// }

// func TestPickerRarestFirst(t *testing.T) {
// 	p := NewPicker(5, bitfield.NewBitfield(5))
// 	p.Strategy = StrategyRarestFirst

// 	a := bitfield.NewBitfield(5)
// 	a.SetIndex(0); a.SetIndex(1); a.SetIndex(2)
// 	b := bitfield.NewBitfield(5)
// 	b.SetIndex(0); b.SetIndex(1)
// 	c := bitfield.NewBitfield(5)
// 	c.SetIndex(0)

// 	p.AddPeer("a", a)
// 	p.AddPeer("b", b)
// 	p.AddPeer("c", c)

// 	if got := p.Pick("a"); got != 2 {
// 		t.Fatalf("got %d, want rarest piece 2", got)
// 	}
// }

// func TestPickerRarestTie(t *testing.T) {
// 	p := NewPicker(4, bitfield.NewBitfield(4))
// 	p.Strategy = StrategyRarestFirst

// 	bf := bitfield.NewBitfield(4)
// 	bf.SetIndex(1)
// 	bf.SetIndex(3)
// 	p.AddPeer("peer", bf)

// 	if got := p.Pick("peer"); got != 1 {
// 		t.Fatalf("got %d, want 1", got)
// 	}
// }

// func TestPickerPeerLifecycle(t *testing.T) {
// 	p := NewPicker(4, bitfield.NewBitfield(4))

// 	a := bitfield.NewBitfield(4)
// 	a.SetIndex(0); a.SetIndex(1)
// 	b := bitfield.NewBitfield(4)
// 	b.SetIndex(0)

// 	p.AddPeer("a", a)
// 	p.AddPeer("b", b)

// 	if p.Availability[0] != 2 || p.Availability[1] != 1 {
// 		t.Fatalf("availability = %v", p.Availability)
// 	}

// 	p.UpdatePeer("a", b)
// 	if p.Availability[0] != 2 || p.Availability[1] != 0 {
// 		t.Fatalf("after update availability = %v", p.Availability)
// 	}

// 	p.RemovePeer("a")
// 	if p.Availability[0] != 1 {
// 		t.Fatalf("after remove availability = %v", p.Availability)
// 	}
// }

// func TestPickerEndGame(t *testing.T) {
// 	p := NewPicker(3, bitfield.NewBitfield(3))
// 	p.Strategy = StrategyEndGame

// 	bf := bitfield.NewBitfield(3)
// 	bf.SetIndex(0); bf.SetIndex(1); bf.SetIndex(2)
// 	p.AddPeer("peer", bf)

// 	if got := p.Pick("peer"); got != 0 {
// 		t.Fatalf("got %d, want 0", got)
// 	}
// }
