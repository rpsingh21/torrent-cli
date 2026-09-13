package peer

type Stat struct {
	DownloadRate      uint64 // byte/sec
	UploadRate        uint64
	Downloaded        uint64
	Uploaded          uint64
	RequestsSent      uint64
	RequestsCompleted uint64
	Timeouts          uint64
	Errors            uint64
}
