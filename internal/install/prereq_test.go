package install

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/Sharper-Flow/Opencode-Advance/internal/health"
	"github.com/Sharper-Flow/Opencode-Advance/internal/subprocess"
)

// mockRun replaces subprocess.Run for testing.
func mockRun(outputs map[string]subprocess.Result) func(ctx context.Context, cmd subprocess.Cmd) (subprocess.Result, error) {
	return func(ctx context.Context, cmd subprocess.Cmd) (subprocess.Result, error) {
		key := cmd.Name + " " + strings.Join(cmd.Args, " ")
		if r, ok := outputs[key]; ok {
			return r, nil
		}
		return subprocess.Result{}, fmt.Errorf("executable file not found in $PATH: %s", cmd.Name)
	}
}

func TestCheckPrerequisites(t *testing.T) {
	tests := []struct {
		name      string
		outputs   map[string]subprocess.Result
		wantNames map[string]health.Status // check name → expected status
	}{
		{
			name: "all present and current",
			outputs: map[string]subprocess.Result{
				"git --version":      {ExitClass: subprocess.ExitSuccess, Output: []byte("git version 2.43.0\n")},
				"tmux -V":            {ExitClass: subprocess.ExitSuccess, Output: []byte("tmux 3.4\n")},
				"opencode --version": {ExitClass: subprocess.ExitSuccess, Output: []byte("opencode v0.5.0\n")},
				"vision --version":   {ExitClass: subprocess.ExitSuccess, Output: []byte("vision v1.2.0\n")},
				"temporal --version": {ExitClass: subprocess.ExitSuccess, Output: []byte("temporal version 1.0.0\n")},
			},
			wantNames: map[string]health.Status{
				"prerequisites.git":      health.StatusPass,
				"prerequisites.tmux":     health.StatusPass,
				"prerequisites.opencode": health.StatusPass,
				"prerequisites.vision":   health.StatusPass,
				"prerequisites.temporal": health.StatusPass,
			},
		},
		{
			name: "required git too old",
			outputs: map[string]subprocess.Result{
				"git --version":      {ExitClass: subprocess.ExitSuccess, Output: []byte("git version 1.8.3\n")},
				"tmux -V":            {ExitClass: subprocess.ExitSuccess, Output: []byte("tmux 3.4\n")},
				"opencode --version": {ExitClass: subprocess.ExitSuccess, Output: []byte("opencode v0.5.0\n")},
				"vision --version":   {ExitClass: subprocess.ExitSuccess, Output: []byte("vision v1.2.0\n")},
				"temporal --version": {ExitClass: subprocess.ExitSuccess, Output: []byte("temporal version 1.0.0\n")},
			},
			wantNames: map[string]health.Status{
				"prerequisites.git":      health.StatusFail,
				"prerequisites.tmux":     health.StatusPass,
				"prerequisites.opencode": health.StatusPass,
				"prerequisites.vision":   health.StatusPass,
				"prerequisites.temporal": health.StatusPass,
			},
		},
		{
			name: "required tmux missing",
			outputs: map[string]subprocess.Result{
				"git --version":      {ExitClass: subprocess.ExitSuccess, Output: []byte("git version 2.43.0\n")},
				"opencode --version": {ExitClass: subprocess.ExitSuccess, Output: []byte("opencode v0.5.0\n")},
				"vision --version":   {ExitClass: subprocess.ExitSuccess, Output: []byte("vision v1.2.0\n")},
				"temporal --version": {ExitClass: subprocess.ExitSuccess, Output: []byte("temporal version 1.0.0\n")},
			},
			wantNames: map[string]health.Status{
				"prerequisites.git":      health.StatusPass,
				"prerequisites.tmux":     health.StatusFail,
				"prerequisites.opencode": health.StatusPass,
				"prerequisites.vision":   health.StatusPass,
				"prerequisites.temporal": health.StatusPass,
			},
		},
		{
			name: "optional vision missing",
			outputs: map[string]subprocess.Result{
				"git --version":      {ExitClass: subprocess.ExitSuccess, Output: []byte("git version 2.43.0\n")},
				"tmux -V":            {ExitClass: subprocess.ExitSuccess, Output: []byte("tmux 3.4\n")},
				"opencode --version": {ExitClass: subprocess.ExitSuccess, Output: []byte("opencode v0.5.0\n")},
				"temporal --version": {ExitClass: subprocess.ExitSuccess, Output: []byte("temporal version 1.0.0\n")},
			},
			wantNames: map[string]health.Status{
				"prerequisites.git":      health.StatusPass,
				"prerequisites.tmux":     health.StatusPass,
				"prerequisites.opencode": health.StatusPass,
				"prerequisites.vision":   health.StatusWarn,
				"prerequisites.temporal": health.StatusPass,
			},
		},
		{
			name: "optional temporal missing",
			outputs: map[string]subprocess.Result{
				"git --version":      {ExitClass: subprocess.ExitSuccess, Output: []byte("git version 2.43.0\n")},
				"tmux -V":            {ExitClass: subprocess.ExitSuccess, Output: []byte("tmux 3.4\n")},
				"opencode --version": {ExitClass: subprocess.ExitSuccess, Output: []byte("opencode v0.5.0\n")},
				"vision --version":   {ExitClass: subprocess.ExitSuccess, Output: []byte("vision v1.2.0\n")},
			},
			wantNames: map[string]health.Status{
				"prerequisites.git":      health.StatusPass,
				"prerequisites.tmux":     health.StatusPass,
				"prerequisites.opencode": health.StatusPass,
				"prerequisites.vision":   health.StatusPass,
				"prerequisites.temporal": health.StatusWarn,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orig := runDetect
			defer func() { runDetect = orig }()
			runDetect = mockRun(tt.outputs)

			checks, err := CheckPrerequisites(context.Background(), nil, health.Options{})
			if err != nil {
				t.Fatal(err)
			}

			if len(checks) != len(tt.wantNames) {
				t.Errorf("got %d checks, want %d", len(checks), len(tt.wantNames))
			}

			for _, c := range checks {
				want, ok := tt.wantNames[c.Name]
				if !ok {
					t.Errorf("unexpected check %q", c.Name)
					continue
				}
				if c.Status != want {
					t.Errorf("check %q: got %s, want %s (message: %s)", c.Name, c.Status, want, c.Message)
				}
			}
		})
	}
}

func TestParseMajorVersion(t *testing.T) {
	tests := []struct {
		input   string
		want    int
		wantErr bool
	}{
		{"git version 2.43.0", 2, false},
		{"git version 1.8.3.1", 1, false},
		{"tmux 3.4", 3, false},
		{"tmux 2.8", 2, false},
		{"v0.5.0", 0, false},
		{"1.2.3", 1, false},
		{"", 0, true},
		{"no-version-here", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseMajorVersion(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseMajorVersion(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("parseMajorVersion(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}
