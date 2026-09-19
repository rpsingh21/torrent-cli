package peer

import (
	"log"

	"github.com/rpsingh21/torrent-cli/internal/piece"
	"github.com/rpsingh21/torrent-cli/internal/torrent"
)

// Load from config
const (
	MAX_PEERS         = 80
	REQUESTS_PER_PEER = 32
	REQUEST_TIMEOUT   = 5
)

type Manager struct {
	activePeer     map[string]*Peer
	PeerChan       chan *Peer
	removePeerChan chan *Peer
	metaInfo       *torrent.MetaInfo
	pieceManager   *piece.Manager
}

func NewManager(metaInfo *torrent.MetaInfo, pieceManager *piece.Manager) *Manager {
	return &Manager{
		activePeer:     make(map[string]*Peer),
		PeerChan:       make(chan *Peer, 10),
		removePeerChan: make(chan *Peer, 5),
		metaInfo:       metaInfo,
		pieceManager:   pieceManager,
	}
}

func (m *Manager) Run() {
	for {
		select {
		case peer := <-m.PeerChan:
			log.Printf("Adding new Peer %v : %v", peer.ID, peer.IP)
			peer.pieceManager = m.pieceManager
			m.AddPeer(peer)

		case peer := <-m.removePeerChan:
			m.removePeer(peer)
		}
	}
}

func (m *Manager) AddPeer(peer *Peer) {
	peer.removeChan = m.removePeerChan
	peer.metaInfo = m.metaInfo
	go peer.Start()

	m.activePeer[peer.ID] = peer
}

func (m *Manager) removePeer(peer *Peer) {
	m.pieceManager.RemovePeer(peer.ID)
	delete(m.activePeer, peer.ID)
}

func (m *Manager) Close() {
	for _, peer := range m.activePeer {
		peer.Close()
	}
}
