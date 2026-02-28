package cmd

import (
	"fmt"
	"os"

	"github.com/DaxxSec/labyrinth/cli/internal/notify"
	"github.com/DaxxSec/labyrinth/cli/internal/tui"
	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
)

var (
	enableNotify bool
	webhookURL   string
	tuiEnvName   string
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch the TUI monitoring dashboard",
	Long:  "Launch an interactive terminal dashboard for monitoring LABYRINTH environments.",
	Run:   runTUI,
}

func init() {
	tuiCmd.Flags().BoolVar(&enableNotify, "notify", true, "Enable desktop notifications")
	tuiCmd.Flags().StringVar(&webhookURL, "webhook", "", "Webhook URL for notifications (Slack/Discord)")
	tuiCmd.Flags().StringVar(&tuiEnvName, "env", "", "Target a specific environment by name")
	rootCmd.AddCommand(tuiCmd)
}

func runTUI(cmd *cobra.Command, args []string) {
	// Configure notifications
	notify.Enabled = enableNotify
	if webhookURL != "" {
		notify.Webhook = notify.WebhookConfig{
			URL:     webhookURL,
			Enabled: true,
		}
	}
	// Also check env var
	if envURL := os.Getenv("LABYRINTH_WEBHOOK_URL"); envURL != "" && webhookURL == "" {
		notify.Webhook = notify.WebhookConfig{
			URL:     envURL,
			Enabled: true,
		}
	}

	app := tui.NewApp(tuiEnvName)
	p := tea.NewProgram(app)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
