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
			log.Printf("storage close failed: %v", closeErr)
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
	completionCheck := time.NewTicker(1 * time.Second)
	defer completionCheck.Stop()

	var preDownloaded, preUploaded int

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
			ms := peerManager.UpdateState()
			printStat(len(ms.PeerStat), ms.Downloaded, ms.Uploaded, 0, 0, ms.Completed, ms.Inprogress,
				ms.Downloaded-preDownloaded, ms.Uploaded-preUploaded)
			preDownloaded = ms.Downloaded
			preUploaded = ms.Uploaded

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

const mib = 1 << 20
const kib = 1 << 10

func printStat(totalPeer, downloaded, uploaded, totalReqs, totalErrs, completed, inprogress, dr, ur int) {

	fmt.Printf(
		"\r\033[KTotalPeer: %v [Downloaded: %.2f MiB | Speed: %.2f MiB/s] [Uploaded: %.2f MiB | Speed: %.2f KiB/s] TotalReq: %v | Errors: %v [Pieces: %v | %v]",
		totalPeer,
		float64(downloaded)/mib,
		float64(dr)/mib,
		float64(uploaded)/mib,
		float64(ur)/kib,
		totalReqs,
		totalErrs,
		completed,
		inprogress,
	)
}
