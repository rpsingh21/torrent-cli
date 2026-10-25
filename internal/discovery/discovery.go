package discovery

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/rpsingh21/torrent-cli/internal/discovery/tracker"
	"github.com/rpsingh21/torrent-cli/internal/peer"
	"github.com/rpsingh21/torrent-cli/internal/torrent"
)

// Discovery collects peers from trackers. DHT and PEX can be added as additional
// discovery sources without changing PeerManager.

func Start(ctx context.Context, metainfo *torrent.MetaInfo, peerChan chan<- *peer.Peer) error {
	interval := 1800 * time.Second
	timer := time.NewTimer(1 * time.Second)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			// Best effort: do not block shutdown on a tracker.
			// stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			// defer cancel()
			// if strings.HasPrefix(d.metaInfo.Announce, "http") {
			// 	if err := d.updatePeerHTTP(stopCtx, "Stopped"); err != nil {
			// 		log.Printf("Stopped tracker announce failed: %v", err)
			// 	}
			// }
			return ctx.Err()

		case <-timer.C:
			AnnounceLoop(ctx, metainfo, peerChan)
			timer.Reset(interval)
		}
	}
}

func AnnounceLoop(ctx context.Context, metainfo *torrent.MetaInfo, peerChan chan<- *peer.Peer) {
	var wg sync.WaitGroup

	for _, url := range metainfo.AnnounceList {
		wg.Go(func() {
			res := GetPeerAddresses(ctx, metainfo, url, "")

			if res != nil && len(res.Addrs) > 0 {
				emitPeerAddressesToPeerManager(ctx, metainfo, res.Addrs, peerChan)
			}
		})
	}

	wg.Wait()
}

func GetPeerAddresses(ctx context.Context, metainfo *torrent.MetaInfo, url, event string) *tracker.Response {
	var res *tracker.Response
	var err error

	if strings.HasPrefix(url, "http") {
		res, err = tracker.AnnounceHTTP(ctx, url, metainfo, event)
	} else {
		res, err = tracker.AnnounceUPD(url, metainfo)
	}

	if err != nil || res == nil {
		log.Printf("tracker announce failed: %v", err)
	}

	return res
}

func emitPeerAddressesToPeerManager(
	ctx context.Context, metainfo *torrent.MetaInfo, addrs []string, peerChan chan<- *peer.Peer) error {

	for _, addr := range addrs {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case peerChan <- peer.NewPeer("", addr, metainfo):
		}
	}

	return nil
}
