package torrent

type TFile struct {
	Length int64
	Path   string
}

type MetaInfo struct {
	AppId        [20]byte
	AppPort      uint16
	Announce     string
	AnnounceList []string
	InfoHash     [20]byte
	PieceHashes  [][20]byte
	PieceLength  int64
	Length       int64
	Name         string
	Files        []TFile
	TotalSize    int64
	TotalPices   int
}
