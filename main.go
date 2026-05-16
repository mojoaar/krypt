package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mojoaar/krypt/internal/cli"
	"github.com/mojoaar/krypt/internal/ui"
)

var version = "dev"

func main() {
	args := os.Args[1:]

	// --version / -v
	for _, arg := range args {
		if arg == "--version" || arg == "-v" {
			fmt.Println("krypt " + version)
			return
		}
	}

	// CLI subcommands (get, list, help)
	if cli.Run(args) {
		return
	}

	// Launch TUI
	app := ui.NewApp(version)
	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "krypt: %v\n", err)
		os.Exit(1)
	}
}
