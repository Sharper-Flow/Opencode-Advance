package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Sharper-Flow/Opencode-Advance/brand"
	"github.com/Sharper-Flow/Opencode-Advance/internal/maintain"
)

func TestMaintainCommand_DryRunReportsPlanWithoutMutation(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("OCA_OPENCODE_CONFIG_DIR", filepath.Join(tmp, "opencode"))
	t.Setenv("OCA_VISION_CONFIG_DIR", filepath.Join(tmp, "vision"))
	t.Setenv("OCA_CACHE_DIR", filepath.Join(tmp, "cache"))

	stackPath := filepath.Join(tmp, "stack.toml")
	writeFile(t, stackPath, `[meta]
version = "1.0.0"
`)

	original := maintainPlanFunc
	maintainPlanFunc = func(ctx context.Context, opts maintain.Options) (maintain.Plan, error) {
		if !opts.DryRun {
			t.Fatalf("dry-run command passed DryRun=false")
		}
		return maintain.Plan{
			SchemaVersion: 1,
			GeneratedAt:   time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC),
			ProjectRoot:   opts.ProjectRoot,
			DefaultBranch: "trunk",
			SessionGate: maintain.GateReport{
				Status: maintain.GateStatusPass,
			},
			Actions: []maintain.Action{{ID: "rebuild:advance", Kind: "rebuild", WouldRun: true, ExecuteAllowed: true}},
		}, nil
	}
	t.Cleanup(func() { maintainPlanFunc = original })

	var stdout, stderr bytes.Buffer
	cmd := newRootCmd(commandOptions{Stdout: &stdout, Stderr: &stderr, Environment: brand.Environment{IsTTY: false}})
	cmd.SetArgs([]string{"--output", "json", "maintain", "--dry-run", "--config", stackPath})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("maintain dry-run: %v stderr=%s", err, stderr.String())
	}

	var got map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("decode output: %v\n%s", err, stdout.String())
	}
	if got["dry_run"] != true {
		t.Fatalf("dry_run=%#v, want true; output=%s", got["dry_run"], stdout.String())
	}
	if got["execute"] != false {
		t.Fatalf("execute=%#v, want false; output=%s", got["execute"], stdout.String())
	}
	if _, ok := got["plan"].(map[string]any); !ok {
		t.Fatalf("plan missing or wrong type: %#v", got["plan"])
	}
}

func TestMaintainCommand_ExecuteRefusesActiveSessionBlockers(t *testing.T) {
	tmp := t.TempDir()
	stackPath := filepath.Join(tmp, "stack.toml")
	writeFile(t, stackPath, `[meta]
version = "1.0.0"
`)

	original := maintainPlanFunc
	maintainPlanFunc = func(ctx context.Context, opts maintain.Options) (maintain.Plan, error) {
		if !opts.Execute {
			t.Fatalf("execute command passed Execute=false")
		}
		return maintain.Plan{
			SchemaVersion: 1,
			SessionGate: maintain.GateReport{
				Status:   maintain.GateStatusBlocked,
				Blockers: []maintain.Blocker{{Code: "ACTIVE_OPENCODE_PROCESS", Message: "opencode pid 123 still running"}},
			},
			Blockers: []maintain.Blocker{{Code: "ACTIVE_OPENCODE_PROCESS", Message: "opencode pid 123 still running"}},
		}, nil
	}
	t.Cleanup(func() { maintainPlanFunc = original })

	var stdout, stderr bytes.Buffer
	cmd := newRootCmd(commandOptions{Stdout: &stdout, Stderr: &stderr, Environment: brand.Environment{IsTTY: false}})
	cmd.SetArgs([]string{"maintain", "--execute", "--config", stackPath})
	err := cmd.Execute()
	if err == nil {
		t.Fatalf("maintain --execute succeeded; stdout=%s", stdout.String())
	}
	var ec exitCoder
	if !errors.As(err, &ec) || ec.ExitCode() != 3 {
		t.Fatalf("error=%v, want exit code 3", err)
	}
	if !strings.Contains(err.Error(), "active maintenance blockers") {
		t.Fatalf("error=%q missing blocker summary", err.Error())
	}
	if !strings.Contains(stdout.String(), "ACTIVE_OPENCODE_PROCESS") {
		t.Fatalf("stdout=%q missing blocker code", stdout.String())
	}
}

func TestMaintainCommand_ExecuteRunsAllowedActions(t *testing.T) {
	tmp := t.TempDir()
	stackPath := filepath.Join(tmp, "stack.toml")
	writeFile(t, stackPath, `[meta]
version = "1.0.0"
`)

	originalPlan := maintainPlanFunc
	maintainPlanFunc = func(ctx context.Context, opts maintain.Options) (maintain.Plan, error) {
		return maintain.Plan{
			SchemaVersion: 1,
			SessionGate:   maintain.GateReport{Status: maintain.GateStatusPass},
			Actions: []maintain.Action{{
				ID:             "merge:verifiedChange",
				Kind:           "merge",
				WouldRun:       true,
				ExecuteAllowed: true,
			}},
		}, nil
	}
	t.Cleanup(func() { maintainPlanFunc = originalPlan })

	called := false
	originalExecute := maintainExecuteFunc
	maintainExecuteFunc = func(ctx context.Context, opts maintain.Options, plan maintain.Plan) error {
		called = true
		if !opts.Execute {
			t.Fatalf("execute function saw Execute=false")
		}
		if len(plan.Actions) != 1 || plan.Actions[0].ID != "merge:verifiedChange" {
			t.Fatalf("plan actions=%#v", plan.Actions)
		}
		return nil
	}
	t.Cleanup(func() { maintainExecuteFunc = originalExecute })

	var stdout, stderr bytes.Buffer
	cmd := newRootCmd(commandOptions{Stdout: &stdout, Stderr: &stderr, Environment: brand.Environment{IsTTY: false}})
	cmd.SetArgs([]string{"maintain", "--execute", "--config", stackPath})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("maintain --execute: %v stderr=%s", err, stderr.String())
	}
	if !called {
		t.Fatal("maintainExecuteFunc was not called")
	}
}
