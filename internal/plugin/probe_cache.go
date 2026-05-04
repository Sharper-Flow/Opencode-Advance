package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type DriftCacheStatus string

const (
	DriftCacheFresh   DriftCacheStatus = "fresh"
	DriftCacheMissing DriftCacheStatus = "missing"
	DriftCacheStale   DriftCacheStatus = "stale"
	DriftCacheCorrupt DriftCacheStatus = "corrupt"
)

type DriftCache struct {
	GeneratedAt time.Time     `json:"generated_at"`
	Results     []DriftResult `json:"results"`
}

func WriteDriftCache(path string, results []DriftResult) error {
	cache := DriftCache{GeneratedAt: time.Now().UTC(), Results: results}
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal drift cache: %w", err)
	}
	data = append(data, '\n')
	if err := writeCacheAtomic(path, data, 0o644); err != nil {
		return fmt.Errorf("write drift cache %s: %w", path, err)
	}
	return nil
}

func writeCacheAtomic(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".oca-drift-cache-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpPath, mode); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func ReadDriftCache(path string, ttl time.Duration) (DriftCacheStatus, DriftCache, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DriftCacheMissing, DriftCache{}, nil
		}
		return DriftCacheCorrupt, DriftCache{}, nil
	}
	var cache DriftCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return DriftCacheCorrupt, DriftCache{}, nil
	}
	if cache.GeneratedAt.IsZero() {
		return DriftCacheCorrupt, DriftCache{}, nil
	}
	if ttl < 0 || (ttl > 0 && time.Since(cache.GeneratedAt) > ttl) {
		return DriftCacheStale, cache, nil
	}
	return DriftCacheFresh, cache, nil
}
