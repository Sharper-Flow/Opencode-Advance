package plugin

import (
	"fmt"
	"strings"
)

// ValidateNPMSource checks that source is a valid "npm:pkg@version" string.
// Returns the extracted package name (without version) on success.
// The npm: prefix must be followed by a non-empty package name with a version.
func ValidateNPMSource(source string) (string, error) {
	const prefix = "npm:"
	if !strings.HasPrefix(source, prefix) {
		return "", fmt.Errorf("npm source must start with %q", prefix)
	}
	body := strings.TrimPrefix(source, prefix)
	if body == "" {
		return "", fmt.Errorf("npm source is empty after %q prefix", prefix)
	}

	// body should be "pkg@version" or "@scope/pkg@version".
	// Find the last @ to split pkg from version (scoped packages have @ at start).
	atIdx := strings.LastIndex(body, "@")
	if atIdx <= 0 {
		return "", fmt.Errorf("npm source %q missing version (expected pkg@version)", source)
	}
	pkg := body[:atIdx]
	if pkg == "" || pkg == "@" {
		return "", fmt.Errorf("npm source %q has empty package name", source)
	}
	// Extract short name (strip scope prefix if scoped).
	short := pkg
	if strings.HasPrefix(short, "@") {
		if idx := strings.Index(short, "/"); idx >= 0 {
			short = short[idx+1:]
		}
	}
	return short, nil
}

// RenderLiteral strips the "npm:" prefix from an npm source string,
// producing the "pkg@version" literal that fits into opencode.json's
// flat plugin string array (confirmed in discovery D6).
func RenderLiteral(source string) string {
	return strings.TrimPrefix(source, "npm:")
}
