package dashboard

import (
	"fmt"
	"os/exec"
	"runtime"
)

// browserCmd returns the exec.Cmd to open a URL in the default browser for the given GOOS.
func browserCmd(goos, url string) *exec.Cmd {
	switch goos {
	case "darwin":
		return exec.Command("open", url)
	case "windows":
		return exec.Command("cmd", "/c", "start", url)
	default: // linux and everything else
		return exec.Command("xdg-open", url)
	}
}

// openBrowser attempts to open the given URL in the default browser.
// Returns an error if the browser command fails to start.
func openBrowser(url string) error {
	cmd := browserCmd(runtime.GOOS, url)
	if cmd == nil {
		return fmt.Errorf("no browser command for GOOS=%s", runtime.GOOS)
	}
	return cmd.Start()
}
