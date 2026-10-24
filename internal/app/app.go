package app

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/rpsingh21/torrent-cli/internal/discovery"
	"github.com/rpsingh21/torrent-cli/internal/peer"
	"github.com/rpsingh21/torrent-cli/internal/piece"
	"github.com/rpsingh21/torrent-cli/internal/storage"
	"github.com/rpsingh21/torrent-cli/internal/torrent"
)

const MAX_PIECE_SIZE = 256 << 20
const OUTPUT_DIR = "./output/"

type AppCMDId uint8

const (
	ADD AppCMDId = iota
	REMOVE
	PUSH
)

type AppCmd struct {
	id  AppCMDId
	arg string
}

type App struct {
	Appid     [20]byte
	discovery *discovery.Discovery
	managers  map[string]*peer.Manager
	cmdChan   chan AppCmd
}

func New() (*App, error) {
	return &App{
		Appid:     calculate_peer_id(),
		discovery: discovery.New(),
		managers:  make(map[string]*peer.Manager),
		cmdChan:   make(chan AppCmd, 8),
	}, nil
}

func (a *App) Start(ctx context.Context) error {
	a.load(ctx)
	metricTicker := time.NewTicker(1 * time.Second)
	defer metricTicker.Stop()

	// var preDownloaded, preUploaded int

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case cmd := <-a.cmdChan:
			switch cmd.id {
			case ADD:
				a.addNewManager(ctx, cmd.arg)
			case REMOVE:
				a.removeManager(cmd.arg)
			case PUSH:
				a.pushManager(cmd.arg)
			}

		}
	}
}

func (a *App) Add(url string) {
	a.cmdChan <- AppCmd{id: ADD, arg: url}
}

func (a *App) Push(id string) {
	a.cmdChan <- AppCmd{id: ADD, arg: id}
}

func (a *App) Remove(id string) {
	a.cmdChan <- AppCmd{id: ADD, arg: id}
}

func (a *App) GetUpdatedManagerStat() map[string]peer.ManagerStatSnapshot {
	// a.mu.RLock()
	// defer a.mu.RUnlock()

	statManager := make(map[string]peer.ManagerStatSnapshot, len(a.managers))
	for _, p := range a.managers {
		statManager[p.Id] = p.UpdateState()
	}
	return statManager
}

func (a *App) Close() {
	close(a.cmdChan)

	for _, v := range a.managers {
		v.Close()
	}
}

// Load old config.
// Load old comoleted or inprocress torrent.
func (a *App) load(ctx context.Context) {
	// Todo: will imp
}

func (a *App) addNewManager(ctx context.Context, url string) error {
	metaInfo, err := getMetaInfo(ctx, url)
	if err != nil {
		log.Printf("metainfo decoing is failed: %v", url)
		return err
	}

	manager, err := createManager(metaInfo)
	if err != nil {
		log.Printf("failed to create manager for %v", url)
		return err
	}

	go manager.Run(ctx)
	go a.discovery.Start(ctx, metaInfo, manager.PeerChan)

	a.managers[string(metaInfo.InfoHash[:])] = manager
	return nil
}

func (a *App) pushManager(id string) error {
	// Todo imp later
	log.Printf("pushed manager (debug only): %v", id)
	return nil
}

func (a *App) removeManager(id string) error {
	m, ok := a.managers[id]
	if !ok {
		return fmt.Errorf("Manager id doesn't found")
	}

	m.Close()
	delete(a.managers, id)
	return nil
}

func createManager(metaInfo *torrent.MetaInfo) (*peer.Manager, error) {
	store, err := storage.NewFileStorage(metaInfo, filepath.Join(OUTPUT_DIR, metaInfo.Name))
	if err != nil {
		return nil, err
	}

	pieceManager := piece.NewManager(metaInfo, piece.StrategySequential, store)
	peerManager := peer.NewManager(metaInfo, pieceManager)

	return peerManager, nil
}

func getMetaInfo(ctx context.Context, url string) (*torrent.MetaInfo, error) {
	if strings.HasPrefix(url, "magnet") {
		metaInfo, err := torrent.MetaInfoFromMagnetURL(url)
		if err != nil {
			log.Printf("failed to decode metaurl: %v", url)
			return nil, err
		}

		if err := updateMetainfoFromPeers(ctx, metaInfo); err != nil {
			return nil, fmt.Errorf("discover torrent metadata: %w", err)
		}

		return metaInfo, nil
	}

	return torrent.NewTorrentDetailFromFile(url)
}
