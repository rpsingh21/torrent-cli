package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"path"
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

	for _, peer := range resp.Peers {
		// Todo: MetaInfo set while creating peers
		data, err := peer.DownloadMetadata(ctx, metaInfo)
		if err != nil {
			log.Printf("Failed while dowlaoing metadata from %v err: %v", peer.IP, err)
			continue
		}
		bdata, err := bencode.NewDecoder(data).Decode()
		info, ok := bdata.(map[string]any)
		if !ok {
			continue
		}
		if err := torrent.UpdateInfo(metaInfo, info); err == nil {
			log.Printf("Updated metainfo from peer %+v", metaInfo)
			break
		}
	}

	if metaInfo.PieceLength > 0 {
		return nil, fmt.Errorf("Error doesn't find metainfo from peer")
	}

	return &App{
		logger:    log.Default(),
		metaInfo:  metaInfo,
		outputDir: outputDir,
	}, nil
}

func (a *App) Download(ctx context.Context) error {
	log.Printf("App updated info %+v", a.metaInfo)

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
	// discovery := discovery.New(a.metaInfo, 300, peerManager.PeerChan)

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	done := make(chan error, 2)
	go func() { done <- discovery.UpdatePeerUDP(runCtx, a.metaInfo, peerManager.PeerChan) }()
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
