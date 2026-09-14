package main

import (
	"log"
	"os"

	"github.com/rpsingh21/torrent-cli/internal/discovery/tracker"
	"github.com/rpsingh21/torrent-cli/internal/peer"
	"github.com/rpsingh21/torrent-cli/internal/torrent"
)

func main() {
	// This is the entry point of the application.
	// You can add your application logic here.
	torrentFilePath := os.Args[1]
	log.Println("Torrent file path:", torrentFilePath)

	metaInfo, err := torrent.NewTorrentDetailFromFile(torrentFilePath)
	if err != nil {
		log.Printf("Failed to load torrent file %v", metaInfo)
	}

	tracker := tracker.NewTracker(metaInfo)
	// piece := piece.NewManager(metaInfo)
	resp, err := tracker.RequestPeers("started")
	if err != nil {
		log.Print("Failed to load peers")
	}
	done := make(chan bool)
	manager := peer.NewManager(metaInfo.InfoHash, tracker.PeerId)
	manager.AddPeers(resp.Peers)
	manager.Run()
	<-done

}
