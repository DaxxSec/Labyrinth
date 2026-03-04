package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/DaxxSec/labyrinth/cli/internal/api"
	"github.com/DaxxSec/labyrinth/cli/internal/forensics"
	"github.com/DaxxSec/labyrinth/cli/internal/report"
	"github.com/spf13/cobra"
)

var (
	reportFormat string
	reportOutput string
	reportAll    bool
)

var reportCmd = &cobra.Command{
	Use:   "report [session-id]",
	Short: "Generate a forensic attack report",
	Long: `Generate a structured forensic report from captured session data.

Reports include executive summary, MITRE ATT&CK timeline, credential analysis,
service interaction logs, attack graphs (Mermaid), and effectiveness assessment.

Examples:
  labyrinth report                            # latest session
  labyrinth report LAB-abc123                  # specific session
  labyrinth report --all                       # all sessions
  labyrinth report --format md -o report.md    # export markdown
  labyrinth report --format json               # JSON to stdout`,
	Run: runReport,
}

func init() {
	reportCmd.Flags().StringVar(&reportFormat, "format", "terminal", "Output format: terminal, md, json")
	reportCmd.Flags().StringVarP(&reportOutput, "output", "o", "", "Write output to file instead of stdout")
	reportCmd.Flags().BoolVar(&reportAll, "all", false, "Generate reports for all sessions")
	rootCmd.AddCommand(reportCmd)
}

func runReport(cmd *cobra.Command, args []string) {
	// Resolve data source: API first, file fallback
	var client *api.Client
	var reader *forensics.Reader

	dashURL := "http://localhost:9000"
	c := api.NewClient(dashURL)
	if c.Healthy() {
		client = c
	} else {
		// File-based fallback
		forensicsDir := "/var/labyrinth/forensics"
		if _, err := os.Stat(forensicsDir); err == nil {
			reader = forensics.NewReader(forensicsDir)
		} else {
			errMsg("Dashboard API not reachable and no forensic data found at " + forensicsDir)
			warn("Is the environment running? Check with: labyrinth status")
			os.Exit(1)
		}
	}

	// Resolve session IDs
	sessionIDs, err := resolveSessionIDs(client, reader, args)
	if err != nil {
		errMsg(err.Error())
		os.Exit(1)
	}

	if len(sessionIDs) == 0 {
		warn("No sessions found")
		os.Exit(0)
	}

	// Build and render reports
	var allOutput strings.Builder
	for i, sid := range sessionIDs {
		r, err := report.Build(client, reader, sid)
		if err != nil {
			errMsg(fmt.Sprintf("Failed to build report for %s: %v", sid, err))
			continue
		}

		switch reportFormat {
		case "json":
			data, err := json.MarshalIndent(r, "", "  ")
			if err != nil {
				errMsg(fmt.Sprintf("JSON marshal failed: %v", err))
				continue
			}
			allOutput.Write(data)
			if i < len(sessionIDs)-1 {
				allOutput.WriteString("\n")
			}
		case "md", "markdown":
			allOutput.WriteString(report.RenderMarkdown(r))
			if i < len(sessionIDs)-1 {
				allOutput.WriteString("\n---\n\n")
			}
		default:
			allOutput.WriteString(report.RenderTerminal(r))
		}
	}

	output := allOutput.String()

	if reportOutput != "" {
		if err := os.WriteFile(reportOutput, []byte(output), 0644); err != nil {
			errMsg(fmt.Sprintf("Failed to write output file: %v", err))
			os.Exit(1)
		}
		info(fmt.Sprintf("Report written to %s", reportOutput))
	} else {
		fmt.Print(output)
	}
}

func resolveSessionIDs(client *api.Client, reader *forensics.Reader, args []string) ([]string, error) {
	// Specific session from args
	if len(args) > 0 && !reportAll {
		return args, nil
	}

	// List all sessions
	var sessions []api.SessionEntry
	if client != nil {
		s, err := client.FetchSessions()
		if err != nil {
			return nil, fmt.Errorf("fetch sessions: %w", err)
		}
		sessions = s
	} else if reader != nil {
		s, err := reader.ReadSessions()
		if err != nil {
			return nil, fmt.Errorf("read sessions: %w", err)
		}
		sessions = s
	}

	if len(sessions) == 0 {
		return nil, nil
	}

	// Sort by file name (includes timestamp) descending
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].File > sessions[j].File
	})

	if reportAll {
		var ids []string
		for _, s := range sessions {
			ids = append(ids, sessionFileToID(s.File))
		}
		return ids, nil
	}

	// Default: latest session
	return []string{sessionFileToID(sessions[0].File)}, nil
}

func sessionFileToID(file string) string {
	// Strip .jsonl extension to get session ID
	return strings.TrimSuffix(file, ".jsonl")
}
