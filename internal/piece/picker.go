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
		delete(m.PeerPieces, peerID)
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

func (m *Manager) RemovePeer(peerID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.removeWithoutLock(peerID)
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
	start := m.next

	for i := range n {
		if m.canPick(peerpieces, i) {
			m.next = i + 1
			if m.next == n {
				m.next = 0
			}
			return i
		}
	}

	for i := range start {
		if m.canPick(peerpieces, i) {
			m.next = i + 1
			if m.next == n {
				m.next = 0
			}
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
