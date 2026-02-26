package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func runInstall() {
	home, err := os.UserHomeDir()
	if err != nil {
		errMsg(fmt.Sprintf("Cannot determine home directory: %v", err))
		os.Exit(1)
	}

	installDir := filepath.Join(home, ".local", "bin")
	installPath := filepath.Join(installDir, "labyrinth")

	// Ensure directory exists
	if err := os.MkdirAll(installDir, 0755); err != nil {
		errMsg(fmt.Sprintf("Cannot create %s: %v", installDir, err))
		os.Exit(1)
	}

	// Get current binary path
	exe, err := os.Executable()
	if err != nil {
		errMsg(fmt.Sprintf("Cannot determine current binary path: %v", err))
		os.Exit(1)
	}

	// Resolve symlinks
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		errMsg(fmt.Sprintf("Cannot resolve binary path: %v", err))
		os.Exit(1)
	}

	// Copy binary
	src, err := os.Open(exe)
	if err != nil {
		errMsg(fmt.Sprintf("Cannot open current binary: %v", err))
		os.Exit(1)
	}
	defer src.Close()

	dst, err := os.OpenFile(installPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		errMsg(fmt.Sprintf("Cannot write to %s: %v", installPath, err))
		os.Exit(1)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		errMsg(fmt.Sprintf("Copy failed: %v", err))
		os.Exit(1)
	}

	info(fmt.Sprintf("Installed labyrinth to %s", installPath))
	fmt.Println()

	// Check if ~/.local/bin is in PATH
	pathEnv := os.Getenv("PATH")
	if !filepath.IsAbs(installDir) {
		return
	}
	found := false
	for _, p := range filepath.SplitList(pathEnv) {
		if p == installDir {
			found = true
			break
		}
	}
	if !found {
		warn(fmt.Sprintf("%s is not in your PATH", installDir))
		dim := "\033[2m"
		reset := "\033[0m"
		fmt.Printf("  %sAdd to your shell profile:%s\n", dim, reset)
		fmt.Printf("  %sexport PATH=\"%s:$PATH\"%s\n", dim, installDir, reset)
		fmt.Println()
	} else {
		info("labyrinth is ready — run 'labyrinth --help' from anywhere")
	}
}
