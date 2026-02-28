package notify

import (
	"fmt"
	"os/exec"
	"runtime"
)

// Enabled controls whether desktop notifications are sent.
var Enabled = true

// Sound controls whether notification sounds play.
var Sound = true

// Send dispatches a desktop notification.
// On macOS, uses osascript. Falls back silently on other platforms.
func Send(title, body string) error {
	if !Enabled {
		return nil
	}

	if runtime.GOOS != "darwin" {
		return nil
	}

	soundClause := ""
	if Sound {
		soundClause = ` sound name "Ping"`
	}

	script := fmt.Sprintf(
		`display notification %q with title %q%s`,
		body, title, soundClause,
	)

	cmd := exec.Command("osascript", "-e", script)
	return cmd.Run()
}
