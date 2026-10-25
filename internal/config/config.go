package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type LogConfig struct {
	Level     string `json:"level"`      // "DEBUG", "INFO", "WARN", "ERROR"
	Format    string `json:"format"`     // "json" or "text"
	FilePath  string `json:"file_path"`  // destination file path
	AddSource bool   `json:"add_source"` // include file:line info
}

type AppConfig struct {
	Env         string    `json:"env"`
	AppDir      string    `json:"app_dir"`
	DownloadDir string    `json:"download_dir"`
	Log         LogConfig `json:"log"`
}

func New() (*AppConfig, error) {
	userDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	return &AppConfig{
		Env:         "dev",
		AppDir:      filepath.Join(userDir, ".local/share/torrent-cli"),
		DownloadDir: filepath.Join(userDir, "Downloads/torrent-cli"),
		Log: LogConfig{
			Level:     "INFO",
			Format:    "text",
			FilePath:  filepath.Join(userDir, ".local/share/torrent-cli/logs/app.log"),
			AddSource: true,
		},
	}, nil
}

func LoadConfig(path string) (*AppConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing json: %w", err)
	}

	return &cfg, nil
}
