package peer

type PeerStat struct {
	DownloadRate      uint64 // bit/sec
	UploadRate        uint64
	RequestsSent      uint64
	RequestsCompleted uint64
	Timeouts          uint64
	Errors            uint64
}
