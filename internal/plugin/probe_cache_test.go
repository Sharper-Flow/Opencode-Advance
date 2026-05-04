package plugin

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

func TestDriftCachePathHonorsCacheOverride(t *testing.T) {
	paths := config.Paths{CacheDir: filepath.Join(t.TempDir(), "cache")}
	if got, want := paths.DriftCachePath(), filepath.Join(paths.CacheDir, "drift_cache.json"); got != want {
		t.Fatalf("DriftCachePath() = %q, want %q", got, want)
	}
}

func TestWriteAndReadFreshDriftCache(t *testing.T) {
	path := filepath.Join(t.TempDir(), "drift_cache.json")
	want := []DriftResult{{Name: "advance", Ref: "trunk", Status: DriftUpdateAvailable, LocalSHA: "a", RemoteSHA: "b"}}
	if err := WriteDriftCache(path, want); err != nil {
		t.Fatalf("WriteDriftCache() = %v", err)
	}

	status, cache, err := ReadDriftCache(path, time.Hour)
	if err != nil {
		t.Fatalf("ReadDriftCache() err = %v", err)
	}
	if status != DriftCacheFresh {
		t.Fatalf("status = %s, want %s", status, DriftCacheFresh)
	}
	if len(cache.Results) != 1 || cache.Results[0].Name != "advance" {
		t.Fatalf("cache results = %+v", cache.Results)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o644 {
		t.Fatalf("cache mode = %o, want 644", got)
	}
}

func TestReadDriftCacheMissingStaleAndCorrupt(t *testing.T) {
	dir := t.TempDir()

	missing, _, err := ReadDriftCache(filepath.Join(dir, "missing.json"), time.Hour)
	if err != nil || missing != DriftCacheMissing {
		t.Fatalf("missing status=%s err=%v", missing, err)
	}

	stalePath := filepath.Join(dir, "stale.json")
	if err := WriteDriftCache(stalePath, []DriftResult{{Name: "advance", Status: DriftUpToDate}}); err != nil {
		t.Fatal(err)
	}
	stale, _, err := ReadDriftCache(stalePath, -time.Second)
	if err != nil || stale != DriftCacheStale {
		t.Fatalf("stale status=%s err=%v", stale, err)
	}

	corruptPath := filepath.Join(dir, "corrupt.json")
	if err := os.WriteFile(corruptPath, []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	corrupt, _, err := ReadDriftCache(corruptPath, time.Hour)
	if err != nil || corrupt != DriftCacheCorrupt {
		t.Fatalf("corrupt status=%s err=%v", corrupt, err)
	}
}
