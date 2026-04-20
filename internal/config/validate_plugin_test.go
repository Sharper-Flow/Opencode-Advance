package config

import (
	"testing"
)

func TestValidatePlugins_GitSourceRequiresCheckoutAndPath(t *testing.T) {
	// Missing checkout on git source → error.
	stack := minimalStack()
	stack.Plugins = PluginsSection{
		"advance": Plugin{
			Source: "https://github.com/Sharper-Flow/Advance.git",
			Ref:    "trunk",
			// checkout missing
			Path: "/dev/advance/plugin",
		},
	}
	errs := validatePlugins(stack)
	if len(errs) == 0 {
		t.Fatal("expected error for missing checkout, got none")
	}
	found := false
	for _, e := range errs {
		if e.Path == "plugins.advance.checkout" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected checkout error, got: %v", errs)
	}
}

func TestValidatePlugins_NPMSourceRequiresOnlySource(t *testing.T) {
	// npm source should not require checkout/path.
	stack := minimalStack()
	stack.Plugins = PluginsSection{
		"fmt": Plugin{
			Source: "npm:@franlol/opencode-md-table-formatter@latest",
		},
	}
	errs := validatePlugins(stack)
	if errs.HasErrors() {
		t.Errorf("npm source should not error: %v", errs)
	}
}

func TestValidatePlugins_ProvidesEnumValidation(t *testing.T) {
	stack := minimalStack()
	stack.Plugins = PluginsSection{
		"advance": Plugin{
			Source:   "https://github.com/SOFAS/SOFA.git",
			Checkout: "~/dev/sofa",
			Path:     "~/dev/sofa/plugin",
			Provides: []ProvidesCategory{"adv-commands", "invalid-category"},
		},
	}
	errs := validatePlugins(stack)
	found := false
	for _, e := range errs {
		if e.Path == "plugins.advance.provides[1]" {
			found = true
		}
	}
	if !found {
		t.Error("expected provides enum error for invalid-category")
	}
}

func TestValidatePlugins_ValidGitSourceOK(t *testing.T) {
	stack := minimalStack()
	stack.Plugins = PluginsSection{
		"advance": Plugin{
			Source:   "https://github.com/Sharper-Flow/Advance.git",
			Ref:      "trunk",
			Checkout: "~/dev/oc-plugins/advance",
			Subdir:   "plugin",
			Path:     "{checkout}/{subdir}",
			Provides: []ProvidesCategory{"adv-commands", "adv-agents"},
		},
	}
	errs := validatePlugins(stack)
	if errs.HasErrors() {
		t.Errorf("valid git plugin should pass: %v", errs)
	}
}

func TestValidateInstructions_EmptyOrderErrors(t *testing.T) {
	stack := minimalStack()
	stack.Instructions = InstructionsSection{
		Order: []string{},
	}
	errs := validateInstructions(stack)
	if !errs.HasErrors() {
		t.Error("empty order should error")
	}
}

func TestValidateInstructions_ValidOrderOK(t *testing.T) {
	stack := minimalStack()
	stack.Instructions = InstructionsSection{
		Order: []string{"identity.md", "rules.yaml", "{checkout}/ADV_INSTRUCTIONS.md"},
	}
	errs := validateInstructions(stack)
	if errs.HasErrors() {
		t.Errorf("valid order should pass: %v", errs)
	}
}

func TestValidateTemporal_ProducesAdvisoryWarning(t *testing.T) {
	stack := minimalStack()
	stack.Temporal = &TemporalSection{Enabled: boolPtr(false)}
	warnings := validateTemporalReserved(stack)
	// validateTemporalReserved produces Warnings (not ValidationErrors),
	// so we check warnings are non-empty.
	if len(warnings) == 0 {
		t.Error("expected advisory warning for [temporal], got none")
	}
}

func TestValidate_PluginsAndInstructionsIntegrated(t *testing.T) {
	// Full Validate() call should include plugin and instruction errors.
	stack := minimalStack()
	stack.Plugins = PluginsSection{
		"bad": Plugin{
			Source: "https://github.com/bad/bad.git",
			// missing checkout
			Path: "/tmp/bad",
		},
	}
	stack.Instructions = InstructionsSection{Order: []string{}}

	err := stack.Validate()
	if err == nil {
		t.Fatal("expected Validate() to return errors")
	}
	errs, ok := err.(ValidationErrors)
	if !ok {
		t.Fatalf("expected ValidationErrors, got %T", err)
	}
	foundPlugin := false
	foundInstr := false
	for _, e := range errs {
		if e.Path == "plugins.bad.checkout" {
			foundPlugin = true
		}
		if e.Path == "instructions.order" {
			foundInstr = true
		}
	}
	if !foundPlugin {
		t.Error("expected plugins.bad.checkout error in Validate()")
	}
	if !foundInstr {
		t.Error("expected instructions.order error in Validate()")
	}
}

// boolPtr is a test helper.
func boolPtr(b bool) *bool { return &b }

// minimalStack returns a valid minimal stack for testing plugin validation.
func minimalStack() *Stack {
	return &Stack{
		Meta: Meta{Version: "1.0.0"},
		MCP: MCPSection{
			Servers: map[string]Server{
				"vision": {Port: 6275, Type: "daemon"},
			},
		},
	}
}
