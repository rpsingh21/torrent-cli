package peer

import (
	"context"
	"fmt"
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
	PeerChan     chan *Peer
	activePeer   map[string]*Peer
	metaInfo     *torrent.MetaInfo
	pieceManager *piece.Manager
	mu           sync.Mutex
	refreshMap   map[string]time.Duration
}

func NewManager(metaInfo *torrent.MetaInfo, pieceManager *piece.Manager) *Manager {
	return &Manager{
		activePeer:   make(map[string]*Peer),
		PeerChan:     make(chan *Peer, 128),
		metaInfo:     metaInfo,
		pieceManager: pieceManager,
		refreshMap:   make(map[string]time.Duration),
	}
}

func (m *Manager) Run(ctx context.Context) error {
	var wg sync.WaitGroup

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	var lastTime = time.Now()
	var mib float64 = 1 << 20
	var kib float64 = 1 << 10
	var preDownload, preUpload int64
	preSnapshot := make(map[string]StatSnapshot)

	for {
		select {
		case <-ctx.Done():
			m.Close()
			wg.Wait()
			return ctx.Err()

		case p := <-m.PeerChan:
			if p == nil {
				continue
			}
			m.addPeer(ctx, p, &wg)

		case <-ticker.C:
			now := time.Now()
			elapsed := now.Sub(lastTime).Seconds()
			lastTime = now

			var download, upload int64
			var totalReqs, totalErrs int64

			m.mu.Lock()
			totalPeer := len(m.activePeer)

			for _, v := range m.activePeer {
				preSnapshot[v.Addr] = v.stat.Snapshot()
			}
			m.mu.Unlock()

			for _, snap := range preSnapshot {
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
				"\r\033[KTotalPeer: %v/%v [Downloaded: %.2f MiB | Speed: %.2f MiB/s] [Uploaded: %.2f MiB | Speed: %.2f KiB/s] TotalReq: %v | Errors: %v [Pieces: %v | %v (%v | %v)]",
				totalPeer,
				len(preSnapshot),
				float64(download)/mib,
				downloadRate/mib,
				float64(upload)/mib,
				uploadRate/kib,
				totalReqs,
				totalErrs,
				completed,
				inprogress,
				inprogress-completed,
				len(m.pieceManager.ReleaseQue),
			)
		}
	}
}

func (m *Manager) addPeer(ctx context.Context, p *Peer, wg *sync.WaitGroup) {
	m.mu.Lock()
	if _, exists := m.activePeer[p.Addr]; exists || len(m.activePeer) >= MAX_PEERS {
		m.mu.Unlock()
		return
	}
	// p.metaInfo = m.metaInfo
	p.pieceManager = m.pieceManager
	m.activePeer[p.Addr] = p
	m.mu.Unlock()

	wg.Go(func() {
		if err := p.Start(ctx); err != nil && ctx.Err() == nil {
			log.Printf("Peer %s failed: %v, Downloaded = %v KiB", p.Addr, err.Error(), p.stat.Snapshot().Downloaded/1024)
		}
		p.Close()
		m.removePeer(p)

		if p.stat.Snapshot().Downloaded > 0 {
			go m.scheduleRefreshDeadPeer(ctx, p)
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
		m.PeerChan <- p
	}
}

func (m *Manager) removePeer(p *Peer) {
	if p == nil {
		return
	}

	m.mu.Lock()
	delete(m.activePeer, p.Addr)
	m.mu.Unlock()

	m.pieceManager.RemovePeer(p.Addr)
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
