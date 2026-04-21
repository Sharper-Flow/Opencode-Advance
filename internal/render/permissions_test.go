package render

import (
	"testing"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

func TestTranslatePermissionsToOpencode_EmptyReturnsNil(t *testing.T) {
	got := translatePermissionsToOpencode(cfg.PermissionsSection{})
	if got != nil {
		t.Fatalf("expected nil for empty permissions section, got %#v", got)
	}
}
