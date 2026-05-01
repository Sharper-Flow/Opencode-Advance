package plugin

import "testing"

func TestValidateNPMSource_ValidPatterns(t *testing.T) {
	cases := []struct {
		input string
		pkg   string
	}{
		{"npm:@franlol/opencode-md-table-formatter@latest", "opencode-md-table-formatter"},
		{"npm:some-pkg@1.2.3", "some-pkg"},
		{"npm:@scope/pkg@^4.0.0", "pkg"},
	}
	for _, tc := range cases {
		pkg, err := ValidateNPMSource(tc.input)
		if err != nil {
			t.Errorf("ValidateNPMSource(%q): unexpected error: %v", tc.input, err)
		}
		if pkg != tc.pkg {
			t.Errorf("ValidateNPMSource(%q) pkg = %q, want %q", tc.input, pkg, tc.pkg)
		}
	}
}

func TestValidateNPMSource_InvalidPatterns(t *testing.T) {
	cases := []string{
		"",                               // empty
		"npm:",                           // empty after prefix
		"npm:@",                          // just scope, no package
		"https://github.com/foo/bar.git", // not npm
	}
	for _, tc := range cases {
		_, err := ValidateNPMSource(tc)
		if err == nil {
			t.Errorf("ValidateNPMSource(%q): expected error, got nil", tc)
		}
	}
}

func TestRenderLiteral(t *testing.T) {
	cases := []struct {
		source string
		want   string
	}{
		{"npm:@franlol/opencode-md-table-formatter@latest", "@franlol/opencode-md-table-formatter@latest"},
		{"npm:some-pkg@1.2.3", "some-pkg@1.2.3"},
		{"npm:@scope/pkg@^4.0.0", "@scope/pkg@^4.0.0"},
	}
	for _, tc := range cases {
		got := RenderLiteral(tc.source)
		if got != tc.want {
			t.Errorf("RenderLiteral(%q) = %q, want %q", tc.source, got, tc.want)
		}
	}
}
