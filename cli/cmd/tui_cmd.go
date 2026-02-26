package cmd

import (
	"fmt"
	"os"

	"github.com/ItzDaxxy/labyrinth/cli/internal/tui"
	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch the TUI monitoring dashboard",
	Long:  "Launch an interactive terminal dashboard for monitoring LABYRINTH environments.",
	Run:   runTUI,
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}

func runTUI(cmd *cobra.Command, args []string) {
	app := tui.NewApp()
	p := tea.NewProgram(app)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
