package tests

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
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
