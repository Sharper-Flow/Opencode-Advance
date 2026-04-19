// Package health provides configuration health checks surfaced via
// `oca doctor`. Each Check represents a single probe (e.g. Vision reachable,
// opencode.json parses) with a status, human-readable message, and optional
// remediation hint. Callers aggregate Checks and render them as text or JSON;
// HasFailures/HasWarnings/Summary helpers drive the doctor exit code.
package health

import "time"

type Status string

const (
	StatusPass Status = "pass"
	StatusWarn Status = "warn"
	StatusFail Status = "fail"
)

// Check is one doctor result row.
type Check struct {
	Name    string        `json:"name"`
	Status  Status        `json:"status"`
	Message string        `json:"message"`
	Hint    string        `json:"hint,omitempty"`
	Elapsed time.Duration `json:"elapsed"`
}
