package brand

import (
	"strings"
	"testing"
)

func TestAssetsExposeCanonicalVariants(t *testing.T) {
	assets := MustAssets()

	if assets.Medium != "OpenCode ADVANCE" {
		t.Fatalf("medium variant = %q, want %q", assets.Medium, "OpenCode ADVANCE")
	}

	if assets.Short != "OCA" {
		t.Fatalf("short variant = %q, want %q", assets.Short, "OCA")
	}

	if len(assets.FullLines) != 3 {
		t.Fatalf("full line count = %d, want 3", len(assets.FullLines))
	}

	if !strings.Contains(assets.FullLines[0], "░▟█▙") {
		t.Fatalf("full wordmark missing stylized A: %q", assets.FullLines[0])
	}

	if assets.Palette.IVORY.Hex != "#E8E6E3" {
		t.Fatalf("ivory hex = %q, want %q", assets.Palette.IVORY.Hex, "#E8E6E3")
	}

	if assets.Palette.INDIGO.Ansi256 != 103 {
		t.Fatalf("indigo ansi256 = %d, want %d", assets.Palette.INDIGO.Ansi256, 103)
	}
}

func TestDetectColorMode(t *testing.T) {
	tests := []struct {
		name string
		env  Environment
		want ColorMode
	}{
		{name: "no color disables styling", env: Environment{IsTTY: true, NoColor: true, Term: "xterm-256color", ColorTerm: "truecolor"}, want: ColorModeMono},
		{name: "non tty is mono", env: Environment{IsTTY: false, Term: "xterm-256color", ColorTerm: "truecolor"}, want: ColorModeMono},
		{name: "truecolor tty wins", env: Environment{IsTTY: true, Term: "xterm-256color", ColorTerm: "truecolor"}, want: ColorModeTruecolor},
		{name: "256 color tty fallback", env: Environment{IsTTY: true, Term: "xterm-256color"}, want: ColorMode256},
		{name: "plain tty fallback", env: Environment{IsTTY: true, Term: "xterm"}, want: ColorModeMono},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DetectColorMode(tt.env); got != tt.want {
				t.Fatalf("DetectColorMode() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderMediumMonoUsesExplicitAdvanceMarker(t *testing.T) {
	got := Render(VariantMedium, ColorModeMono)
	if got != "OpenCode *ADVANCE*" {
		t.Fatalf("mono medium = %q, want %q", got, "OpenCode *ADVANCE*")
	}
}

func TestRenderFull256AppliesSplitColorSequences(t *testing.T) {
	got := Render(VariantFull, ColorMode256)

	if !strings.Contains(got, "\x1b[38;5;255m") {
		t.Fatalf("full 256 render missing ivory ansi code: %q", got)
	}

	if !strings.Contains(got, "\x1b[38;5;103m") {
		t.Fatalf("full 256 render missing indigo ansi code: %q", got)
	}

	if !strings.Contains(got, "░█▀█░█▀█░█▀▀") {
		t.Fatalf("full 256 render missing left half glyphs: %q", got)
	}

	if !strings.Contains(got, "░▟█▙░█▀▄░█░█") {
		t.Fatalf("full 256 render missing right half glyphs: %q", got)
	}
}

func TestParsePaletteRejectsMissingRequiredEntries(t *testing.T) {
	_, err := parsePalette([]byte("IVORY=#E8E6E3,255,232,230,227\n"))
	if err == nil {
		t.Fatal("parsePalette() error = nil, want missing required color entry")
	}

	if !strings.Contains(err.Error(), "missing required color entry") {
		t.Fatalf("parsePalette() error = %q, want missing required color entry", err)
	}
}

func TestSplitWordmarkLineRejectsMalformedInput(t *testing.T) {
	_, _, ok := splitWordmarkLine("no-delimiter-here")
	if ok {
		t.Fatal("splitWordmarkLine() ok = true, want false for malformed input")
	}

	left, right, ok := splitWordmarkLine("left   right")
	if !ok || left != "left" || right != "right" {
		t.Fatalf("splitWordmarkLine() = (%q, %q, %v), want (%q, %q, true)", left, right, ok, "left", "right")
	}
}
