package peer

import "sync"

type Stat struct {
	mu                sync.RWMutex
	DownloadRate      int64
	UploadRate        int64
	Downloaded        int64
	Uploaded          int64
	RequestsSent      int64
	RequestsCompleted int64
	Timeouts          int64
	Errors            int64
}

type StatSnapshot struct {
	DownloadRate      int64
	UploadRate        int64
	Downloaded        int64
	Uploaded          int64
	RequestsSent      int64
	RequestsCompleted int64
	Timeouts          int64
	Errors            int64
}

func (s *Stat) AddDownloaded(n int) {
	s.mu.Lock()
	s.Downloaded += int64(n)
	s.mu.Unlock()
}

func (s *Stat) AddUploaded(n int) {
	s.mu.Lock()
	s.Uploaded += int64(n)
	s.mu.Unlock()
}

func (s *Stat) IncRequestsSent() {
	s.mu.Lock()
	s.RequestsSent++
	s.mu.Unlock()
}

func (s *Stat) IncRequestsCompleted() {
	s.mu.Lock()
	s.RequestsCompleted++
	s.mu.Unlock()
}

func (s *Stat) IncTimeouts() {
	s.mu.Lock()
	s.Timeouts++
	s.mu.Unlock()
}

func (s *Stat) Snapshot() StatSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return StatSnapshot{
		DownloadRate: s.DownloadRate, UploadRate: s.UploadRate, Downloaded: s.Downloaded,
		Uploaded: s.Uploaded, RequestsSent: s.RequestsSent, RequestsCompleted: s.RequestsCompleted,
		Timeouts: s.Timeouts, Errors: s.Errors,
	}
}
