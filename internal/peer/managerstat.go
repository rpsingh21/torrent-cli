package peer

import (
	"sync"
	"time"

	"github.com/rpsingh21/torrent-cli/internal/torrent"
)

const SEED_META_PATH = "~/.torrent-tui/data"

type ManagerStat struct {
	StartAt    time.Time
	Lastupdate time.Time
	mu         sync.Mutex
	Downloaded int
	Uploaded   int
	Completed  int
	Inprogress int
	TotalErr   int
	MetaInfo   *torrent.MetaInfo
	PeerStat   map[string]StatSnapshot // Todo: use pointer with sync map
}

type ManagerStatSnapshot struct {
	StartAt    time.Time
	Lastupdate time.Time
	Downloaded int
	Uploaded   int
	Completed  int
	Inprogress int
	TotalErr   int
	MetaInfo   *torrent.MetaInfo
	PeerStat   map[string]StatSnapshot
}

func NewManagerStat(metainfo *torrent.MetaInfo) *ManagerStat {
	return &ManagerStat{
		StartAt:    time.Now(),
		Lastupdate: time.Now(),
		PeerStat:   make(map[string]StatSnapshot),
	}
}

func (s *ManagerStat) updatePeerStatMap(snapMap map[string]StatSnapshot, completed, inprogress int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var downloadChange, uploadChange int64
	for addr, snap := range snapMap {
		var preDownloaded, preUploaded int64
		if preSnap, ok := s.PeerStat[addr]; ok {
			preDownloaded, preUploaded = preSnap.Downloaded, preSnap.Uploaded
		}
		downloadChange += snap.Downloaded - preDownloaded
		uploadChange += snap.Uploaded - preUploaded
		s.PeerStat[addr] = snap
	}

	s.Completed = completed
	s.Inprogress = inprogress
	s.Downloaded += int(downloadChange)
	s.Uploaded += int(uploadChange)
}

func (s *ManagerStat) updatePeerStat(addr string, snap StatSnapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var preDownloaded, preUploaded int64
	if preSnap, ok := s.PeerStat[addr]; ok {
		preDownloaded, preUploaded = preSnap.Downloaded, preSnap.Uploaded
	}
	downloadChange := snap.Downloaded - preDownloaded
	uploadChange := snap.Uploaded - preUploaded

	s.Downloaded += int(downloadChange)
	s.Uploaded += int(uploadChange)
	s.PeerStat[addr] = snap
}

func (s *ManagerStat) Snapshot() ManagerStatSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	return ManagerStatSnapshot{
		StartAt:    s.StartAt,
		Lastupdate: s.Lastupdate,
		Downloaded: s.Downloaded,
		Uploaded:   s.Uploaded,
		Completed:  s.Completed,
		Inprogress: s.Inprogress,
		MetaInfo:   s.MetaInfo,
		PeerStat:   s.PeerStat,
	}
}

// func(s *ManagerStat) func () {

// }
