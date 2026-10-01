package torrent

import (
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
)

func MetaInfoFromMagnetURL(murl string) (*MetaInfo, error) {
	ur, err := url.Parse(murl)
	if err != nil {
		return nil, err
	}

	params := ur.Query()
	var infoHash [20]byte

	hexHash := strings.TrimPrefix(params.Get("xt"), "urn:btih:")
	n, err := hex.Decode(infoHash[:], []byte(hexHash))
	if err != nil || n != 20 {
		return nil, fmt.Errorf("InfoHash Decoding error")
	}

	name := params.Get("dn")
	tracker := params.Get("tr")
	allTrackers := params["tr"]

	metaInfo := &MetaInfo{
		AppId:        calculate_peer_id(),
		Announce:     tracker,
		AnnounceList: allTrackers,
		InfoHash:     infoHash,
		Name:         name,
	}

	return metaInfo, err
}
