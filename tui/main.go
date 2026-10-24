package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rpsingh21/torrent-cli/internal/app"
	"github.com/rpsingh21/torrent-cli/tui/component"
)

func main() {
	// 1. Redirect logs to file so background logs never corrupt the TUI alt-screen
	logFile, err := os.OpenFile("torrent-cli.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err == nil {
		defer logFile.Close()
		log.SetOutput(logFile)
		slog.SetDefault(slog.New(slog.NewTextHandler(logFile, nil)))
	}

	// 2. Root context listening for OS SIGINT / SIGTERM
	ctx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	engine, err := app.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize torrent engine: %v\n", err)
		os.Exit(1)
	}

	// 3. Start background engine with WaitGroup tracking
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := engine.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
			slog.Error("engine stopped with error", "err", err)
		}
	}()

	// 4. Bind Bubble Tea to the same root context
	p := tea.NewProgram(
		component.InitialModel(engine),
		tea.WithContext(ctx),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	_, runErr := p.Run()

	// 5. Cancel context when TUI exits (/exit, Esc, Ctrl+C) to trigger engine.closeAll()
	stopSignals()

	// 6. Wait for managers & file storage to close gracefully with a hard timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		fmt.Fprintln(os.Stderr, "shutdown timed out after 5s, forcing exit")
	}

	if runErr != nil && !errors.Is(runErr, tea.ErrProgramKilled) {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", runErr)
		os.Exit(1)
	}
}
