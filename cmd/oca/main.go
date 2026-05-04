package main

import (
	"fmt"
	"os"

	_ "modernc.org/sqlite"
)

func main() {
	if err := execute(); err != nil {
		exitCode := 1
		if ec, ok := err.(interface{ ExitCode() int }); ok {
			exitCode = ec.ExitCode()
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitCode)
	}
}
