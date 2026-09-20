package app

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/rpsingh21/torrent-cli/internal/discovery"
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
	peerManager := peer.NewManager(a.metaInfo, pieceManager)
	discovery := discovery.New(a.metaInfo, 300, peerManager.PeerChan)

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	done := make(chan error, 2)
	go func() { done <- discovery.Start(runCtx) }()
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
