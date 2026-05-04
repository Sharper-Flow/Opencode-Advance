package maintain

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
	"github.com/Sharper-Flow/Opencode-Advance/internal/subprocess"
)

type DriftStatus string

const (
	DriftStatusFresh         DriftStatus = "fresh"
	DriftStatusMissingMarker DriftStatus = "missing_marker"
	DriftStatusStale         DriftStatus = "stale"
	DriftStatusUnknown       DriftStatus = "unknown"
)

type BuildMarker struct {
	SchemaVersion    int       `json:"schema_version"`
	Plugin           string    `json:"plugin"`
	SourceRoot       string    `json:"source_root"`
	GitSHA           string    `json:"git_sha"`
	Branch           string    `json:"branch"`
	BuiltAt          time.Time `json:"built_at"`
	DistIndex        string    `json:"dist_index"`
	WorkerBundle     string    `json:"worker_bundle,omitempty"`
	BuildCommandHash string    `json:"build_command_hash"`
}

type PluginDriftReport struct {
	Plugin     string       `json:"plugin"`
	Status     DriftStatus  `json:"status"`
	SourceRoot string       `json:"source_root"`
	MarkerPath string       `json:"marker_path"`
	GitSHA     string       `json:"git_sha,omitempty"`
	Marker     *BuildMarker `json:"marker,omitempty"`
	Reason     string       `json:"reason,omitempty"`
}

func DetectPluginDrift(ctx context.Context, name string, plugin cfg.Plugin) (PluginDriftReport, error) {
	root := pluginRuntimeRoot(plugin)
	markerPath := BuildMarkerPath(plugin)
	report := PluginDriftReport{Plugin: name, SourceRoot: sourceRoot(plugin), MarkerPath: markerPath}
	head, err := gitOutput(ctx, sourceRoot(plugin), "rev-parse", "HEAD")
	if err != nil {
		report.Status = DriftStatusUnknown
		report.Reason = err.Error()
		return report, nil
	}
	report.GitSHA = head
	marker, err := ReadBuildMarker(markerPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			report.Status = DriftStatusMissingMarker
			report.Reason = "build marker missing"
			return report, nil
		}
		report.Status = DriftStatusUnknown
		report.Reason = err.Error()
		return report, nil
	}
	report.Marker = &marker
	if marker.GitSHA != head {
		report.Status = DriftStatusStale
		report.Reason = fmt.Sprintf("marker git_sha %s does not match HEAD %s", marker.GitSHA, head)
		return report, nil
	}
	if marker.BuildCommandHash != buildCommandHash(plugin.Build) {
		report.Status = DriftStatusStale
		report.Reason = "build command hash changed"
		return report, nil
	}
	if _, err := os.Stat(filepath.Join(root, marker.DistIndex)); err != nil {
		report.Status = DriftStatusStale
		report.Reason = fmt.Sprintf("dist index missing: %v", err)
		return report, nil
	}
	report.Status = DriftStatusFresh
	return report, nil
}

func WriteBuildMarker(ctx context.Context, name string, plugin cfg.Plugin) (BuildMarker, error) {
	root := pluginRuntimeRoot(plugin)
	head, err := gitOutput(ctx, sourceRoot(plugin), "rev-parse", "HEAD")
	if err != nil {
		return BuildMarker{}, err
	}
	branch, _ := gitOutput(ctx, sourceRoot(plugin), "branch", "--show-current")
	marker := BuildMarker{
		SchemaVersion:    SchemaVersion,
		Plugin:           name,
		SourceRoot:       sourceRoot(plugin),
		GitSHA:           head,
		Branch:           branch,
		BuiltAt:          time.Now().UTC(),
		DistIndex:        filepath.ToSlash(relativeOrBase(root, distIndexPath(plugin))),
		WorkerBundle:     filepath.ToSlash(relativeOrEmpty(root, filepath.Join(root, "dist", "temporal", "worker.js"))),
		BuildCommandHash: buildCommandHash(plugin.Build),
	}
	markerPath := BuildMarkerPath(plugin)
	if err := os.MkdirAll(filepath.Dir(markerPath), 0o755); err != nil {
		return BuildMarker{}, err
	}
	data, err := json.MarshalIndent(marker, "", "  ")
	if err != nil {
		return BuildMarker{}, err
	}
	if err := os.WriteFile(markerPath, append(data, '\n'), 0o644); err != nil {
		return BuildMarker{}, err
	}
	return marker, nil
}

func ReadBuildMarker(path string) (BuildMarker, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return BuildMarker{}, err
	}
	var marker BuildMarker
	if err := json.Unmarshal(data, &marker); err != nil {
		return BuildMarker{}, err
	}
	return marker, nil
}

func BuildMarkerPath(plugin cfg.Plugin) string {
	return filepath.Join(pluginRuntimeRoot(plugin), "dist", "oca-build.json")
}

func pluginRuntimeRoot(plugin cfg.Plugin) string {
	if plugin.Checkout != "" && plugin.Subdir != "" {
		return filepath.Join(plugin.Checkout, plugin.Subdir)
	}
	if plugin.Checkout != "" {
		return plugin.Checkout
	}
	if plugin.Path != "" {
		if strings.HasSuffix(plugin.Path, ".js") {
			return filepath.Dir(filepath.Dir(plugin.Path))
		}
		return plugin.Path
	}
	return "."
}

func sourceRoot(plugin cfg.Plugin) string {
	if plugin.Checkout != "" {
		return plugin.Checkout
	}
	return pluginRuntimeRoot(plugin)
}

func distIndexPath(plugin cfg.Plugin) string {
	if plugin.Path != "" && strings.HasSuffix(plugin.Path, ".js") {
		return plugin.Path
	}
	return filepath.Join(pluginRuntimeRoot(plugin), "dist", "index.js")
}

func buildCommandHash(commands []string) string {
	h := sha256.New()
	for _, cmd := range commands {
		_, _ = h.Write([]byte(cmd))
		_, _ = h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func gitOutput(ctx context.Context, dir string, args ...string) (string, error) {
	res, err := subprocess.Run(ctx, subprocess.Cmd{Name: "git", Args: args, Dir: dir, Timeout: 10 * time.Second})
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(res.Output)), nil
}

func relativeOrBase(root, path string) string {
	if rel := relativeOrEmpty(root, path); rel != "" {
		return rel
	}
	return filepath.Base(path)
}

func relativeOrEmpty(root, path string) string {
	if _, err := os.Stat(path); err != nil {
		return ""
	}
	if rel, err := filepath.Rel(root, path); err == nil && !strings.HasPrefix(rel, "..") {
		return rel
	}
	return ""
}
