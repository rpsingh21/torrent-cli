package app

import (
	"context"
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

func updateMetainfoFromPeers(ctx context.Context, metaInfo *torrent.MetaInfo) error {
	discovery := discovery.New(metaInfo, 900)

	resp := discovery.GetPeerAddresses(ctx, metaInfo.Announce, "")

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	result := make(chan map[string]any, 1)
	var wg sync.WaitGroup

	for _, addr := range resp.Addrs {
		wg.Go(func() {
			peer := peer.NewPeer("", addr, metaInfo)

			// Exit early if another peer already succeeded and cancelled the context
			if ctx.Err() != nil {
				return
			}

			data, err := peer.DownloadMetadata(ctx, metaInfo)
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
				cancel() // Cancel context to cancel other peer metadata download
			case <-ctx.Done():
			}
		})
	}

	done := make(chan struct{})

	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case info := <-result:
		if err := torrent.UpdateInfo(metaInfo, info); err != nil {
			wg.Wait()
			return fmt.Errorf("update metainfo: %w", err)
		}

		slog.Info("complete metadata dowload!")
		wg.Wait()
		return nil

	case <-done:
		return errors.New("all peers failed to get metainfo")

	case <-ctx.Done():
		return ctx.Err()
	}

}
