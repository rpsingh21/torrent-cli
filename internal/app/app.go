package app

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/rpsingh21/torrent-cli/internal/discovery"
	"github.com/rpsingh21/torrent-cli/internal/peer"
	"github.com/rpsingh21/torrent-cli/internal/piece"
	"github.com/rpsingh21/torrent-cli/internal/storage"
	"github.com/rpsingh21/torrent-cli/internal/torrent"
)

const (
	MAX_PIECE_SIZE = 256 << 20
	OUTPUT_DIR     = "./output/"
)

type managedTorrent struct {
	manager *peer.Manager
	cancel  context.CancelFunc
	paused  bool
}

type App struct {
	Appid     [20]byte
	discovery *discovery.Discovery
	mu        sync.RWMutex
	ctx       context.Context
	managers  map[string]*managedTorrent
}

func New() (*App, error) {
	return &App{
		Appid:    calculate_peer_id(),
		managers: make(map[string]*managedTorrent),
		ctx:      context.Background(),
	}, nil
}

func (a *App) Start(ctx context.Context) error {
	a.mu.Lock()
	a.ctx = ctx
	a.mu.Unlock()

	a.load(ctx)

	<-ctx.Done()
	a.Close()
	return ctx.Err()
}

func (a *App) Add(url string) (string, string, error) {
	a.mu.RLock()
	baseCtx := a.ctx
	a.mu.RUnlock()

	metaInfo, err := getMetaInfo(baseCtx, url)
	if err != nil {
		return "", "", fmt.Errorf("decode metainfo: %w", err)
	}

	manager, err := createManager(metaInfo)
	if err != nil {
		return "", "", fmt.Errorf("create manager: %w", err)
	}

	tCtx, cancel := context.WithCancel(baseCtx)

	a.mu.Lock()
	if _, exists := a.managers[manager.Id]; exists {
		a.mu.Unlock()
		cancel()
		return "", "", fmt.Errorf("torrent %s already exists", manager.Id[:8])
	}
	a.managers[manager.Id] = &managedTorrent{
		manager: manager,
		cancel:  cancel,
		paused:  false,
	}
	a.mu.Unlock()

	go manager.Run(tCtx)
	go a.discovery.Start(tCtx, metaInfo, manager.PeerChan)

	return manager.Id[:8], metaInfo.Name, nil
}

func (a *App) Pause(idPrefix string) (string, error) {
	a.mu.Lock()
	fullID, entry := a.findLocked(idPrefix)
	if entry == nil {
		a.mu.Unlock()
		return "", fmt.Errorf("manager id %q not found", idPrefix)
	}
	if entry.paused {
		a.mu.Unlock()
		return "", fmt.Errorf("manager %s is already paused", fullID[:8])
	}
	entry.paused = true
	cancel := entry.cancel
	a.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	entry.manager.Close()
	return fullID[:8], nil
}

func (a *App) Resume(idPrefix string) (string, error) {
	a.mu.Lock()
	fullID, entry := a.findLocked(idPrefix)
	if entry == nil {
		a.mu.Unlock()
		return "", fmt.Errorf("manager id %q not found", idPrefix)
	}
	if !entry.paused {
		a.mu.Unlock()
		return "", fmt.Errorf("manager %s is already running", fullID[:8])
	}

	tCtx, cancel := context.WithCancel(a.ctx)
	entry.cancel = cancel
	entry.paused = false
	metaInfo := entry.manager.MStat.MetaInfo
	a.mu.Unlock()

	go entry.manager.Run(tCtx)
	if metaInfo != nil {
		go a.discovery.Start(tCtx, metaInfo, entry.manager.PeerChan)
	}
	return fullID[:8], nil
}

func (a *App) Remove(idPrefix string) (string, error) {
	a.mu.Lock()
	fullID, entry := a.findLocked(idPrefix)
	if entry != nil {
		delete(a.managers, fullID)
	}
	a.mu.Unlock()

	if entry == nil {
		return "", fmt.Errorf("manager id %q not found", idPrefix)
	}

	if entry.cancel != nil {
		entry.cancel()
	}
	entry.manager.Close()
	return fullID[:8], nil
}

func (a *App) findLocked(idPrefix string) (string, *managedTorrent) {
	if idPrefix == "" {
		return "", nil
	}
	for fullID, mt := range a.managers {
		if fullID == idPrefix || strings.HasPrefix(fullID, idPrefix) {
			return fullID, mt
		}
	}
	return "", nil
}

func (a *App) GetUpdatedManagerStat() map[string]peer.ManagerStatSnapshot {
	a.mu.RLock()
	defer a.mu.RUnlock()

	statManager := make(map[string]peer.ManagerStatSnapshot, len(a.managers))
	for id, entry := range a.managers {
		statManager[id] = entry.manager.UpdateState()
	}
	return statManager
}

func (a *App) Close() {
	a.mu.Lock()
	defer a.mu.Unlock()

	for id, entry := range a.managers {
		if entry.cancel != nil {
			entry.cancel()
		}
		entry.manager.Close()
		delete(a.managers, id)
	}
}

func (a *App) load(ctx context.Context) {}

func createManager(metaInfo *torrent.MetaInfo) (*peer.Manager, error) {
	store, err := storage.NewFileStorage(metaInfo, filepath.Join(OUTPUT_DIR, metaInfo.Name))
	if err != nil {
		return nil, err
	}

	pieceManager := piece.NewManager(metaInfo, piece.StrategySequential, store)
	return peer.NewManager(metaInfo, pieceManager), nil
}

func getMetaInfo(ctx context.Context, url string) (*torrent.MetaInfo, error) {
	if strings.HasPrefix(url, "magnet") {
		metaInfo, err := torrent.MetaInfoFromMagnetURL(url)
		if err != nil {
			return nil, err
		}

		if err := updateMetainfoFromPeers(ctx, metaInfo); err != nil {
			return nil, fmt.Errorf("discover torrent metadata: %w", err)
		}

		return metaInfo, nil
	}

	return torrent.NewTorrentDetailFromFile(url)
}
