package discovery

import (
	"log"
	"time"

	"github.com/rpsingh21/torrent-cli/internal/discovery/tracker"
	"github.com/rpsingh21/torrent-cli/internal/peer"
	"github.com/rpsingh21/torrent-cli/internal/torrent"
)

// This collect peers for Tracker(announcers), DHC, PEX
// As of now only support Tracker. Implement DHC and PEX
type Discovery struct {
	tracker  *tracker.Tracker
	interval int
	peerChan chan *peer.Peer
}

func New(metaInfo *torrent.MetaInfo, defaultInterval int, peerChan chan *peer.Peer) *Discovery {
	discovery := &Discovery{
		interval: defaultInterval,
		peerChan: peerChan,
	}
	if metaInfo.Announce != "" {
		discovery.tracker = tracker.NewTracker(metaInfo)
	}
	discovery.updatePeersFromTracker("started")
	return discovery
}

// implement gracefull close
func (d *Discovery) Start() {
	for {
		time.Sleep(time.Duration(d.interval) * time.Second)
		d.updatePeersFromTracker("")
	}
}

func (d *Discovery) updatePeersFromTracker(event string) {
	res, err := d.tracker.RequestPeers(event)
	if err != nil {
		log.Println("Failed to get peers from tracker", err)
	}

	if res.Interval > 0 {
		d.interval = res.Interval
	}
	for _, peer := range res.Peers {
		d.peerChan <- peer
	}
}
