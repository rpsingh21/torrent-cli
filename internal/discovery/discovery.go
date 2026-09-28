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
	peerChan chan *peer.Peer
}

func New(metaInfo *torrent.MetaInfo, defaultInterval int, peerChan chan *peer.Peer) *Discovery {
	return &Discovery{
		metaInfo: metaInfo,
		interval: time.Duration(defaultInterval) * time.Second,
		peerChan: peerChan,
	}
}

// func UpdatePeerUDP(ctx context.Context, metaInfo *torrent.MetaInfo, peerChan chan *peer.Peer) error {
// 	interval := 1 * time.Second

// 	timer := time.NewTimer(interval)
// 	defer timer.Stop()

// 	for {
// 		select {
// 		case <-ctx.Done():
// 			return ctx.Err()
// 		case <-timer.C:
// 			res, err := tracker.AnnounceUPD(metaInfo, 6881)
// 			if err != nil {
// 				timer.Reset(5 * time.Second)
// 			} else {
// 				for _, p := range res.Peers {
// 					peerChan <- p
// 				}
// 				timer.Reset(15 * time.Minute)
// 			}
// 		}
// 	}
// }

func (d *Discovery) Start(ctx context.Context) error {

	// interval := d.interval
	// if err := d.updatePeersFromTracker(ctx, "started"); err == nil {
	// 	if d.interval > 0 {
	// 		interval = d.interval
	// 	}
	// } else {
	// 	log.Printf("Initial tracker announce failed: %v", err)
	// 	interval = 5 * time.Second
	// }

	// Start immideate
	timer := time.NewTimer(1 * time.Second)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			// Best effort: do not block shutdown on a tracker.
			stopCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if strings.HasPrefix(d.metaInfo.Announce, "http") {
				if err := d.updatePeerHTTP(stopCtx, "Stopped"); err != nil {
					log.Printf("Stopped tracker announce failed: %v", err)
				}
			}
			return ctx.Err()
		case <-timer.C:
			if strings.HasPrefix(d.metaInfo.Announce, "http") {
				if err := d.updatePeerHTTP(ctx, ""); err != nil && ctx.Err() == nil {
					log.Printf("Tracker announce failed: %v", err)
				}
			} else {
				if err := d.updatePeerUDP(ctx); err != nil && ctx.Err() == nil {
					log.Printf("Tracker announce failed: %v", err)
				}
			}

			timer.Reset(d.interval)
		}
	}
}

func (d *Discovery) updatePeerUDP(ctx context.Context) error {
	res, err := tracker.AnnounceUPD(d.metaInfo, 6881)
	if err != nil {
		return err
	}
	for _, p := range res.Peers {
		select {
		case d.peerChan <- p:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func (d *Discovery) updatePeerHTTP(ctx context.Context, event string) error {
	res, err := tracker.AnnounceHTTP(ctx, d.metaInfo, event)
	if err != nil {
		return err
	}
	if res.Interval > 0 {
		// d.interval = time.Duration(res.Interval) * time.Second
		d.interval = 15 * time.Minute
	}

	for _, p := range res.Peers {
		select {
		case d.peerChan <- p:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}
