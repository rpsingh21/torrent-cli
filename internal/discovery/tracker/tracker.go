package tracker

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rpsingh21/torrent-cli/internal/bencode"
	"github.com/rpsingh21/torrent-cli/internal/torrent"
)

type Tracker struct {
	MetaInfo *torrent.MetaInfo
	client   *http.Client
}

func NewTracker(metaInfo *torrent.MetaInfo) *Tracker {
	return &Tracker{
		MetaInfo: metaInfo,
		client:   &http.Client{Timeout: 15 * time.Second},
	}
}

func (t *Tracker) RequestPeers(ctx context.Context, event string) (*Response, error) {
	url, err := t.MetaInfo.BuildTrackerURL(event)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Tracker HTTP status %d (%s)", resp.StatusCode, t.MetaInfo.Announce)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}

	respData, err := bencode.NewDecoder(body).Decode()
	if err != nil {
		return nil, fmt.Errorf("decode tracker response: %w", err)
	}

	return UnmarshalTrackerResponse(respData)
}
