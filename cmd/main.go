package main

import (
	"flag"
	"log"

	"github.com/rpsingh21/torrent-cli/internal/discovery/tracker"
	"github.com/rpsingh21/torrent-cli/internal/peer"
	"github.com/rpsingh21/torrent-cli/internal/torrent"
)

// This is the entry point of the cmd application.
func main() {

	torrentFilePath := flag.String("-tf", "", "Path of torrent file")
	ouputDir := flag.String("out", "./output", "Dir where want to store dowloaded files")

	switch {
	case torrentFilePath != nil:
		metaInfo, err := torrent.NewTorrentDetailFromFile(*torrentFilePath)
		if err != nil {
			log.Fatalf("Failed to load torrent file %+v", err)
		}

		stratTorrentDowload(metaInfo, *ouputDir)

	default:
		log.Fatalln("Please provide valid torrent provide(torrenfile)")
	}

	log.Println("Torrent file path:", torrentFilePath)
}

func stratTorrentDowload(metaInfo *torrent.MetaInfo, outputDir string) {
	tracker := tracker.NewTracker(metaInfo)

	resp, err := tracker.RequestPeers("started")
	if err != nil {
		log.Fatal("Failed to load peers", outputDir)
	}
	manager := peer.NewManager(metaInfo.InfoHash, tracker.PeerId)
	manager.Run()
}
