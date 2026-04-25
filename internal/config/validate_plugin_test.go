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

func TestValidateTemporal_NilSectionIsNoop(t *testing.T) {
	// nil Temporal → no validation errors.
	stack := minimalStack()
	errs := validateTemporal(stack)
	if errs.HasErrors() {
		t.Errorf("nil Temporal should produce no errors: %v", errs)
	}
}

func TestValidateTemporal_DefaultsLoopbackOK(t *testing.T) {
	// enabled=true, no address → defaults to 127.0.0.1:7233, no error.
	stack := minimalStack()
	stack.Temporal = &TemporalSection{Enabled: boolPtr(true)}
	errs := validateTemporal(stack)
	if errs.HasErrors() {
		t.Errorf("default loopback config should pass: %v", errs)
	}
	// Verify defaults were applied.
	if stack.Temporal.Address != "127.0.0.1:7233" {
		t.Errorf("Address = %q, want 127.0.0.1:7233", stack.Temporal.Address)
	}
	if stack.Temporal.Namespace != "default" {
		t.Errorf("Namespace = %q, want default", stack.Temporal.Namespace)
	}
}

func TestValidateTemporal_DisabledNoop(t *testing.T) {
	// enabled=false → no validation, no defaults applied.
	stack := minimalStack()
	stack.Temporal = &TemporalSection{Enabled: boolPtr(false)}
	errs := validateTemporal(stack)
	if errs.HasErrors() {
		t.Errorf("disabled temporal should produce no errors: %v", errs)
	}
}

func TestValidateTemporal_NonLoopbackWithoutAllowRemote(t *testing.T) {
	// Non-loopback address without allow_remote=true → error.
	stack := minimalStack()
	stack.Temporal = &TemporalSection{
		Enabled:     boolPtr(true),
		Address:     "10.0.0.1:7233",
		AllowRemote: boolPtr(false),
	}
	errs := validateTemporal(stack)
	if !errs.HasErrors() {
		t.Fatal("expected error for non-loopback without allow_remote")
	}
	found := false
	for _, e := range errs {
		if e.Path == "temporal.allow_remote" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected allow_remote error, got: %v", errs)
	}
}

func TestValidateTemporal_NonLoopbackWithAllowRemote(t *testing.T) {
	// Non-loopback address WITH allow_remote=true → OK.
	stack := minimalStack()
	stack.Temporal = &TemporalSection{
		Enabled:     boolPtr(true),
		Address:     "10.0.0.1:7233",
		AllowRemote: boolPtr(true),
	}
	errs := validateTemporal(stack)
	if errs.HasErrors() {
		t.Errorf("non-loopback with allow_remote should pass: %v", errs)
	}
}

func TestValidateTemporal_IPv6MappedLoopback(t *testing.T) {
	// ::ffff:127.0.0.1 is a loopback address (IPv6-mapped IPv4).
	stack := minimalStack()
	stack.Temporal = &TemporalSection{
		Enabled: boolPtr(true),
		Address: "::ffff:127.0.0.1:7233",
	}
	errs := validateTemporal(stack)
	if errs.HasErrors() {
		t.Errorf("IPv6-mapped loopback should pass without allow_remote: %v", errs)
	}
}

func TestValidateTemporal_IPv6MappedNonLoopback(t *testing.T) {
	// ::ffff:10.0.0.1 is NOT loopback → requires allow_remote.
	stack := minimalStack()
	stack.Temporal = &TemporalSection{
		Enabled: boolPtr(true),
		Address: "::ffff:10.0.0.1:7233",
	}
	errs := validateTemporal(stack)
	if !errs.HasErrors() {
		t.Fatal("expected error for IPv6-mapped non-loopback without allow_remote")
	}
}

func TestValidateTemporal_ExplicitDefaultsPreserved(t *testing.T) {
	// Explicit address matching default should not error.
	stack := minimalStack()
	stack.Temporal = &TemporalSection{
		Enabled:   boolPtr(true),
		Address:   "127.0.0.1:7233",
		Namespace: "production",
	}
	errs := validateTemporal(stack)
	if errs.HasErrors() {
		t.Errorf("explicit loopback config should pass: %v", errs)
	}
	if stack.Temporal.Namespace != "production" {
		t.Errorf("Namespace = %q, want production", stack.Temporal.Namespace)
	}
}

func TestValidateTemporal_NilEnabledDefaultsTrue(t *testing.T) {
	// nil Enabled (field absent) treated as enabled → defaults applied.
	stack := minimalStack()
	stack.Temporal = &TemporalSection{
		Address: "127.0.0.1:7233",
	}
	errs := validateTemporal(stack)
	if errs.HasErrors() {
		t.Errorf("nil Enabled with loopback should pass: %v", errs)
	}
	if stack.Temporal.Address != "127.0.0.1:7233" {
		t.Errorf("Address = %q, want 127.0.0.1:7233", stack.Temporal.Address)
	}
}

func TestValidateTemporal_IntegratedWithValidate(t *testing.T) {
	// Full Validate() call includes temporal errors.
	stack := minimalStack()
	stack.Temporal = &TemporalSection{
		Enabled:     boolPtr(true),
		Address:     "192.168.1.1:7233",
		AllowRemote: boolPtr(false),
	}
	err := stack.Validate()
	if err == nil {
		t.Fatal("expected Validate() to return errors for non-loopback temporal")
	}
	if !ValidationErrorContains(err, "allow_remote") {
		t.Errorf("expected allow_remote error in Validate(), got: %v", err)
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
