package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/rpsingh21/torrent-cli/internal/bencode"
	"github.com/rpsingh21/torrent-cli/internal/discovery/tracker"
	"github.com/rpsingh21/torrent-cli/internal/torrent"
)

func updateMetainfoFromPeers(ctx context.Context, metaInfo *torrent.MetaInfo) error {
	resp, err := tracker.AnnounceUPD(metaInfo, 6881)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	result := make(chan map[string]any, 1)
	var wg sync.WaitGroup

	for _, peer := range resp.Peers {
		wg.Go(func() {

			// Exit early if another peer already succeeded and cancelled the context
			if ctx.Err() != nil {
				return
			}

			data, err := peer.DownloadMetadata(ctx, metaInfo)
			if err != nil {
				log.Printf("Failed while downloading metadata from %v err: %v", peer.IP, err)
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
				cancel()
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

		log.Println("==================================================== complete metadata dowload!")

		wg.Wait()
		return nil

	case <-done:
		return errors.New("all peers failed to get metainfo")

	case <-ctx.Done():
		return ctx.Err()
	}

}
