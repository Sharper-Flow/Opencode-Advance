// Command session-debug is a development-only diagnostic utility for
// inspecting raw `tmux list-sessions` output and the subprocess wrapper's
// classification of it (ExitClass, ExitCode, output bytes). It is NOT
// part of the installed `oca` CLI surface — it lives outside cmd/oca/
// on purpose so it cannot be invoked through the user-facing entry point.
//
// Build with `go run ./cmd/session-debug` while debugging session manager
// issues. Add new prints freely; remove temporary scratch code before
// committing if it is purely throwaway. The file is intentionally short
// and stays that way.
package main

import (
	"context"
	"fmt"

	"github.com/Sharper-Flow/Opencode-Advance/internal/subprocess"
)

func main() {
	// Direct subprocess call to see what Output actually contains
	res, err := subprocess.Run(context.Background(), subprocess.Cmd{
		Name: "tmux",
		Args: []string{"-L", "ocait-debug-x", "list-sessions", "-F", "#{session_name}\t#{session_attached}"},
	})
	fmt.Printf("err: %v\n", err)
	fmt.Printf("ExitClass: %s\n", res.ExitClass)
	fmt.Printf("ExitCode: %d\n", res.ExitCode)
	fmt.Printf("Output bytes: %d\n", len(res.Output))
	fmt.Printf("Output: %q\n", string(res.Output))
}
