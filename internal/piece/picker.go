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

func (m *Manager) AddPeer(peerID string, peerpieces *bitfield.Bitfield) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if peerpieces == nil {
		m.removeWithoutLock(peerID)
		return
	}

	m.removeWithoutLock(peerID)
	m.PeerPieces[peerID] = peerpieces

	for i := range m.Availability {
		if peerpieces.Have(i) {
			m.Availability[i]++
		}
	}
}

func (m *Manager) PeerHasPiece(peerID string, index int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	pieces := m.PeerPieces[peerID]
	if pieces == nil || index < 0 || index >= len(m.Availability) || pieces.Have(index) {
		return
	}
	pieces.SetIndex(index)
	m.Availability[index]++
}

func (m *Manager) removeWithoutLock(peerID string) {
	peerpieces, ok := m.PeerPieces[peerID]
	if !ok {
		return
	}

	for i := range m.Availability {
		if peerpieces.Have(i) && m.Availability[i] > 0 {
			m.Availability[i]--
		}
	}

	delete(m.PeerPieces, peerID)
}

func (m *Manager) UpdatePeer(peerID string, pieces *bitfield.Bitfield) {
	m.AddPeer(peerID, pieces)
}

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
	start := m.next
	for step := range n {
		i := (start + step) % n
		if m.canPick(peerpieces, i) {
			m.next = i
			return i
		}
	}

	return -1
}

func (m *Manager) rarestFirst(peerID string) int {
	peerpieces := m.PeerPieces[peerID]
	if peerpieces == nil || len(m.Availability) == 0 {
		return -1
	}
	best, bestAvailability := -1, ^uint32(0)
	for i, availability := range m.Availability {
		if m.canPick(peerpieces, i) && availability < bestAvailability {
			best, bestAvailability = i, availability
		}
	}
	return best
}

// End-game duplicate block requests belong in the block/request scheduler.
// At the piece level, use rarest-first selection for now.
func (m *Manager) endGame(peerID string) int {
	return m.rarestFirst(peerID)
}

// Todo Review m.Pieces[index].NextMissingBlock() != nil
func (m *Manager) canPick(peerpieces *bitfield.Bitfield, index int) bool {
	return peerpieces.Have(index) && !m.Have.Have(index) && !m.Pieces[index].Verifying && m.Pieces[index].NextMissingBlock() != nil
}
