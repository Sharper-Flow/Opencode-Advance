package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Sharper-Flow/Opencode-Advance/brand"
)

func TestRootCommandShowsHelpWhenNoSubcommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := newRootCmd(commandOptions{
		Stdout: &stdout,
		Stderr: &stderr,
		Version: VersionInfo{
			Version: "0.1.0-test",
		},
		Environment: brand.Environment{IsTTY: false, Term: "xterm"},
	})

	cmd.SetArgs(nil)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	got := stdout.String() + stderr.String()
	if !strings.Contains(got, "OpenCode Advance") {
		t.Fatalf("help output missing app name: %q", got)
	}

	if !strings.Contains(got, "version") {
		t.Fatalf("help output missing version command: %q", got)
	}

	if strings.Contains(strings.ToLower(got), "scaffold") {
		t.Fatalf("help output still contains scaffold text: %q", got)
	}
}

func TestVersionCommandPrintsBrandedBanner(t *testing.T) {
	var stdout bytes.Buffer

	cmd := newRootCmd(commandOptions{
		Stdout: &stdout,
		Stderr: &stdout,
		Version: VersionInfo{
			Version: "0.1.0-test",
		},
		Environment: brand.Environment{IsTTY: false, Term: "xterm"},
	})

	cmd.SetArgs([]string{"version"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	got := stdout.String()
	if !strings.Contains(got, "░█▀█░█▀█░█▀▀") {
		t.Fatalf("version output missing wordmark: %q", got)
	}

	if !strings.Contains(got, "version: 0.1.0-test") {
		t.Fatalf("version output missing version line: %q", got)
	}

	if strings.Contains(strings.ToLower(got), "scaffold") {
		t.Fatalf("version output still contains scaffold text: %q", got)
	}
}

func TestVersionCommandUsesTruecolorWhenSupported(t *testing.T) {
	var stdout bytes.Buffer

	cmd := newRootCmd(commandOptions{
		Stdout: &stdout,
		Stderr: &stdout,
		Version: VersionInfo{
			Version: "0.1.0-test",
		},
		Environment: brand.Environment{IsTTY: true, Term: "xterm-256color", ColorTerm: "truecolor"},
	})

	cmd.SetArgs([]string{"version"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	got := stdout.String()
	if !strings.Contains(got, "\x1b[38;2;232;230;227m") {
		t.Fatalf("version output missing truecolor ivory sequence: %q", got)
	}

	if !strings.Contains(got, "\x1b[38;2;108;122;184m") {
		t.Fatalf("version output missing truecolor indigo sequence: %q", got)
	}
}

func TestVersionCommandRespectsNoColor(t *testing.T) {
	var stdout bytes.Buffer

	cmd := newRootCmd(commandOptions{
		Stdout: &stdout,
		Stderr: &stdout,
		Version: VersionInfo{
			Version: "0.1.0-test",
		},
		Environment: brand.Environment{IsTTY: true, NoColor: true, Term: "xterm-256color", ColorTerm: "truecolor"},
	})

	cmd.SetArgs([]string{"version"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	got := stdout.String()
	if strings.Contains(got, "\x1b[") {
		t.Fatalf("version output should not contain ANSI sequences when no-color is active: %q", got)
	}
}
