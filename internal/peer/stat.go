package peer

import "sync/atomic"

type Stat struct {
	DownloadRate atomic.Int64
	UploadRate   atomic.Int64

	Downloaded atomic.Int64
	Uploaded   atomic.Int64

	RequestsSent      atomic.Int64
	RequestsCompleted atomic.Int64
	Timeouts          atomic.Int64
	Errors            atomic.Int64
}

type StatSnapshot struct {
	DownloadRate int64
	UploadRate   int64

	Downloaded int64
	Uploaded   int64

	RequestsSent      int64
	RequestsCompleted int64
	Timeouts          int64
	Errors            int64
}

func (s *Stat) AddDownloaded(n int)         { s.Downloaded.Add(int64(n)) }
func (s *Stat) AddUploaded(n int)           { s.Uploaded.Add(int64(n)) }
func (s *Stat) IncRequestsSent()            { s.RequestsSent.Add(1) }
func (s *Stat) IncRequestsCompleted()       { s.RequestsCompleted.Add(1) }
func (s *Stat) IncTimeouts()                { s.Timeouts.Add(1) }
func (s *Stat) IncErrors()                  { s.Errors.Add(1) }
func (s *Stat) GetRequestsCompleted() int64 { return s.RequestsCompleted.Load() }

// Reset clears session counters before reconnecting a dead peer so historical
// bytes already accounted for in ManagerStat are not double-counted.
func (s *Stat) Reset() {
	s.DownloadRate.Store(0)
	s.UploadRate.Store(0)
	s.Downloaded.Store(0)
	s.Uploaded.Store(0)
	s.RequestsSent.Store(0)
	s.RequestsCompleted.Store(0)
	s.Timeouts.Store(0)
	s.Errors.Store(0)
}

func (s *Stat) Snapshot() StatSnapshot {
	return StatSnapshot{
		DownloadRate:      s.DownloadRate.Load(),
		UploadRate:        s.UploadRate.Load(),
		Downloaded:        s.Downloaded.Load(),
		Uploaded:          s.Uploaded.Load(),
		RequestsSent:      s.RequestsSent.Load(),
		RequestsCompleted: s.RequestsCompleted.Load(),
		Timeouts:          s.Timeouts.Load(),
		Errors:            s.Errors.Load(),
	}
}
