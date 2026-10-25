package logger

import (
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/rpsingh21/torrent-cli/internal/config"
)

func TestParseConfig(t *testing.T) {
	dir, err := os.Getwd()

	if err != nil {
		log.Print(err)
	}

	baseDir := filepath.Dir(dir)
	configPath := filepath.Join(baseDir, "config/config.dev.json")

	cnf, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("error while loding config: %v", err)
	}

	log.Printf("config : %+v", cnf)
}
