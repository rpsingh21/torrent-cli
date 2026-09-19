package peer

type Stat struct {
	DownloadRate      int // byte/sec
	UploadRate        int
	Downloaded        int
	Uploaded          int
	RequestsSent      int
	RequestsCompleted int
	Timeouts          int
	Errors            int
}
