package torrent

import (
	"net/url"
	"strconv"
)

type TFile struct {
	Length int64
	Path   string
}

type MetaInfo struct {
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

func (tm *MetaInfo) BuildTrackerURL(peerId [20]byte, event string) (string, error) {
	base, err := url.Parse(tm.Announce)
	if err != nil {
		return "", err
	}

	params := url.Values{
		"info_hash":  []string{string(tm.InfoHash[:])},
		"peer_id":    []string{string(peerId[:])},
		"port":       []string{strconv.Itoa(int(6889))},
		"uploaded":   []string{"0"},
		"downloaded": []string{"0"},
		"compact":    []string{"1"},
		"left":       []string{strconv.Itoa(int(tm.Length))},
	}
	if event != "" {
		params.Add("event", event)
	}

	base.RawQuery = params.Encode()
	return base.String(), nil
}
