package main

import (
	"flag"
	"log"

	"github.com/rpsingh21/torrent-cli/internal/app"
)

// This is the entry point of the cmd application.
func main() {

	torrentFilePath := flag.String("tf", "", "Path of torrent file")
	ouputDir := flag.String("out", "./output", "Dir where want to store dowloaded files")
	flag.Parse()

	switch {
	case torrentFilePath != nil:
		app := app.NewAppFromTorrentFile(*torrentFilePath, *ouputDir)
		app.Download()
	default:
		log.Fatalln("Please provide valid torrent provide(torrenfile)")
	}

	log.Println("Torrent downloaded successfully!:", ouputDir)
}
