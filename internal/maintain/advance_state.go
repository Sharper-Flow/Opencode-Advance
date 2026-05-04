package maintain

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
	"github.com/Sharper-Flow/Opencode-Advance/internal/subprocess"
)

type AdvanceInspectReport struct {
	SchemaVersion    int                        `json:"schema_version"`
	ProjectRoot      string                     `json:"project_root"`
	EligibleArchives []AdvanceArchiveCandidate  `json:"eligible_archives"`
	ArchivedChanges  []AdvanceArchiveCandidate  `json:"archived_changes"`
	Verification     AdvanceVerificationSummary `json:"verification_summary"`
}

type AdvanceVerificationSummary struct {
	ArchivedCount   int `json:"archived_count"`
	EligibleCount   int `json:"eligible_count"`
	IneligibleCount int `json:"ineligible_count"`
}

type AdvanceArchiveCandidate struct {
	ChangeID    string `json:"change_id"`
	Branch      string `json:"branch,omitempty"`
	Status      string `json:"status,omitempty"`
	ReleaseGate string `json:"release_gate"`
	Eligible    bool   `json:"eligible"`
	Reason      string `json:"reason,omitempty"`
}

func (r AdvanceInspectReport) Candidates() []AdvanceArchiveCandidate {
	if len(r.ArchivedChanges) > 0 {
		return r.ArchivedChanges
	}
	return r.EligibleArchives
}

func (c AdvanceArchiveCandidate) IneligibleReason() string {
	if c.Reason != "" {
		return c.Reason
	}
	if c.ReleaseGate != "done" {
		return "release gate is " + c.ReleaseGate
	}
	if !c.Eligible {
		return "not marked eligible by Advance inspect"
	}
	return "not eligible"
}

func InspectAdvancePlugin(ctx context.Context, plugin cfg.Plugin) (AdvanceInspectReport, error) {
	if plugin.Checkout == "" {
		return AdvanceInspectReport{}, nil
	}
	script := filepath.Join(plugin.Checkout, "scripts", "maintenance", "inspect.mjs")
	res, err := subprocess.Run(ctx, subprocess.Cmd{
		Name:    "node",
		Args:    []string{script, "--project-root", plugin.Checkout},
		Dir:     plugin.Checkout,
		Timeout: 30 * time.Second,
	})
	if err != nil {
		return AdvanceInspectReport{}, fmt.Errorf("advance inspect failed: %w\noutput:\n%s", err, strings.TrimSpace(string(res.Output)))
	}
	var report AdvanceInspectReport
	if err := json.Unmarshal(res.Output, &report); err != nil {
		return AdvanceInspectReport{}, fmt.Errorf("decode advance inspect: %w", err)
	}
	if report.SchemaVersion != SchemaVersion {
		return AdvanceInspectReport{}, fmt.Errorf("unsupported advance inspect schema_version %d", report.SchemaVersion)
	}
	return report, nil
}
