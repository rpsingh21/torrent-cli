package app

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"sync"

	"github.com/rpsingh21/torrent-cli/internal/bencode"
	"github.com/rpsingh21/torrent-cli/internal/discovery"
	"github.com/rpsingh21/torrent-cli/internal/peer"
	"github.com/rpsingh21/torrent-cli/internal/torrent"
)

func updateMetainfoFromPeers(pctx context.Context, metaInfo *torrent.MetaInfo) error {
	var wg sync.WaitGroup

	result := make(chan map[string]any, 1)
	defer close(result)
	peerChan := make(chan *peer.Peer, 16)
	ctx, cancel := context.WithCancel(pctx)
	defer cancel()

	done := make(chan struct{})

	wg.Go(func() {
		discovery.AnnounceLoop(ctx, metaInfo, peerChan)
		close(peerChan)
	})

	go peerLoopForMetainfo(ctx, peerChan, result)

	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case info := <-result:
		cancel()
		if err := torrent.UpdateInfo(metaInfo, info); err != nil {
			return fmt.Errorf("update metainfo: %w", err)
		}

		slog.Info("complete metadata dowload!")
		return nil

	case <-done:
		return errors.New("all peers failed to get metainfo")

	case <-ctx.Done():
		return ctx.Err()
	}

}

func peerLoopForMetainfo(ctx context.Context, peerChan <-chan *peer.Peer, result chan<- map[string]any) {
	for {
		select {
		case <-ctx.Done():
			return
		case peer := <-peerChan:
			go fetchMetaInfo(ctx, peer, result)
		}
	}
}

func fetchMetaInfo(ctx context.Context, peer *peer.Peer, result chan<- map[string]any) {
	// Exit early if another peer already succeeded and cancelled the context
	if ctx.Err() != nil {
		return
	}

	data, err := peer.DownloadMetadata(ctx)
	if err != nil {
		log.Printf("Failed while downloading metadata from %v err: %v", peer.Addr, err)
		return
	}

	bdata, err := bencode.NewDecoder(data).Decode()
	if err != nil {
		return
	}

	info, ok := bdata.(map[string]any)
	if !ok {
		return
	}

	select {
	case result <- info:
	default:
		// Result already sent by another peer
	}
}

func calculate_peer_id() [20]byte {
	var appId [20]byte
	if _, err := rand.Read(appId[:]); err != nil {
		// Fallback or handle entropy failure if necessary
		log.Printf("Warning: failed to read random bytes for peerId: %v", err)
		panic("failed to read random bytes for appId")
	}
	return appId
}
