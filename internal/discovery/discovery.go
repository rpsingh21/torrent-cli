package discovery

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/rpsingh21/torrent-cli/internal/discovery/tracker"
	"github.com/rpsingh21/torrent-cli/internal/peer"
	"github.com/rpsingh21/torrent-cli/internal/torrent"
)

// Discovery collects peers from trackers. DHT and PEX can be added as additional
// discovery sources without changing PeerManager.
type Discovery struct {
	metaInfo *torrent.MetaInfo
	interval time.Duration
}

func New(metaInfo *torrent.MetaInfo, defaultInterval int) *Discovery {
	return &Discovery{
		metaInfo: metaInfo,
		interval: time.Duration(defaultInterval) * time.Second,
	}
}

func (d *Discovery) Start(ctx context.Context, peerChan chan<- *peer.Peer) error {
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
			var res *tracker.Response

			for _, url := range d.metaInfo.AnnounceList {
				log.Printf("starting getting peer from: %v", url)
				res = d.GetPeerAddresses(ctx, url, "")

				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}

				if res != nil {
					break
				}
			}

			if res != nil {
				d.emitPeerAddressesToPeerManager(ctx, res.Addrs, peerChan)
				d.interval = time.Duration(res.Interval) * time.Second
			}

			timer.Reset(d.interval)
		}
	}
}

func (d *Discovery) GetPeerAddresses(ctx context.Context, url string, event string) *tracker.Response {
	var res *tracker.Response
	var err error

	if strings.HasPrefix(url, "http") {
		res, err = tracker.AnnounceHTTP(ctx, url, d.metaInfo, event)
	} else {
		res, err = tracker.AnnounceUPD(url, d.metaInfo)
	}

	if err != nil || res == nil {
		log.Printf("tracker announce failed: %v", err)
	}

	return res
}

func (d *Discovery) emitPeerAddressesToPeerManager(
	ctx context.Context, addrs []string, peerChan chan<- *peer.Peer) error {

	for _, addr := range addrs {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case peerChan <- peer.NewPeer("", addr, d.metaInfo):
		}
	}

	return nil
}
