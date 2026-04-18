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
