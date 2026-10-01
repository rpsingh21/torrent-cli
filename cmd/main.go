package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/rpsingh21/torrent-cli/internal/app"
)

func main() {
	pprof := flag.Bool("pprof", false, "Set to enbale profiler")
	torrentFilePath := flag.String("tf", "", "Path of torrent file")
	magnetLink := flag.String("m", "", "Magnet link")
	out := flag.String("out", "./output", "Dir where want to store downloaded files")
	flag.Parse()

	// Only Debug enabled if pass -pprof flag
	if *pprof {
		go func() {
			log.Println("pprof: http://localhost:6060/debug/pprof/")
			log.Println(http.ListenAndServe("localhost:6060", nil))
		}()
	}

	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "dev"
	}

	// 1. Initialize the environment-specific logger
	logger := setupLogger(env)

	// 2. Optional: Set it as the global default if you prefer not to use DI
	slog.SetDefault(logger)

	outputDir, err := filepath.Abs(*out)
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if *torrentFilePath != "" {
		app, err := app.NewAppFromTorrentFile(*torrentFilePath, outputDir)
		if err != nil {
			log.Fatalf("Failed to load torrent file %q: %v", *torrentFilePath, err)
		}

		if err := app.Download(ctx); err != nil {
			log.Fatalf("Download failed: %v", err)
		}
	} else if *magnetLink != "" {
		app, err := app.NewAppFromMagnetLink(ctx, *magnetLink, outputDir)
		if err != nil {
			log.Fatalf("Failed to load torrent file %q: %v", *torrentFilePath, err)
		}

		if err := app.Download(ctx); err != nil {
			log.Fatalf("Download failed: %v", err)
		}
	} else {
		log.Fatalf("Torrent file or Magnet link require")
	}

	slog.Info(fmt.Sprintf("Torrent downloaded successfully: %s", outputDir))

}

func setupLogger(env string) *slog.Logger {
	var handler slog.Handler

	switch env {
	case "prod":
		// Production: JSON format, Info level minimum
		opts := &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}
		handler = slog.NewJSONHandler(os.Stdout, opts)

	case "test":
		// Test: Discard logs entirely to keep test output clean
		// (Or use a custom buffer if you need to assert log output)
		opts := &slog.HandlerOptions{
			Level: slog.LevelError,
		}
		handler = slog.NewTextHandler(os.Stdout, opts) // Or io.Discard

	default:
		file, err := os.OpenFile("dev.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("failed to open log file: %v", err)
		}

		// Local: Text format, Debug level minimum, file output only
		opts := &slog.HandlerOptions{
			Level:     slog.LevelDebug,
			AddSource: true,
		}

		// Pass 'file' directly to the handler
		handler = slog.NewTextHandler(file, opts)
	}

	return slog.New(handler)
}
