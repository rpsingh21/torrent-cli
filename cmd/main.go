package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/rpsingh21/torrent-cli/internal/app"
)

func main() {

	// Only Debug
	go func() {
		log.Println("pprof: http://localhost:6060/debug/pprof/")
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	torrentFilePath := flag.String("tf", "", "Path of torrent file")
	out := flag.String("out", "./output", "Dir where want to store downloaded files")
	flag.Parse()

	if *torrentFilePath == "" {
		log.Fatal("Please provide a torrent file with -tf")
	}

	outputDir, err := filepath.Abs(*out)
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app, err := app.NewAppFromTorrentFile(*torrentFilePath, outputDir)
	if err != nil {
		log.Fatalf("Failed to load torrent file %q: %v", *torrentFilePath, err)
	}

	if err := app.Download(ctx); err != nil {
		log.Fatalf("Download failed: %v", err)
	}

	log.Printf("Torrent downloaded successfully: %s", outputDir)
}
