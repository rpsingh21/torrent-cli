package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"path"
	"time"

	"github.com/rpsingh21/torrent-cli/internal/discovery"
	"github.com/rpsingh21/torrent-cli/internal/peer"
	"github.com/rpsingh21/torrent-cli/internal/piece"
	"github.com/rpsingh21/torrent-cli/internal/storage"
	"github.com/rpsingh21/torrent-cli/internal/torrent"
)

const MAX_PIECE_SIZE = 256 << 20

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

func NewAppFromMagnetLink(ctx context.Context, url, outputDir string) (*App, error) {
	metaInfo, err := torrent.MetaInfoFromMagnetURL(url)
	if err != nil {
		return nil, err
	}

	// Todo: Add safe dir create in storage.
	if metaInfo.Name != "" {
		outputDir = path.Join(outputDir, metaInfo.Name)
	}

	if metaInfo.PieceHashes == nil {
		if err := updateMetainfoFromPeers(ctx, metaInfo); err != nil {
			return nil, fmt.Errorf("discover torrent metadata: %w", err)
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
	fmt.Printf("download details == Piece Size %v | Pieces %v | Blocks per Piece %v\n\n",
		a.metaInfo.PieceLength, a.metaInfo.TotalSize/a.metaInfo.PieceLength,
		a.metaInfo.PieceLength/(16*1024))
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
	discovery := discovery.New(a.metaInfo, 300)

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	done := make(chan error, 2)
	go func() { done <- discovery.Start(runCtx, peerManager.PeerChan) }()
	go func() { done <- peerManager.Run(runCtx) }()

	completed := false
	completionCheck := time.NewTicker(2 * time.Second)
	defer completionCheck.Stop()

	for finished := 0; finished < 2; {
		select {
		case err := <-done:
			finished++
			fmt.Println("")
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
