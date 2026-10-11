package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rpsingh21/torrent-cli/tui/component"
)

func main() {
	p := tea.NewProgram(
		component.InitialModel(),
		tea.WithAltScreen(),       // This enters the full-screen mode (like Claude CLI/vim)
		tea.WithMouseCellMotion(), // Allows scrolling the viewport with the mouse wheel
	)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Fatal error: %v", err)
		os.Exit(1)
	}
}
