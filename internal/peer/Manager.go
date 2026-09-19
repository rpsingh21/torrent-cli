package peer

import "context"

// Load from config
const (
	MAX_PEERS         = 80
	REQUESTS_PER_PEER = 32
	REQUEST_TIMEOUT   = 15
)

type Manager struct {
	ctx            context.Context
	activePeer     map[string]*Peer
	peerChan       chan *Peer
	removePeerChan chan *Peer
	infohash       [20]byte
	myid           [20]byte
}

func NewManager(infoHash, myid [20]byte) *Manager {
	ctx := context.WithoutCancel(context.Background())
	return &Manager{
		ctx:            ctx,
		activePeer:     make(map[string]*Peer),
		peerChan:       make(chan *Peer, 10),
		removePeerChan: make(chan *Peer, 5),
		infohash:       infoHash,
		myid:           myid,
	}
}

func (m *Manager) Run() {
	go func() {
		for {
			select {
			case peer := <-m.peerChan:
				m.AddPeer(peer)
			case peer := <-m.removePeerChan:
				m.removePeer(peer)
			case <-m.ctx.Done():
				m.Close()
			}

		}
	}()
}

func (m *Manager) AddPeer(peer *Peer) {
	peer.InfoHash = m.infohash
	peer.MyPeerId = m.myid
	peer.removeChan = m.removePeerChan
	go peer.Start()
	m.activePeer[peer.ID] = peer
}

func (m *Manager) removePeer(peer *Peer) {
	delete(m.activePeer, peer.ID)
}

func (m *Manager) Close() {
	for _, peer := range m.activePeer {
		peer.Close()
	}
}
