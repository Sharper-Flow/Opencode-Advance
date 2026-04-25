package health

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func writeFakeBinary(t *testing.T, dir, name, output string) string {
	t.Helper()
	var path string
	if runtime.GOOS == "windows" {
		path = filepath.Join(dir, name+".bat")
		//nolint:gosec // test-only bat file
		if err := os.WriteFile(path, []byte("@echo "+output+"\n"), 0755); err != nil {
			t.Fatal(err)
		}
	} else {
		path = filepath.Join(dir, name)
		//nolint:gosec // test-only shell script
		if err := os.WriteFile(path, []byte("#!/bin/sh\necho '"+output+"'\n"), 0755); err != nil {
			t.Fatal(err)
		}
	}
	return path
}

func TestCheckDependencies_BothPresent(t *testing.T) {
	tmp := t.TempDir()
	writeFakeBinary(t, tmp, "node", "v22.3.0")
	writeFakeBinary(t, tmp, "temporal", "temporal version 1.2.3")
	t.Setenv("PATH", tmp+string(filepath.ListSeparator)+os.Getenv("PATH"))

	checks, err := CheckDependencies(context.Background(), nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(checks) != 2 {
		t.Fatalf("expected 2 checks, got %d: %#v", len(checks), checks)
	}

	node := checks[0]
	if node.Name != "dependencies.node" {
		t.Fatalf("expected dependencies.node, got %s", node.Name)
	}
	if node.Status != StatusPass {
		t.Fatalf("expected node pass, got %s: %s", node.Status, node.Message)
	}
	if !strings.Contains(node.Message, "v22.3.0") {
		t.Fatalf("expected version in message, got %q", node.Message)
	}

	cli := checks[1]
	if cli.Name != "dependencies.temporal_cli" {
		t.Fatalf("expected dependencies.temporal_cli, got %s", cli.Name)
	}
	if cli.Status != StatusPass {
		t.Fatalf("expected cli pass, got %s: %s", cli.Status, cli.Message)
	}
	if !strings.Contains(cli.Message, "1.2.3") {
		t.Fatalf("expected version in message, got %q", cli.Message)
	}
}

func TestCheckDependencies_NodeTooOld(t *testing.T) {
	tmp := t.TempDir()
	writeFakeBinary(t, tmp, "node", "v18.17.0")
	writeFakeBinary(t, tmp, "temporal", "temporal version 1.0.0")
	t.Setenv("PATH", tmp+string(filepath.ListSeparator)+os.Getenv("PATH"))

	checks, err := CheckDependencies(context.Background(), nil, Options{})
	if err != nil {
		t.Fatal(err)
	}

	node := checks[0]
	if node.Status != StatusWarn {
		t.Fatalf("expected node warn, got %s: %s", node.Status, node.Message)
	}
	if !strings.Contains(node.Message, "too old") {
		t.Fatalf("expected 'too old' in message, got %q", node.Message)
	}
	if !strings.Contains(node.Hint, "upgrade") {
		t.Fatalf("expected upgrade hint, got %q", node.Hint)
	}

	cli := checks[1]
	if cli.Status != StatusPass {
		t.Fatalf("expected cli pass, got %s: %s", cli.Status, cli.Message)
	}
}

func TestCheckDependencies_NodeMissing(t *testing.T) {
	tmp := t.TempDir()
	writeFakeBinary(t, tmp, "temporal", "temporal version 1.0.0")
	// Only add tmp to PATH so node is not found
	t.Setenv("PATH", tmp)

	checks, err := CheckDependencies(context.Background(), nil, Options{})
	if err != nil {
		t.Fatal(err)
	}

	node := checks[0]
	if node.Status != StatusWarn {
		t.Fatalf("expected node warn, got %s: %s", node.Status, node.Message)
	}
	if !strings.Contains(node.Hint, "install Node.js") {
		t.Fatalf("expected install hint, got %q", node.Hint)
	}

	cli := checks[1]
	if cli.Status != StatusPass {
		t.Fatalf("expected cli pass, got %s: %s", cli.Status, cli.Message)
	}
}

func TestCheckDependencies_TemporalMissing(t *testing.T) {
	tmp := t.TempDir()
	writeFakeBinary(t, tmp, "node", "v20.0.0")
	// Only add tmp to PATH so temporal is not found
	t.Setenv("PATH", tmp)

	checks, err := CheckDependencies(context.Background(), nil, Options{})
	if err != nil {
		t.Fatal(err)
	}

	node := checks[0]
	if node.Status != StatusPass {
		t.Fatalf("expected node pass, got %s: %s", node.Status, node.Message)
	}

	cli := checks[1]
	if cli.Status != StatusWarn {
		t.Fatalf("expected cli warn, got %s: %s", cli.Status, cli.Message)
	}
	if !strings.Contains(cli.Hint, "install Temporal CLI") {
		t.Fatalf("expected install hint, got %q", cli.Hint)
	}
}

func TestCheckDependencies_BothMissing(t *testing.T) {
	tmp := t.TempDir()
	// Empty PATH means neither binary is found
	t.Setenv("PATH", tmp)

	checks, err := CheckDependencies(context.Background(), nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(checks) != 2 {
		t.Fatalf("expected 2 checks, got %d", len(checks))
	}
	if checks[0].Status != StatusWarn {
		t.Fatalf("expected node warn, got %s", checks[0].Status)
	}
	if checks[1].Status != StatusWarn {
		t.Fatalf("expected cli warn, got %s", checks[1].Status)
	}
}
