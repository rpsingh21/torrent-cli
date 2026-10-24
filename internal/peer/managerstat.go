package peer

import (
	"maps"
	"sync"
	"time"

	"github.com/rpsingh21/torrent-cli/internal/torrent"
)

const SEED_META_PATH = "~/.torrent-tui/data"

type ManagerStat struct {
	StartAt      time.Time
	Lastupdate   time.Time
	mu           sync.RWMutex
	Downloaded   int64
	Uploaded     int64
	DownloadRate int64
	UploadRate   int64
	Completed    int
	Inprogress   int
	TotalErr     int
	MetaInfo     *torrent.MetaInfo
	PeerStat     map[string]StatSnapshot
}

type ManagerStatSnapshot struct {
	StartAt      time.Time
	Lastupdate   time.Time
	Downloaded   int64
	Uploaded     int64
	DownloadRate int64
	UploadRate   int64
	Completed    int
	Inprogress   int
	TotalErr     int
	MetaInfo     *torrent.MetaInfo
	PeerStat     map[string]StatSnapshot
}

func NewManagerStat(metainfo *torrent.MetaInfo) *ManagerStat {
	now := time.Now()
	return &ManagerStat{
		StartAt:    now,
		Lastupdate: now,
		MetaInfo:   metainfo,
		PeerStat:   make(map[string]StatSnapshot),
	}
}

// updatePeerStatMap calculates per-peer and manager speeds, accumulates total bytes,
// and swaps s.PeerStat with snapMap so only active peers remain in memory.
func (s *ManagerStat) updatePeerStatMap(snapMap map[string]StatSnapshot, completed, inprogress int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(s.Lastupdate).Seconds()

	var downloadChange, uploadChange int64
	var totalDownRate, totalUpRate int64

	for addr, snap := range snapMap {
		var preDownloaded, preUploaded int64
		preSnap, ok := s.PeerStat[addr]
		if ok {
			preDownloaded, preUploaded = preSnap.Downloaded, preSnap.Uploaded
		}

		dDown := max(0, snap.Downloaded-preDownloaded)
		dUp := max(0, snap.Uploaded-preUploaded)
		downloadChange += dDown
		uploadChange += dUp

		if elapsed >= 0.1 {
			snap.DownloadRate = int64(float64(dDown) / elapsed)
			snap.UploadRate = int64(float64(dUp) / elapsed)
		} else if ok {
			snap.DownloadRate = preSnap.DownloadRate
			snap.UploadRate = preSnap.UploadRate
		}

		totalDownRate += snap.DownloadRate
		totalUpRate += snap.UploadRate
		snapMap[addr] = snap
	}

	s.Completed = completed
	s.Inprogress = inprogress
	s.Downloaded += downloadChange
	s.Uploaded += uploadChange
	s.DownloadRate = totalDownRate
	s.UploadRate = totalUpRate
	s.PeerStat = snapMap
	s.Lastupdate = now
}

// removePeerStat records final bytes transferred by a disconnecting peer before
// removing it from the active PeerStat map.
func (s *ManagerStat) removePeerStat(addr string, snap StatSnapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var preDownloaded, preUploaded int64
	if preSnap, ok := s.PeerStat[addr]; ok {
		preDownloaded, preUploaded = preSnap.Downloaded, preSnap.Uploaded
		delete(s.PeerStat, addr)
	}

	s.Downloaded += max(0, snap.Downloaded-preDownloaded)
	s.Uploaded += max(0, snap.Uploaded-preUploaded)
	s.TotalErr += int(snap.Errors)
}

func (s *ManagerStat) Snapshot() ManagerStatSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return ManagerStatSnapshot{
		StartAt:      s.StartAt,
		Lastupdate:   s.Lastupdate,
		Downloaded:   s.Downloaded,
		Uploaded:     s.Uploaded,
		DownloadRate: s.DownloadRate,
		UploadRate:   s.UploadRate,
		Completed:    s.Completed,
		Inprogress:   s.Inprogress,
		TotalErr:     s.TotalErr,
		MetaInfo:     s.MetaInfo,
		PeerStat:     maps.Clone(s.PeerStat),
	}
}
