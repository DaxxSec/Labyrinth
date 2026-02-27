package cmd

import (
	"fmt"
	"os"

	"github.com/DaxxSec/labyrinth/cli/internal/banner"
	"github.com/spf13/cobra"
)

var installFlag bool

var rootCmd = &cobra.Command{
	Use:   "labyrinth",
	Short: "LABYRINTH — Adversarial Cognitive Portal Trap Architecture",
	Long:  "Deploy, manage, and monitor LABYRINTH portal trap environments.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Print banner for all commands except TUI (it takes over the screen)
		if cmd.Name() != "tui" {
			banner.Print()
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		if installFlag {
			runInstall()
			return
		}
		cmd.Help()
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.Flags().BoolVar(&installFlag, "install", false, "Install labyrinth to ~/.local/bin")
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}

// Logging helpers matching deploy.sh style
func info(msg string) {
	fmt.Printf("  \033[0;32m[+]\033[0m %s\n", msg)
}

func warn(msg string) {
	fmt.Printf("  \033[1;33m[!]\033[0m %s\n", msg)
}

func errMsg(msg string) {
	fmt.Fprintf(os.Stderr, "  \033[0;31m[✗]\033[0m %s\n", msg)
}

func section(title string) {
	fmt.Printf("\n  \033[0;35m━━━ %s ━━━\033[0m\n\n", title)
}
