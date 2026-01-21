package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"mangahub/internal/tui"
	"mangahub/internal/tui/config"
)

func main() {
	// Load configuration
	cfg, err := config.Load("")
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		fmt.Println("Using default configuration...")
		cfg = config.Default()
	}

	// Create TUI application
	app := tui.New(cfg)

	// Run the Bubble Tea program
	p := tea.NewProgram(
		app,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
