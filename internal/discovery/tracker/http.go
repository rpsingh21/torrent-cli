package tracker

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/rpsingh21/torrent-cli/internal/bencode"
	"github.com/rpsingh21/torrent-cli/internal/torrent"
)

func AnnounceHTTP(ctx context.Context, http_url string, metaInfo *torrent.MetaInfo, event string) (*Response, error) {
	url, err := BuildTrackerURL(http_url, event, int(metaInfo.Length), metaInfo.InfoHash, metaInfo.AppId, metaInfo.AppPort)
	if err != nil {
		return nil, err
	}

	client := http.Client{Timeout: 15 * time.Second}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Tracker HTTP status %d (%s)", resp.StatusCode, metaInfo.Announce)
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

func BuildTrackerURL(http_url, event string, length int, iHash, appId [20]byte, port uint16) (string, error) {
	base, err := url.Parse(http_url)
	if err != nil {
		return "", err
	}

	params := url.Values{
		"info_hash":  []string{string(iHash[:])},
		"peer_id":    []string{string(appId[:])},
		"port":       []string{strconv.Itoa(int(port))},
		"uploaded":   []string{"0"},
		"downloaded": []string{"0"},
		"compact":    []string{"1"},
		"left":       []string{strconv.Itoa(int(length))},
	}
	if event != "" {
		params.Add("event", event)
	}

	base.RawQuery = params.Encode()
	return base.String(), nil
}
