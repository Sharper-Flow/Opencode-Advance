package render

import "regexp"

// Each redaction pattern has exactly three capture groups:
//
//	group 1: everything up to (and including) the separator + opening quote
//	group 2: the secret value to replace
//	group 3: the trailing quote (if any) — empty when the value was unquoted
//
// This lets the replacer preserve the original separator (":", ": ", "="),
// surrounding whitespace, and quoting style without guessing, which keeps the
// redacted output valid JSON/YAML/env syntax.
var redactPatterns = []*regexp.Regexp{
	// Authorization: Bearer <token>  |  Authorization=<token>
	// No value-side quotes are matched here; the header form never quotes.
	regexp.MustCompile(`(?i)(authorization\s*[:=]\s*(?:bearer\s+)?)(\S+)()`),

	// URL-embedded credentials: https://user:pass@host -> https://user:***REDACTED***@host
	// Preserve scheme + username; replace password only. Group 3 anchors the '@'.
	regexp.MustCompile(`(?i)(https?://[^:/\s]+:)([^@\s/]+)(@)`),

	// KEY_NAME=value (uppercase env-style) where KEY_NAME ends with a secret
	// suffix. An optional trailing quote after the key covers the JSON form
	// `"GITHUB_TOKEN":"value"` where a `"` sits between the key and separator.
	regexp.MustCompile(`([A-Z][A-Z0-9_]*(?:TOKEN|KEY|SECRET|PASSWORD|CREDENTIAL|PRIVATE|AUTH)"?\s*[:=]\s*"?)([^"\s,}\]]+)("?)`),

	// key_name=value (lowercase env-style) — same suffix list as the uppercase
	// form above. Covers `api_token=...`, `aws_secret_access_key=...`, etc.
	regexp.MustCompile(`([a-z][a-z0-9_]*(?:token|key|secret|password|credential|private|auth)"?\s*[:=]\s*"?)([^"\s,}\]]+)("?)`),

	// Hyphenated/bracketed JSON-style secret keys that don't match the pure
	// env-name regexes above (e.g. `"api-key": "value"`, `"bearer_token":"x"`).
	regexp.MustCompile(`(?i)("?(?:api[_-]?key|access[_-]?token|secret[_-]?key|private[_-]?key|bearer[_-]?token)"?\s*[:=]\s*"?)([^"\s,}\]]+)("?)`),

	// Generic "password": "<value>" — lower priority than the env-style rules
	// so it only fires when the above haven't already matched.
	regexp.MustCompile(`(?i)("?password"?\s*[:=]\s*"?)([^"\s,}\]]+)("?)`),
}

const redactedMarker = "***REDACTED***"

// Redact returns a copy of b with secret values replaced by ***REDACTED***.
// Key names, separators, and quoting are preserved so operators can still
// correlate which field held the secret and the output remains valid
// JSON/YAML/env syntax. Returns nil when b is nil so callers can distinguish
// absent slices from empty ones.
//
// Exported so other packages (notably internal/health) can redact error
// payloads before they surface in operator-facing output.
func Redact(b []byte) []byte {
	if b == nil {
		return nil
	}
	out := b
	for _, re := range redactPatterns {
		out = re.ReplaceAllFunc(out, func(match []byte) []byte {
			sub := re.FindSubmatch(match)
			if len(sub) < 4 {
				// Defensive: shouldn't happen given our fixed 3-group shape.
				return []byte(redactedMarker)
			}
			prefix := sub[1]
			trailingQuote := sub[3]
			result := make([]byte, 0, len(prefix)+len(redactedMarker)+len(trailingQuote))
			result = append(result, prefix...)
			result = append(result, redactedMarker...)
			result = append(result, trailingQuote...)
			return result
		})
	}
	// Always return a fresh slice so callers can't mutate the original plan's
	// backing array via the returned bytes.
	cp := make([]byte, len(out))
	copy(cp, out)
	return cp
}

// RedactPlan returns a deep copy of p with secret-bearing values in each
// target's Before and After byte slices redacted. Call this before emitting
// a plan over JSON (debug plan, apply --dry-run --output json).
//
// All non-secret fields (paths, ops, modes, reasons) are copied verbatim so
// operators retain full visibility into what OCA plans to do. Returns nil
// when p is nil so callers can pass through unpopulated plans safely.
func RedactPlan(p *Plan) *Plan {
	if p == nil {
		return nil
	}
	out := &Plan{
		Source:   p.Source,
		LockPath: p.LockPath,
		Targets:  make([]TargetOp, len(p.Targets)),
	}
	for i, t := range p.Targets {
		out.Targets[i] = TargetOp{
			Name:       t.Name,
			Path:       t.Path,
			Op:         t.Op,
			Before:     Redact(t.Before),
			After:      Redact(t.After),
			Mode:       t.Mode,
			BackupPath: t.BackupPath,
			Reason:     t.Reason,
		}
	}
	return out
}
