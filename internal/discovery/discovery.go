package discovery

import (
	"context"
	"log"
	"time"

	"github.com/rpsingh21/torrent-cli/internal/discovery/tracker"
	"github.com/rpsingh21/torrent-cli/internal/peer"
	"github.com/rpsingh21/torrent-cli/internal/torrent"
)

// Discovery collects peers from trackers. DHT and PEX can be added as additional
// discovery sources without changing PeerManager.
type Discovery struct {
	tracker  *tracker.Tracker
	interval time.Duration
	peerChan chan *peer.Peer
}

func New(metaInfo *torrent.MetaInfo, defaultInterval int, peerChan chan *peer.Peer) *Discovery {
	d := &Discovery{
		interval: time.Duration(defaultInterval) * time.Second,
		peerChan: peerChan,
	}
	if metaInfo.Announce != "" {
		d.tracker = tracker.NewTracker(metaInfo)
	}
	return d
}

func (d *Discovery) Start(ctx context.Context) error {
	if d.tracker == nil {
		return nil
	}

	interval := d.interval
	if err := d.updatePeersFromTracker(ctx, "started"); err == nil {
		if d.interval > 0 {
			interval = d.interval
		}
	} else {
		log.Printf("Initial tracker announce failed: %v", err)
	}

	timer := time.NewTimer(interval)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			// Best effort: do not block shutdown on a tracker.
			stopCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if err := d.updatePeersFromTracker(stopCtx, "Stopped"); err != nil {
				log.Printf("Stopped tracker announce failed: %v", err)
			}
			return ctx.Err()
		case <-timer.C:
			if err := d.updatePeersFromTracker(ctx, ""); err != nil && ctx.Err() == nil {
				log.Printf("Tracker announce failed: %v", err)
			}
			interval = d.interval
			if interval <= 0 {
				interval = 15 * time.Minute
			}
			timer.Reset(interval)
		}
	}
}

func (d *Discovery) updatePeersFromTracker(ctx context.Context, event string) error {
	res, err := d.tracker.RequestPeers(ctx, event)
	if err != nil {
		return err
	}
	if res.Interval > 0 {
		d.interval = time.Duration(res.Interval) * time.Second
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
