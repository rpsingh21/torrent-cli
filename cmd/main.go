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
	"sync"
	"syscall"

	"github.com/rpsingh21/torrent-cli/internal/app"
	"github.com/rpsingh21/torrent-cli/internal/config"
	"github.com/rpsingh21/torrent-cli/internal/peer"
)

func main() {
	pprof := flag.Bool("pprof", false, "Set to enbale profiler")
	url := flag.String("tf", "", "torrent file or mannet link")
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

	cnf, err := config.New()
	if err != nil {
		panic(err)
	}

	app, err := app.New(cnf)
	var wg sync.WaitGroup

	statChan := make(chan peer.ManagerStatSnapshot)
	defer close(statChan)

	wg.Go(func() {
		app.Start(ctx)
	})

	wg.Go(func() {
		readSrat(ctx, statChan)
	})

	app.Add(*url)

	wg.Wait()
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
		os.Remove("dev.log")
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

const mib = 1 << 20
const kib = 1 << 10

func readSrat(ctx context.Context, ch <-chan peer.ManagerStatSnapshot) {
	var preDownloaded, preUploaded int64

	for {
		select {
		case <-ctx.Done():
			return
		case ms := <-ch:
			printStat(len(ms.PeerStat), ms.Downloaded, ms.Uploaded, 0, 0, ms.Completed, ms.Inprogress,
				ms.Downloaded-preDownloaded, ms.Uploaded-preUploaded)
			preDownloaded = ms.Downloaded
			preUploaded = ms.Uploaded
		}
	}

}

func printStat(totalPeer int, downloaded, uploaded, totalReqs, totalErrs int64, completed, inprogress int, dr, ur int64) {

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
