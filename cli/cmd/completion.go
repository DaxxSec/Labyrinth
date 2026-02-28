package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion <bash|zsh|fish>",
	Short: "Generate shell autocompletion",
	Long: `Generate shell completion scripts for labyrinth.

  labyrinth completion bash       Print bash completion script
  labyrinth completion zsh        Print zsh completion script
  labyrinth completion fish       Print fish completion script
  labyrinth completion install    Auto-install for your current shell`,
	ValidArgs: []string{"bash", "zsh", "fish", "install"},
	Args:      cobra.ExactArgs(1),
	Run:       runCompletion,
}

func init() {
	rootCmd.AddCommand(completionCmd)
}

func runCompletion(cmd *cobra.Command, args []string) {
	switch args[0] {
	case "bash":
		rootCmd.GenBashCompletion(os.Stdout)
	case "zsh":
		rootCmd.GenZshCompletion(os.Stdout)
	case "fish":
		rootCmd.GenFishCompletion(os.Stdout, true)
	case "install":
		installCompletion()
	default:
		errMsg(fmt.Sprintf("Unknown shell: %s (supported: bash, zsh, fish)", args[0]))
		os.Exit(1)
	}
}

func installCompletion() {
	shell := detectShell()
	if shell == "" {
		errMsg("Could not detect your shell")
		fmt.Println()
		dim := "\033[2m"
		reset := "\033[0m"
		fmt.Printf("  %sManually generate completion with:%s\n", dim, reset)
		fmt.Printf("  %s  labyrinth completion bash%s\n", dim, reset)
		fmt.Printf("  %s  labyrinth completion zsh%s\n", dim, reset)
		fmt.Printf("  %s  labyrinth completion fish%s\n", dim, reset)
		os.Exit(1)
	}

	section("Installing Shell Completion")

	switch shell {
	case "bash":
		installBashCompletion()
	case "zsh":
		installZshCompletion()
	case "fish":
		installFishCompletion()
	}
}

func detectShell() string {
	shellPath := os.Getenv("SHELL")
	base := filepath.Base(shellPath)
	switch base {
	case "bash":
		return "bash"
	case "zsh":
		return "zsh"
	case "fish":
		return "fish"
	}
	return ""
}

func installBashCompletion() {
	home, _ := os.UserHomeDir()

	// Try system directory first, fall back to user rc file
	dirs := []string{
		"/usr/local/etc/bash_completion.d",
		"/etc/bash_completion.d",
	}

	for _, dir := range dirs {
		if fi, err := os.Stat(dir); err == nil && fi.IsDir() {
			dst := filepath.Join(dir, "labyrinth")
			f, err := os.Create(dst)
			if err == nil {
				rootCmd.GenBashCompletion(f)
				f.Close()
				info(fmt.Sprintf("Installed to %s", dst))
				fmt.Println()
				info("Restart your shell or run: source " + dst)
				return
			}
		}
	}

	// Fall back to ~/.bash_completion.d/
	dir := filepath.Join(home, ".bash_completion.d")
	os.MkdirAll(dir, 0755)
	dst := filepath.Join(dir, "labyrinth")
	f, err := os.Create(dst)
	if err != nil {
		errMsg(fmt.Sprintf("Could not write %s: %v", dst, err))
		os.Exit(1)
	}
	rootCmd.GenBashCompletion(f)
	f.Close()
	info(fmt.Sprintf("Installed to %s", dst))

	// Ensure it's sourced from .bashrc
	bashrc := filepath.Join(home, ".bashrc")
	sourceLine := fmt.Sprintf("\n# labyrinth shell completion\n[ -f %s ] && source %s\n", dst, dst)
	if data, err := os.ReadFile(bashrc); err != nil || !contains(string(data), dst) {
		appendToFile(bashrc, sourceLine)
		info("Added source line to ~/.bashrc")
	}
	fmt.Println()
	info("Restart your shell or run: source " + dst)
}

func installZshCompletion() {
	home, _ := os.UserHomeDir()

	// Use ~/.zsh/completions (works without modifying fpath if we add it)
	dir := filepath.Join(home, ".zsh", "completions")
	os.MkdirAll(dir, 0755)
	dst := filepath.Join(dir, "_labyrinth")
	f, err := os.Create(dst)
	if err != nil {
		errMsg(fmt.Sprintf("Could not write %s: %v", dst, err))
		os.Exit(1)
	}
	rootCmd.GenZshCompletion(f)
	f.Close()
	info(fmt.Sprintf("Installed to %s", dst))

	// Ensure fpath includes our dir and compinit is called
	zshrc := filepath.Join(home, ".zshrc")
	fpathLine := fmt.Sprintf("\n# labyrinth shell completion\nfpath=(~/.zsh/completions $fpath)\nautoload -Uz compinit && compinit\n")
	if data, err := os.ReadFile(zshrc); err != nil || !contains(string(data), "/.zsh/completions") {
		appendToFile(zshrc, fpathLine)
		info("Added completion setup to ~/.zshrc")
	}
	fmt.Println()
	info("Restart your shell or run: exec zsh")
}

func installFishCompletion() {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".config", "fish", "completions")
	os.MkdirAll(dir, 0755)
	dst := filepath.Join(dir, "labyrinth.fish")
	f, err := os.Create(dst)
	if err != nil {
		errMsg(fmt.Sprintf("Could not write %s: %v", dst, err))
		os.Exit(1)
	}
	rootCmd.GenFishCompletion(f, true)
	f.Close()
	info(fmt.Sprintf("Installed to %s", dst))
	fmt.Println()
	info("Restart your shell — fish picks it up automatically")
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && stringContains(s, substr)
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func appendToFile(path, content string) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	f.WriteString(content)
}
