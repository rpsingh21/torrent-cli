package torrent

import (
	"encoding/hex"
	"encoding/json/v2"
	"os"
	"path/filepath"
)

type TFile struct {
	Length int64
	Path   string
}

type MetaInfo struct {
	AppId        [20]byte
	AppPort      uint16
	Storepath    string
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

func (m *MetaInfo) Save(appdir string) error {
	filename := filepath.Join(appdir, "metainfo", hex.EncodeToString(m.InfoHash[:])+".json")

	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return err
	}

	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}

	defer file.Close()
	return json.MarshalWrite(file, m)
}

func (m *MetaInfo) Remove(appdir string) error {
	filename := filepath.Join(appdir, "metainfo", hex.EncodeToString(m.InfoHash[:])+".json")
	return os.Remove(filename)
}

func LoadMetainfo(metafilename string, metainfo *MetaInfo) error {
	data, err := os.ReadFile(metafilename)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, metainfo)
}
