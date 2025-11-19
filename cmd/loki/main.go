package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yshuman1/loki/internal/tui"
)

func main() {
	// Create TUI model
	m := tui.NewModel()

	// Create Bubbletea program
	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	// Run the program
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running Loki: %v\n", err)
		os.Exit(1)
	}
}
