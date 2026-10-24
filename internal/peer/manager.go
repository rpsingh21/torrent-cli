package peer

import (
	"context"
	"encoding/hex"
	"log"
	"sync"
	"time"

	"github.com/rpsingh21/torrent-cli/internal/piece"
	"github.com/rpsingh21/torrent-cli/internal/torrent"
)

const (
	MAX_PEERS             = 1000
	MAX_REQUESTS_PER_PEER = 128
	MIN_REQUESTS_PER_PEER = 16
	REQUEST_TIMEOUT       = 30 * time.Second
	KEEPALIVE_TIMEOUT     = 2 * time.Minute
	MAX_MESSAGE_LENGTH    = 2 << 20
)

type Manager struct {
	Id           string
	PeerChan     chan *Peer
	activePeer   map[string]*Peer
	metaInfo     *torrent.MetaInfo
	pieceManager *piece.Manager
	mu           sync.Mutex
	refreshMap   map[string]time.Duration
	MStat        *ManagerStat
	cancel       context.CancelFunc
}

func NewManager(metaInfo *torrent.MetaInfo, pieceManager *piece.Manager) *Manager {
	return &Manager{
		Id:           hex.EncodeToString(metaInfo.InfoHash[:]),
		activePeer:   make(map[string]*Peer),
		PeerChan:     make(chan *Peer, 128),
		metaInfo:     metaInfo,
		pieceManager: pieceManager,
		refreshMap:   make(map[string]time.Duration),
		MStat:        NewManagerStat(metaInfo),
	}
}

func (m *Manager) Run(parentCtx context.Context) error {
	ctx, cancel := context.WithCancel(parentCtx)
	m.mu.Lock()
	m.cancel = cancel
	m.mu.Unlock()
	defer cancel()

	var wg sync.WaitGroup

	for {
		select {
		case <-ctx.Done():
			m.closeActivePeers()
			wg.Wait()
			return ctx.Err()

		case p := <-m.PeerChan:
			if p == nil {
				continue
			}
			m.addPeer(ctx, p, &wg)
		}
	}
}

func (m *Manager) UpdateState() ManagerStatSnapshot {
	completed, inprogress := m.pieceManager.GetStat()

	m.mu.Lock()
	preSnapshot := make(map[string]StatSnapshot, len(m.activePeer))
	for _, v := range m.activePeer {
		preSnapshot[v.Addr] = v.stat.Snapshot()
	}
	m.MStat.updatePeerStatMap(preSnapshot, completed, max(0, inprogress-completed))
	m.mu.Unlock()

	return m.MStat.Snapshot()
}

func (m *Manager) addPeer(ctx context.Context, p *Peer, wg *sync.WaitGroup) {
	m.mu.Lock()
	if _, exists := m.activePeer[p.Addr]; exists || len(m.activePeer) >= MAX_PEERS {
		m.mu.Unlock()
		return
	}
	p.pieceManager = m.pieceManager
	m.activePeer[p.Addr] = p
	m.mu.Unlock()

	wg.Go(func() {
		err := p.Start(ctx)
		p.Close()

		// Capture final snapshot once before removing and potentially resetting
		finalSnap := p.stat.Snapshot()
		if err != nil && ctx.Err() == nil {
			log.Printf("Peer %s failed: %v, Downloaded = %v KiB", p.Addr, err.Error(), finalSnap.Downloaded/1024)
		}

		m.removePeer(p, finalSnap)

		if ctx.Err() == nil && finalSnap.Downloaded > 0 && !p.pieceManager.Completed() {
			wg.Go(func() {
				m.scheduleRefreshDeadPeer(ctx, p)
			})
		}
	})
}

func (m *Manager) scheduleRefreshDeadPeer(ctx context.Context, p *Peer) {
	m.mu.Lock()
	d, ok := m.refreshMap[p.Addr]
	if !ok {
		d = 1 * time.Minute
	}
	m.refreshMap[p.Addr] = d << 1
	m.mu.Unlock()

	if d > 16*time.Minute {
		log.Printf("peer %v refresh time out", p.Addr)
		return
	}

	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return
	case <-timer.C:
		if m.pieceManager.Completed() {
			return
		}
		log.Printf("reconnecting peer %v after %v", p.Addr, d)
		p.stat.Reset() // Prevent double-counting previous session's bytes

		select {
		case m.PeerChan <- p:
		case <-ctx.Done():
		}
	}
}

func (m *Manager) removePeer(p *Peer, finalSnap StatSnapshot) {
	if p == nil {
		return
	}

	m.mu.Lock()
	delete(m.activePeer, p.Addr)
	m.MStat.removePeerStat(p.Addr, finalSnap)
	m.mu.Unlock()

	m.pieceManager.RemovePeer(p.Addr)
}

func (m *Manager) closeActivePeers() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, p := range m.activePeer {
		p.Close()
	}
}

func (m *Manager) Close() {
	m.mu.Lock()
	cancel := m.cancel
	m.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	m.closeActivePeers()
}

func (m *Manager) ActivePeers() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.activePeer)
}
