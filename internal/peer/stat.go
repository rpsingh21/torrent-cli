package peer

import (
	"sync"
	"time"
)

type Stat struct {
	mu sync.RWMutex

	DownloadLatency    time.Duration
	MinDownloadLatency time.Duration

	DownloadRate int64
	UploadRate   int64

	Downloaded int64
	Uploaded   int64

	RequestsSent      int64
	RequestsCompleted int64
	Timeouts          int64
	Errors            int64
}

type StatSnapshot struct {
	DownloadRate int64
	UploadRate   int64

	Downloaded int64
	Uploaded   int64

	DownloadLatency    time.Duration
	MinDownloadLatency time.Duration

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

func (s *Stat) updateDownloadWithLatency(n int, latency time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Downloaded += int64(n)

	if s.MinDownloadLatency == 0 || latency < s.MinDownloadLatency {
		s.MinDownloadLatency = latency
	}

	if s.DownloadLatency == 0 {
		s.DownloadLatency = latency
		return
	}

	currDownloadRate := int64(float64(n) / latency.Seconds())
	s.DownloadRate = (2*currDownloadRate + 8*s.DownloadRate) / 10

	s.DownloadLatency = (2*latency + 8*s.DownloadLatency) / 10
}

func (s *Stat) queueDelayAndDowloadrateSnapshot() (time.Duration, int64) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.DownloadLatency - s.MinDownloadLatency, s.DownloadRate
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
		DownloadRate: s.DownloadRate,
		UploadRate:   s.UploadRate,

		Downloaded: s.Downloaded,
		Uploaded:   s.Uploaded,

		DownloadLatency:    s.DownloadLatency,
		MinDownloadLatency: s.MinDownloadLatency,

		RequestsSent:      s.RequestsSent,
		RequestsCompleted: s.RequestsCompleted,
		Timeouts:          s.Timeouts,
		Errors:            s.Errors,
	}
}
