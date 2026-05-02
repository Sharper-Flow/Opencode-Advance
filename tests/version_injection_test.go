package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMakefileBuildTargetSupportsVersionInjection(t *testing.T) {
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	makefile := string(data)
	for _, want := range []string{
		"VERSION ?= 0.0.0-dev",
		"LDFLAGS ?= -X main.version=$(VERSION)",
		"go build -ldflags \"$(LDFLAGS)\" -o bin/oca ./cmd/oca",
	} {
		if !strings.Contains(makefile, want) {
			t.Fatalf("Makefile missing %q\n%s", want, makefile)
		}
	}
}
