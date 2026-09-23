package peer

import (
	"context"
	"fmt"
	"log"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/rpsingh21/torrent-cli/internal/piece"
	"github.com/rpsingh21/torrent-cli/internal/torrent"
)

const (
	MAX_PEERS         = 100
	REQUESTS_PER_PEER = 64
	REQUEST_TIMEOUT   = 10
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

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	var lastTime = time.Now()

	var mbp float64 = 1000_000
	var kbp float64 = 1000
	var preDownload, preUpload int64

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
		case <-ticker.C:
			now := time.Now()
			elapsed := now.Sub(lastTime).Seconds()
			lastTime = now

			var download, upload int64
			var totalReqs, totalErrs int64

			totalPeer := len(m.activePeer)

			for _, v := range m.activePeer {
				snap := v.stat.Snapshot()

				download += snap.Downloaded
				upload += snap.Uploaded

				totalReqs += snap.RequestsSent
				totalErrs += snap.Errors
			}

			downloadRate := float64(download-preDownload) / elapsed
			uploadRate := float64(upload-preUpload) / elapsed

			preDownload = download
			preUpload = upload

			completed, inprogress := m.pieceManager.GetStat()

			fmt.Printf(
				"\r\033[KTotalPeer: %v [Downloaded: %.2f MB | Speed: %.2f MB/s] [Uploaded: %.2f MB | Speed: %.2f KB/s] TotalReq: %v | Errors: %v [Pieces: %v | %v (%v) | %v]",
				totalPeer,
				float64(download)/mbp,
				downloadRate/mbp,
				float64(upload)/mbp,
				uploadRate/kbp,
				totalReqs,
				totalErrs,
				completed,
				inprogress,
				inprogress-completed,
				m.metaInfo.TotalPices,
			)
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

	m.peersWG.Go(func() {
		if err := p.Start(ctx); err != nil && ctx.Err() == nil {
			downloaded := p.stat.Snapshot().Downloaded
			log.Printf("Peer %s failed: %v, Downloaded = %v", key, err.Error(), downloaded/1000)
		}
		select {
		case m.removePeerChan <- p:
		case <-ctx.Done():
		}
	})
}

func (m *Manager) removePeer(p *Peer) {
	if p == nil {
		return
	}
	key := p.Key()

	m.mu.Lock()
	delete(m.activePeer, key)
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
