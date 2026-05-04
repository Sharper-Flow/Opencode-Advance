package tests

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

var (
	ocaBuildOnce sync.Once
	ocaBuildPath string
	ocaBuildErr  error
)

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Dir(wd)
}

func runOCA(t *testing.T, root string, env []string, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	bin := ocaBinary(t, root)
	cmd := exec.Command(bin, args...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), env...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err = cmd.Run()
	return outBuf.String(), errBuf.String(), err
}

func ocaBinary(t *testing.T, root string) string {
	t.Helper()
	ocaBuildOnce.Do(func() {
		tmpDir, err := os.MkdirTemp("", "oca-test-bin-")
		if err != nil {
			ocaBuildErr = err
			return
		}
		ocaBuildPath = filepath.Join(tmpDir, "oca")
		cmd := exec.Command("go", "build", "-o", ocaBuildPath, "./cmd/oca")
		cmd.Dir = root
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		if err := cmd.Run(); err != nil {
			ocaBuildErr = err
			if out.Len() > 0 {
				ocaBuildErr = exec.ErrNotFound
			}
		}
	})
	if ocaBuildErr != nil {
		t.Fatalf("build oca test binary: %v", ocaBuildErr)
	}
	return ocaBuildPath
}

func writeIsolatedStackExample(t *testing.T, root string, base string) string {
	t.Helper()
	advanceRemote := initPluginRemote(t, filepath.Join(base, "advance-src"), "advance")
	morphRemote := initPluginRemote(t, filepath.Join(base, "morph-src"), "root")
	visionRemote := initPluginRemote(t, filepath.Join(base, "vision-src"), "vision-subdir")

	fixtureBytes, err := os.ReadFile(filepath.Join(root, "stack.example.toml"))
	if err != nil {
		t.Fatal(err)
	}
	fixture := string(fixtureBytes)
	fixture = strings.ReplaceAll(fixture, "https://github.com/Sharper-Flow/Advance.git", advanceRemote)
	fixture = strings.ReplaceAll(fixture, "~/dev/oc-plugins/advance", filepath.Join(base, "checkouts", "advance"))
	fixture = strings.ReplaceAll(fixture, "build        = [\"pnpm install\", \"pnpm build\"]", "build        = []")
	fixture = strings.ReplaceAll(fixture, "build    = [\"pnpm install\", \"pnpm build\"]", "build    = []")
	fixture = strings.ReplaceAll(fixture, "https://github.com/JRedeker/opencode-morph-fast-apply.git", morphRemote)
	fixture = strings.ReplaceAll(fixture, "~/dev/oc-plugins/morph-fast-apply", filepath.Join(base, "checkouts", "morph-fast-apply"))
	fixture = strings.ReplaceAll(fixture, "https://github.com/Sharper-Flow/vision.git", visionRemote)
	fixture = strings.ReplaceAll(fixture, "~/dev/vision", filepath.Join(base, "checkouts", "vision"))
	fixture = strings.ReplaceAll(fixture, `ref          = "trunk"`, `ref          = "master"`)
	fixture = strings.ReplaceAll(fixture, `ref      = "trunk"`, `ref      = "master"`)
	fixture = strings.ReplaceAll(fixture, `local:~/dev/opencodeadvance/plugins/oca`, `local:`+filepath.Join(root, "plugins", "oca"))
	fixture = strings.ReplaceAll(fixture, `~/dev/opencodeadvance/plugins/oca`, filepath.Join(root, "plugins", "oca"))

	stackPath := filepath.Join(base, "stack.example.toml")
	if err := os.WriteFile(stackPath, []byte(fixture), 0o644); err != nil {
		t.Fatal(err)
	}
	return stackPath
}
