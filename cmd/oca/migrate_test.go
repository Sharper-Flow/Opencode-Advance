package main

import (
	"strings"
	"testing"
)

func TestMigrateInitCmd(t *testing.T) {
	opts := defaultCommandOptions()
	root := newRootCmd(opts)
	root.SetArgs([]string{"migrate", "init"})

	out := &strings.Builder{}
	root.SetOut(out)
	root.SetErr(out)

	if err := root.Execute(); err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "[meta]") {
		t.Error("missing meta section")
	}
	if !strings.Contains(output, "[mcp.servers.vision]") {
		t.Error("missing vision server")
	}
	if !strings.Contains(output, "[plugins.advance]") {
		t.Error("missing advance plugin")
	}
}

func TestMigrateFromOpenChad_DryRun(t *testing.T) {
	opts := defaultCommandOptions()
	root := newRootCmd(opts)
	root.SetArgs([]string{"migrate", "from-open-chad", "--dry-run"})

	out := &strings.Builder{}
	root.SetOut(out)
	root.SetErr(out)

	if err := root.Execute(); err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Dry run") {
		t.Error("missing dry-run header")
	}
	if !strings.Contains(output, "open-chad repo") {
		t.Error("missing open-chad repo path")
	}
}
