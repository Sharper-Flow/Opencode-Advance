package discord

import (
	"errors"
	"os"
	"path/filepath"
	"time"

	rich "github.com/hugolgst/rich-go/client"
)

const DefaultDetails = "OpenCode Advance"

// Activity is the OCA-local presence payload. RichGoClient adapts it to the
// concrete rich-go type so tests can use a fake client without Discord IPC.
type Activity struct {
	Details string
	State   string
	Start   time.Time
}

// RPCClient is the minimal Discord Rich Presence surface OCA needs.
type RPCClient interface {
	Login(appID string) error
	SetActivity(activity Activity) error
	Logout() error
}

// RichGoClient adapts github.com/hugolgst/rich-go/client to RPCClient.
type RichGoClient struct{}

func (RichGoClient) Login(appID string) error { return rich.Login(appID) }

func (RichGoClient) SetActivity(activity Activity) error {
	return rich.SetActivity(rich.Activity{
		Details: activity.Details,
		State:   activity.State,
		Timestamps: &rich.Timestamps{
			Start: &activity.Start,
		},
	})
}

func (RichGoClient) Logout() error {
	rich.Logout()
	return nil
}

// PresenceManager coordinates taglines, rate limiting, and Discord IPC.
type PresenceManager struct {
	Client        RPCClient
	AppID         string
	Details       string
	StartedAt     time.Time
	TaglinePath   string
	LastIndexPath string
	RateLimitPath string
	RateLimit     time.Duration
	Clock         func() time.Time
}

type PresenceStatus struct {
	Skipped bool
	Tagline string
	Updated time.Time
	Warning error
}

func (m PresenceManager) Update() (PresenceStatus, error) {
	now := m.now()
	if m.rateLimited(now) {
		return PresenceStatus{Skipped: true, Updated: now}, nil
	}

	taglines, warning := LoadTaglines(m.TaglinePath)
	tagline, err := SelectTagline(taglines, m.LastIndexPath)
	if err != nil {
		return PresenceStatus{Tagline: tagline, Updated: now, Warning: warning}, err
	}

	client := m.client()
	if err := client.Login(m.AppID); err != nil {
		return PresenceStatus{Tagline: tagline, Updated: now, Warning: warning}, err
	}

	activity := Activity{
		Details: m.details(),
		State:   tagline,
		Start:   m.startTime(now),
	}
	if err := client.SetActivity(activity); err != nil {
		return PresenceStatus{Tagline: tagline, Updated: now, Warning: warning}, err
	}
	if err := writeTimestamp(m.RateLimitPath, now); err != nil {
		return PresenceStatus{Tagline: tagline, Updated: now, Warning: warning}, err
	}
	return PresenceStatus{Tagline: tagline, Updated: now, Warning: warning}, nil
}

func (m PresenceManager) Disconnect() error {
	return m.client().Logout()
}

func (m PresenceManager) client() RPCClient {
	if m.Client != nil {
		return m.Client
	}
	return RichGoClient{}
}

func (m PresenceManager) now() time.Time {
	if m.Clock != nil {
		return m.Clock()
	}
	return time.Now()
}

func (m PresenceManager) details() string {
	if m.Details != "" {
		return m.Details
	}
	return DefaultDetails
}

func (m PresenceManager) startTime(fallback time.Time) time.Time {
	if !m.StartedAt.IsZero() {
		return m.StartedAt
	}
	return fallback
}

func (m PresenceManager) rateLimit() time.Duration {
	if m.RateLimit > 0 {
		return m.RateLimit
	}
	return 15 * time.Second
}

func (m PresenceManager) rateLimited(now time.Time) bool {
	last, err := readTimestamp(m.RateLimitPath)
	if err != nil {
		return false
	}
	return now.Sub(last) < m.rateLimit()
}

func readTimestamp(path string) (time.Time, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return time.Time{}, err
	}
	return time.Parse(time.RFC3339Nano, string(data))
}

func writeTimestamp(path string, ts time.Time) error {
	if path == "" {
		return errors.New("rate limit path is required")
	}
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
	}
	return os.WriteFile(path, []byte(ts.Format(time.RFC3339Nano)), 0o600)
}
