package piece

import (
	"github.com/rpsingh21/torrent-cli/internal/peer"
)

type Picker struct {
	Have  peer.Bitfield
	Peers map[string]*peer.Peer
}

func NewPicker(totalPiece int) *Picker {
	return &Picker{
		Have: peer.NewBitfield(totalPiece),
	}
}

// func (p *Picker) Pick(peer *peer.Peer) (int, bool) {

// }

func (p *Picker) Complete() bool {
	return p.Have.AllSet()
}

// func (p *Picker) rarestFirst()
// func (p *Picker) sequential()
// func (p *Picker) endGame()
