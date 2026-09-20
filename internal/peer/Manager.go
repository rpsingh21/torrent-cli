package peer

import (
	"context"
	"log"
	"net"
	"strconv"
	"sync"

	"github.com/rpsingh21/torrent-cli/internal/piece"
	"github.com/rpsingh21/torrent-cli/internal/torrent"
)

const (
	MAX_PEERS         = 100
	REQUESTS_PER_PEER = 32
	REQUEST_TIMEOUT   = 30
)

type Manager struct {
	activePeer     map[string]*Peer
	PeerChan       chan *Peer
	removePeerChan chan *Peer
	metaInfo       *torrent.MetaInfo
	pieceManager   *piece.Manager
	mu             sync.Mutex
	peersWG        sync.WaitGroup
}

func NewManager(metaInfo *torrent.MetaInfo, pieceManager *piece.Manager) *Manager {
	return &Manager{
		activePeer:     make(map[string]*Peer),
		PeerChan:       make(chan *Peer, 128),
		removePeerChan: make(chan *Peer, 128),
		metaInfo:       metaInfo,
		pieceManager:   pieceManager,
	}
}

func (m *Manager) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			m.Close()
			m.peersWG.Wait()
			return ctx.Err()
		case p := <-m.PeerChan:
			if p == nil {
				continue
			}
			m.addPeer(ctx, p)
		case p := <-m.removePeerChan:
			m.removePeer(p)
		}
	}
}

func (m *Manager) addPeer(ctx context.Context, p *Peer) {
	key := p.Key()
	m.mu.Lock()
	if _, exists := m.activePeer[key]; exists || len(m.activePeer) >= MAX_PEERS {
		m.mu.Unlock()
		return
	}
	p.metaInfo = m.metaInfo
	p.pieceManager = m.pieceManager
	p.removeChan = m.removePeerChan
	m.activePeer[key] = p
	m.mu.Unlock()

	m.peersWG.Add(1)
	go func() {
		defer m.peersWG.Done()
		if err := p.Start(ctx); err != nil && ctx.Err() == nil {
			log.Printf("peer %s failed: %v", key, err)
		}
		select {
		case m.removePeerChan <- p:
		case <-ctx.Done():
		}
	}()
}

func (m *Manager) removePeer(p *Peer) {
	if p == nil {
		return
	}
	key := p.Key()

	m.mu.Lock()
	if _, ok := m.activePeer[key]; ok {
		delete(m.activePeer, key)
	}
	m.mu.Unlock()

	m.pieceManager.RemovePeer(p.schedulerID())
}

func (m *Manager) Close() {
	m.mu.Lock()
	peers := make([]*Peer, 0, len(m.activePeer))
	for _, p := range m.activePeer {
		peers = append(peers, p)
	}
	m.mu.Unlock()

	for _, p := range peers {
		_ = p.Close()
	}
}

func (m *Manager) ActivePeers() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.activePeer)
}

func (m *Manager) peerKey(p *Peer) string {
	return net.JoinHostPort(p.IP, strconv.Itoa(int(p.Port)))
}
