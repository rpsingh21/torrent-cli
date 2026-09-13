package peer

import (
	"log"
	"time"
)

type Manager struct {
	activePeer map[string]*Peer
	peerChan   chan []Peer
	infohash   [20]byte
	myid       [20]byte
}

func NewManager(infoHash, myid [20]byte) *Manager {
	return &Manager{
		activePeer: make(map[string]*Peer),
		peerChan:   make(chan []Peer, 50),
		infohash:   infoHash,
		myid:       myid,
	}
}

func (m *Manager) Run() {
	tiker := time.NewTicker(5 * time.Second)
	go func() {
		for t := range tiker.C {
			log.Println("Tiker at ", t)
			for key := range m.activePeer {
				log.Println("Peer ", m.activePeer[key].ID)
				// if m.activePeer[key].Connection.conn.
			}
		}
	}()
}

func (m *Manager) AddPeers(peers []*Peer) {
	for _, peer := range peers {
		peer.InfoHash = m.infohash
		peer.MyPeerId = m.myid
		go peer.Start()
		m.activePeer[peer.ID] = peer
	}
}
