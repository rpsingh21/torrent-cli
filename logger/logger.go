package logger

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/rpsingh21/torrent-cli/internal/config"
	"gopkg.in/natefinch/lumberjack.v2"
)

// trimSource extracts "parent_dir/file.go" without heap allocations.
func trimSource(path string) string {
	last := strings.LastIndexByte(path, filepath.Separator)
	if last <= 0 {
		return path
	}
	secondLast := strings.LastIndexByte(path[:last], filepath.Separator)
	if secondLast == -1 {
		return path
	}
	return path[secondLast+1:]
}

// Setup initializes slog according to the loaded json config.
func Setup(cfg *config.AppConfig) *slog.Logger {
	var level slog.Level
	switch strings.ToUpper(cfg.Log.Level) {
	case "DEBUG":
		level = slog.LevelDebug
	case "WARN":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	// Route outputs
	var writer io.Writer = os.Stdout
	if cfg.Log.FilePath != "" {
		rotator := &lumberjack.Logger{
			Filename:   cfg.Log.FilePath,
			MaxSize:    5, // megabytes per file
			MaxBackups: 3, // keep up to 3 old backups
			MaxAge:     3, // days
			Compress:   true,
		}

		writer = rotator
	}

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: cfg.Log.AddSource,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.SourceKey {
				if src, ok := a.Value.Any().(*slog.Source); ok && src != nil {
					src.File = trimSource(src.File)
				}
			}
			return a
		},
	}

	var handler slog.Handler
	if cfg.Env == "prod" {
		handler = slog.NewJSONHandler(writer, opts)
	} else {
		handler = slog.NewTextHandler(writer, opts)
	}

	logger := slog.New(handler).With(
		slog.String("env", cfg.Env),
	)

	slog.SetDefault(logger)
	return logger
}
