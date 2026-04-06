// Package main is the entry point for the oca (OpenCode Advance) CLI.
//
// This file is a scaffold placeholder. The real implementation lands in
// Phase 0 per docs/proposals/phases.md — at which point this file will be
// replaced with a cobra-based command tree that prints the wordmark,
// parses stack.toml, and dispatches to internal/ subpackages.
//
// Do not add real logic here until Phase 0 begins as an ADV change.
package main

import "fmt"

const scaffoldVersion = "0.0.0-scaffold"

func main() {
	fmt.Println("OpenCode Advance — scaffold build")
	fmt.Printf("version: %s\n", scaffoldVersion)
	fmt.Println()
	fmt.Println("This is a pre-implementation scaffold. The real CLI lands in Phase 0.")
	fmt.Println("See docs/proposals/phases.md for the implementation plan.")
}
