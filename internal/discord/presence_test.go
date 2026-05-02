package discord

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

type fakeRPCClient struct {
	loginAppID string
	activity   Activity
	setCalls   int
	logout     bool
}

func (f *fakeRPCClient) Login(appID string) error {
	f.loginAppID = appID
	return nil
}

func (f *fakeRPCClient) SetActivity(activity Activity) error {
	f.activity = activity
	f.setCalls++
	return nil
}

func (f *fakeRPCClient) Logout() error {
	f.logout = true
	return nil
}

func TestPresenceManagerUpdateSetsDiscordActivity(t *testing.T) {
	tmp := t.TempDir()
	taglinePath := filepath.Join(tmp, "taglines.toml")
	if err := os.WriteFile(taglinePath, []byte("taglines = [\"Workflow state synchronized\"]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	started := time.Date(2026, 5, 2, 4, 0, 0, 0, time.UTC)
	client := &fakeRPCClient{}
	manager := PresenceManager{
		Client:        client,
		AppID:         "1476685752363516135",
		Details:       "OpenCode Advance",
		StartedAt:     started,
		TaglinePath:   taglinePath,
		LastIndexPath: filepath.Join(tmp, "last-index"),
		RateLimitPath: filepath.Join(tmp, "last-update"),
		RateLimit:     15 * time.Second,
		Clock:         func() time.Time { return started.Add(time.Minute) },
	}

	status, err := manager.Update()
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if status.Skipped {
		t.Fatal("first update should not be skipped")
	}
	if client.loginAppID != "1476685752363516135" {
		t.Fatalf("login app ID = %q", client.loginAppID)
	}
	if client.setCalls != 1 {
		t.Fatalf("set calls = %d, want 1", client.setCalls)
	}
	if client.activity.State != "Workflow state synchronized" {
		t.Fatalf("activity state = %q", client.activity.State)
	}
	if client.activity.Details != "OpenCode Advance" {
		t.Fatalf("activity details = %q", client.activity.Details)
	}
	if !client.activity.Start.Equal(started) {
		t.Fatalf("activity start = %v, want %v", client.activity.Start, started)
	}
}

func TestPresenceManagerRateLimitSkipsRecentUpdate(t *testing.T) {
	tmp := t.TempDir()
	now := time.Date(2026, 5, 2, 4, 0, 0, 0, time.UTC)
	rateLimitPath := filepath.Join(tmp, "last-update")
	if err := os.WriteFile(rateLimitPath, []byte(now.Add(-5*time.Second).Format(time.RFC3339Nano)), 0o644); err != nil {
		t.Fatal(err)
	}
	client := &fakeRPCClient{}
	manager := PresenceManager{
		Client:        client,
		AppID:         "1476685752363516135",
		Details:       "OpenCode Advance",
		StartedAt:     now,
		TaglinePath:   filepath.Join(tmp, "missing.toml"),
		LastIndexPath: filepath.Join(tmp, "last-index"),
		RateLimitPath: rateLimitPath,
		RateLimit:     15 * time.Second,
		Clock:         func() time.Time { return now },
	}

	status, err := manager.Update()
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if !status.Skipped {
		t.Fatal("recent update should be skipped")
	}
	if client.setCalls != 0 {
		t.Fatalf("set calls = %d, want 0", client.setCalls)
	}
}

func TestPresenceManagerDisconnect(t *testing.T) {
	client := &fakeRPCClient{}
	manager := PresenceManager{Client: client}
	if err := manager.Disconnect(); err != nil {
		t.Fatalf("Disconnect returned error: %v", err)
	}
	if !client.logout {
		t.Fatal("Disconnect should call Logout")
	}
}
