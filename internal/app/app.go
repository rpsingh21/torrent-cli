package app

import (
	"log"
	"sync"

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

func NewAppFromTorrentFile(tfPath string, outputDir string) *App {
	logger := log.Default()
	metaInfo, err := torrent.NewTorrentDetailFromFile(tfPath)
	if err != nil {
		log.Fatalf("Failed to load torrent file (%v) %+v", tfPath, err)
	}

	return &App{
		logger:    logger,
		metaInfo:  metaInfo,
		outputDir: outputDir,
	}
}

func (a *App) Download() {
	var wg sync.WaitGroup

	storage, err := storage.NewFileStorage(a.metaInfo, a.outputDir)
	if err != nil {
		log.Fatalln("Failed to create storage: ", err)
	}

	pieceManager := piece.NewManager(a.metaInfo, piece.StrategySequential, storage)
	peerManager := peer.NewManager(a.metaInfo, pieceManager)
	discovery := discovery.New(a.metaInfo, 300, peerManager.PeerChan)

	wg.Go(discovery.Start)
	wg.Go(peerManager.Run)

	wg.Wait()
	peerManager.Close()
	storage.Close()
}
