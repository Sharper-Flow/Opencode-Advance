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
