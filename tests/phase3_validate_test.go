package tests

import (
	"testing"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

// TestValidateProviders_OK verifies no errors for a valid providers section.
func TestValidateProviders_OK(t *testing.T) {
	toml := `
[meta]
version = "1.0.0"

[providers.google]
[providers.google.models.gemini]
name = "Gemini"
context = 100000
output = 10000
inputs = ["text"]
outputs = ["text"]

[providers.openai]
[providers.openai.models.gpt5]
name = "GPT 5"
variants.high.text_verbosity = "high"
`
	stack, err := cfg.Parse([]byte(toml))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if err := stack.Validate(); err != nil {
		t.Fatalf("Validate unexpectedly failed: %v", err)
	}
}

// TestValidateProviders_Errors verifies validation errors for invalid providers.
func TestValidateProviders_Errors(t *testing.T) {
	cases := []struct {
		name    string
		toml    string
		wantErr string
	}{
		{
			name: "empty provider table",
			toml: `
[meta]
version = "1.0.0"

[providers.google]
`,
			wantErr: "[providers.google]:",
		},
		{
			name: "model missing name and variants",
			toml: `
[meta]
version = "1.0.0"

[providers.google]
[providers.google.models.gemini]
context = 100000
output = 10000
`,
			wantErr: "[providers.google.models.gemini]:",
		},
		{
			name: "negative context",
			toml: `
[meta]
version = "1.0.0"

[providers.google]
[providers.google.models.gemini]
name = "Gemini"
context = -1
output = 10000
`,
			wantErr: "[providers.google.models.gemini.context]:",
		},
		{
			name: "empty inputs entry",
			toml: `
[meta]
version = "1.0.0"

[providers.google]
[providers.google.models.gemini]
name = "Gemini"
context = 100000
output = 10000
inputs = ["text", ""]
`,
			wantErr: "[providers.google.models.gemini.inputs[1]]:",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stack, err := cfg.Parse([]byte(tc.toml))
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}
			err = stack.Validate()
			if err == nil {
				t.Fatal("Validate expected error, got nil")
			}
			if !cfg.ValidationErrorContains(err, tc.wantErr) {
				t.Errorf("Validate error = %v; want containing %q", err, tc.wantErr)
			}
		})
	}
}

// TestValidatePermissions_OK verifies no errors for a valid permissions section.
func TestValidatePermissions_OK(t *testing.T) {
	toml := `
[meta]
version = "1.0.0"

[permissions]
default = "allow"
doom_loop = "ask"

[permissions.external_directory]
all = "ask"
dev = "allow"

[permissions.bash]
all = "allow"
git_push = "ask"
`
	stack, err := cfg.Parse([]byte(toml))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if err := stack.Validate(); err != nil {
		t.Fatalf("Validate unexpectedly failed: %v", err)
	}
}

// TestValidatePermissions_Errors verifies validation errors for invalid permissions.
func TestValidatePermissions_Errors(t *testing.T) {
	cases := []struct {
		name    string
		toml    string
		wantErr string
	}{
		{
			name: "invalid default value",
			toml: `
[meta]
version = "1.0.0"

[permissions]
default = "invalid"
`,
			wantErr: "[permissions.default]:",
		},
		{
			name: "invalid doom_loop value",
			toml: `
[meta]
version = "1.0.0"

[permissions]
doom_loop = "maybe"
`,
			wantErr: "[permissions.doom_loop]:",
		},
		{
			name: "invalid external_directory action",
			toml: `
[meta]
version = "1.0.0"

[permissions.external_directory]
all = "invalid"
`,
			wantErr: "[permissions.external_directory.all]:",
		},
		{
			name: "invalid bash action",
			toml: `
[meta]
version = "1.0.0"

[permissions.bash]
all = "deny"
rm_rf = "maybe"
`,
			wantErr: "[permissions.bash.rm_rf]:",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stack, err := cfg.Parse([]byte(tc.toml))
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}
			err = stack.Validate()
			if err == nil {
				t.Fatal("Validate expected error, got nil")
			}
			if !cfg.ValidationErrorContains(err, tc.wantErr) {
				t.Errorf("Validate error = %v; want containing %q", err, tc.wantErr)
			}
		})
	}
}

// TestValidateWatcher_OK verifies no errors for a valid watcher section.
func TestValidateWatcher_OK(t *testing.T) {
	toml := `
[meta]
version = "1.0.0"

[watcher]
ignore = ["node_modules/**", ".git/**"]
`
	stack, err := cfg.Parse([]byte(toml))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if err := stack.Validate(); err != nil {
		t.Fatalf("Validate unexpectedly failed: %v", err)
	}
}

// TestValidateWatcher_Errors verifies validation errors for invalid watcher.
func TestValidateWatcher_Errors(t *testing.T) {
	cases := []struct {
		name    string
		toml    string
		wantErr string
	}{
		{
			name: "empty ignore entry",
			toml: `
[meta]
version = "1.0.0"

[watcher]
ignore = ["node_modules/**", ""]
`,
			wantErr: "[watcher.ignore[1]]:",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stack, err := cfg.Parse([]byte(tc.toml))
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}
			err = stack.Validate()
			if err == nil {
				t.Fatal("Validate expected error, got nil")
			}
			if !cfg.ValidationErrorContains(err, tc.wantErr) {
				t.Errorf("Validate error = %v; want containing %q", err, tc.wantErr)
			}
		})
	}
}

// TestValidateLSP_OK verifies no errors for a valid LSP section.
func TestValidateLSP_OK(t *testing.T) {
	toml := `
[meta]
version = "1.0.0"

[lsp.pyright]
command = ["pyright", "lsp"]
extensions = [".py", ".pyi"]

[lsp.ts]
command = ["typescript", "lsp"]
extensions = [".ts", ".tsx"]
disabled = true
`
	stack, err := cfg.Parse([]byte(toml))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if err := stack.Validate(); err != nil {
		t.Fatalf("Validate unexpectedly failed: %v", err)
	}
}

// TestValidateLSP_Errors verifies validation errors for invalid LSP.
func TestValidateLSP_Errors(t *testing.T) {
	cases := []struct {
		name    string
		toml    string
		wantErr string
	}{
		{
			name: "enabled server with empty command",
			toml: `
[meta]
version = "1.0.0"

[lsp.pyright]
command = []
extensions = [".py"]
`,
			wantErr: "[lsp.pyright.command]:",
		},
		{
			name: "empty LSP table",
			toml: `
[meta]
version = "1.0.0"

[lsp.pyright]
`,
			wantErr: "[lsp.pyright]:",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stack, err := cfg.Parse([]byte(tc.toml))
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}
			err = stack.Validate()
			if err == nil {
				t.Fatal("Validate expected error, got nil")
			}
			if !cfg.ValidationErrorContains(err, tc.wantErr) {
				t.Errorf("Validate error = %v; want containing %q", err, tc.wantErr)
			}
		})
	}
}
