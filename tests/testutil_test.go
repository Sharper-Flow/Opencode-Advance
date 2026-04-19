package tests

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
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
	cmd := exec.Command("go", append([]string{"run", "./cmd/oca"}, args...)...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), env...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err = cmd.Run()
	return outBuf.String(), errBuf.String(), err
}
