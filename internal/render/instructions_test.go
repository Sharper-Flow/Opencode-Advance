package render

import (
	"reflect"
	"testing"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

func TestRenderInstructionsList_BaseOrderPlusPluginInstructions(t *testing.T) {
	stack := &cfg.Stack{
		Instructions: cfg.InstructionsSection{Order: []string{"identity.md", "rules.yaml"}},
		Plugins: cfg.PluginsSection{
			"advance": {Instructions: []string{"/plugins/advance/ADV_INSTRUCTIONS.md"}},
			"morph":   {Instructions: []string{"/plugins/morph/morph-tools.md"}},
		},
	}

	got := RenderInstructionsList(stack)
	want := []string{"identity.md", "rules.yaml", "/plugins/advance/ADV_INSTRUCTIONS.md", "/plugins/morph/morph-tools.md"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
}

func TestRenderInstructionsList_FiltersAdvInstructionsWhenProvided(t *testing.T) {
	stack := &cfg.Stack{
		Instructions: cfg.InstructionsSection{Order: []string{"identity.md"}},
		Plugins: cfg.PluginsSection{
			"advance": {
				Provides:     []cfg.ProvidesCategory{cfg.ProvidesInstructions},
				Instructions: []string{"/plugins/advance/ADV_INSTRUCTIONS.md"},
			},
			"morph": {Instructions: []string{"/plugins/morph/morph-tools.md"}},
		},
	}

	got := RenderInstructionsList(stack)
	want := []string{"identity.md", "/plugins/morph/morph-tools.md"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
}

func TestRenderInstructionsList_SkipsDisabledPluginInstructions(t *testing.T) {
	disabled := false
	stack := &cfg.Stack{
		Instructions: cfg.InstructionsSection{Order: []string{"identity.md"}},
		Plugins: cfg.PluginsSection{
			"advance": {
				Enabled:      &disabled,
				Instructions: []string{"/plugins/advance/ADV_INSTRUCTIONS.md"},
			},
		},
	}

	got := RenderInstructionsList(stack)
	want := []string{"identity.md"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
}

func TestRenderInstructionsList_DeterministicPluginOrder(t *testing.T) {
	stack := &cfg.Stack{
		Instructions: cfg.InstructionsSection{Order: []string{"identity.md"}},
		Plugins: cfg.PluginsSection{
			"zeta":  {Instructions: []string{"/plugins/zeta/z.md"}},
			"alpha": {Instructions: []string{"/plugins/alpha/a.md"}},
		},
	}

	got := RenderInstructionsList(stack)
	want := []string{"identity.md", "/plugins/alpha/a.md", "/plugins/zeta/z.md"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
}
