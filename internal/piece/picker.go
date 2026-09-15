package piece

import (
	"github.com/rpsingh21/torrent-cli/pkg/bitfield"
)

type PickStrategy uint8

const (
	StrategySequential PickStrategy = iota
	StrategyRarestFirst
	StrategyEndGame
)

// type Picker struct {
// 	PeerPieces   map[string]*bitfield.Bitfield
// 	Have         *bitfield.Bitfield
// 	Availability []uint32
// 	Strategy     PickStrategy
// 	next         int
// }

// func NewPicker(totalPieces int, have *bitfield.Bitfield) *Picker {
// 	return &Picker{
// 		PeerPieces:   make(map[string]*bitfield.Bitfield),
// 		Have:         have,
// 		Availability: make([]uint32, totalPieces),
// 	}
// }

func (m *Manager) AddPeer(peerID string, pieces *bitfield.Bitfield) {
	if pieces == nil {
		delete(m.PeerPieces, peerID)
		return
	}
	m.RemovePeer(peerID)

	m.mu.Lock()
	defer m.mu.Unlock()

	m.PeerPieces[peerID] = pieces

	for i := range m.Availability {
		if pieces.Have(i) {
			m.Availability[i]++
		}
	}
}

func (m *Manager) RemovePeer(peerID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	pieces, ok := m.PeerPieces[peerID]
	if !ok {
		return
	}

	for i := range m.Availability {
		if pieces.Have(i) && m.Availability[i] > 0 {
			m.Availability[i]--
		}
	}

	delete(m.PeerPieces, peerID)
}

func (m *Manager) UpdatePeer(peerID string, pieces *bitfield.Bitfield) {
	m.AddPeer(peerID, pieces)
}

// Locked on manager level
func (m *Manager) Pick(peerID string) int {
	switch m.Strategy {
	case StrategyRarestFirst:
		return m.rarestFirst(peerID)
	case StrategyEndGame:
		return m.endGame(peerID)
	default:
		return m.sequential(peerID)
	}
}

func (m *Manager) sequential(peerID string) int {
	peerpieces := m.PeerPieces[peerID]
	if peerpieces == nil || len(m.Availability) == 0 {
		return -1
	}

	n := len(m.Availability)
	for checked := range n {
		index := (m.next + checked) % n
		if m.canPick(peerpieces, index) {
			m.next = (index + 1) % n
			return index
		}
	}
	return -1
}

func (m *Manager) rarestFirst(peerID string) int {
	peerpieces := m.PeerPieces[peerID]
	if peerpieces == nil || len(m.Availability) == 0 {
		return -1
	}

	best := -1
	bestAvailability := ^uint32(0)
	for i, availability := range m.Availability {
		if !m.canPick(peerpieces, i) {
			continue
		}
		if availability < bestAvailability {
			best = i
			bestAvailability = availability
		}
	}
	return best
}

// End-game duplicate block requests belong in the block/request scheduler.
// At the piece level, use rarest-first selection for now.
func (m *Manager) endGame(peerID string) int {
	return m.rarestFirst(peerID)
}

func (m *Manager) canPick(peerpieces *bitfield.Bitfield, index int) bool {
	return peerpieces.Have(index) && !m.Have.Have(index)
}
