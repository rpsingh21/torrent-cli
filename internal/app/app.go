package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"path"
	"sync"
	"time"

	"github.com/rpsingh21/torrent-cli/internal/bencode"
	"github.com/rpsingh21/torrent-cli/internal/discovery"
	"github.com/rpsingh21/torrent-cli/internal/discovery/tracker"
	"github.com/rpsingh21/torrent-cli/internal/peer"
	"github.com/rpsingh21/torrent-cli/internal/piece"
	"github.com/rpsingh21/torrent-cli/internal/storage"
	"github.com/rpsingh21/torrent-cli/internal/torrent"
)

type App struct {
	logger    *log.Logger
	metaInfo  *torrent.MetaInfo
	outputDir string
}

func NewAppFromTorrentFile(tfPath, outputDir string) (*App, error) {
	metaInfo, err := torrent.NewTorrentDetailFromFile(tfPath)
	if err != nil {
		return nil, err
	}

	// Todo: Add safe dir create in storage.
	if metaInfo.Name != "" {
		outputDir = path.Join(outputDir, metaInfo.Name)
	}

	return &App{
		logger:    log.Default(),
		metaInfo:  metaInfo,
		outputDir: outputDir,
	}, nil
}

func NewAppFromMagnetLink(url, outputDir string) (*App, error) {
	metaInfo, err := torrent.MetaInfoFromMagnetURL(url)
	if err != nil {
		return nil, err
	}

	// Todo: Add safe dir create in storage.
	if metaInfo.Name != "" {
		outputDir = path.Join(outputDir, metaInfo.Name)
	}

	resp, err := tracker.AnnounceUPD(metaInfo, 6881)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	successChan := make(chan map[string]any, 1)
	var wg sync.WaitGroup

	for _, p := range resp.Peers {
		wg.Add(1)
		go func(pr *peer.Peer) {
			defer wg.Done()

			// Exit early if another peer already succeeded and cancelled the context
			if ctx.Err() != nil {
				return
			}

			data, err := pr.DownloadMetadata(ctx, metaInfo)
			if err != nil {
				log.Printf("Failed while downloading metadata from %v err: %v", pr.IP, err)
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

			// Safely attempt to send the info.
			// If another peer finished first and triggered cancel(), ctx.Done() will unblock this.
			select {
			case successChan <- info:
			case <-ctx.Done():
			}
		}(p)
	}

	// Background goroutine to close the channel if ALL peers fail, preventing a deadlock.
	go func() {
		wg.Wait()
		close(successChan)
	}()

	// Block until the first peer succeeds or all peers fail.
	if info, ok := <-successChan; ok {
		if err := torrent.UpdateInfo(metaInfo, info); err == nil {
			// log.Printf("Updated metainfo from peer %+v", metaInfo)
			cancel() // This cancels the context, stopping all other in-flight peer dials in extensions.go
		}
	}

	if metaInfo.PieceLength <= 0 {
		return nil, fmt.Errorf("Error doesn't find metainfo from peers")
	}

	return &App{
		logger:    log.Default(),
		metaInfo:  metaInfo,
		outputDir: outputDir,
	}, nil
}

func (a *App) Download(ctx context.Context) error {
	store, err := storage.NewFileStorage(a.metaInfo, a.outputDir)
	if err != nil {
		return err
	}

	defer func() {
		if closeErr := store.Close(); closeErr != nil {
			a.logger.Printf("storage close failed: %v", closeErr)
		}
	}()

	pieceManager := piece.NewManager(a.metaInfo, piece.StrategySequential, store)
	// pieceManager := piece.NewManager(a.metaInfo, piece.StrategyRarestFirst, store)
	peerManager := peer.NewManager(a.metaInfo, pieceManager)
	discovery := discovery.New(a.metaInfo, 300, peerManager.PeerChan)

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	done := make(chan error, 2)
	go func() { done <- discovery.Start(runCtx) }()
	// go func() { done <- discovery.UpdatePeerUDP(runCtx, a.metaInfo, peerManager.PeerChan) }()
	go func() { done <- peerManager.Run(runCtx) }()

	completed := false
	completionCheck := time.NewTicker(250 * time.Millisecond)
	defer completionCheck.Stop()

	for finished := 0; finished < 2; {
		select {
		case err := <-done:
			finished++
			if err != nil && !errors.Is(err, context.Canceled) && !completed {
				cancel()
				peerManager.Close()
				return err
			}
		case <-completionCheck.C:
			if pieceManager.Completed() {
				completed = true
				cancel()
			}
		case <-ctx.Done():
			cancel()
		}
	}

	peerManager.Close()
	if completed {
		return nil
	}
	return ctx.Err()
}
