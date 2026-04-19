package render

import "regexp"

// Secret-bearing patterns scrubbed from plan Before/After byte slices before
// the plan is emitted via `oca debug plan` or `oca apply --dry-run --output json`.
// Patterns mirror Vision's admin scrubber so OCA and Vision present identical
// redaction behavior for operators inspecting rendered configuration.
var redactPatterns = []*regexp.Regexp{
	// Authorization: Bearer <token>  |  Authorization=<token>
	regexp.MustCompile(`(?i)(authorization)\s*[:=]\s*(?:bearer\s+)?(\S+)`),
	// KEY_NAME=value where KEY_NAME ends with TOKEN/KEY/SECRET/PASSWORD/CREDENTIAL/PRIVATE/AUTH
	regexp.MustCompile(`(?i)([A-Z][A-Z0-9_]*(?:TOKEN|KEY|SECRET|PASSWORD|CREDENTIAL|PRIVATE|AUTH))\s*[:=]\s*(\S+)`),
	// Generic "password=<value>" in rendered JSON/YAML
	regexp.MustCompile(`(?i)"?(password)"?\s*[:=]\s*"?([^"\s,}]+)"?`),
	// "api_key": "<value>" / api-key=<value> style keys that don't match the
	// uppercase pattern above (JSON-rendered mcp env blocks often use lower case).
	regexp.MustCompile(`(?i)"?(api[_-]?key|access[_-]?token|secret[_-]?key|private[_-]?key|bearer[_-]?token)"?\s*[:=]\s*"?([^"\s,}]+)"?`),
}

// redactBytes returns a copy of b with secret values replaced by ***REDACTED***.
// Preserves key names so operators can still correlate which field held the
// secret. Returns nil for nil input so callers can distinguish absent slices.
func redactBytes(b []byte) []byte {
	if b == nil {
		return nil
	}
	out := b
	for _, re := range redactPatterns {
		out = re.ReplaceAllFunc(out, func(match []byte) []byte {
			sub := re.FindSubmatch(match)
			if len(sub) < 2 {
				return []byte("***REDACTED***")
			}
			key := sub[1]
			// Preserve the original separator style so redacted output remains
			// readable as JSON/YAML/env syntax.
			result := make([]byte, 0, len(key)+len("=***REDACTED***")+2)
			// Detect leading quote on the key (JSON style).
			quoted := len(match) > 0 && match[0] == '"'
			if quoted {
				result = append(result, '"')
			}
			result = append(result, key...)
			if quoted {
				result = append(result, '"')
			}
			// Detect separator: colon preferred for JSON/YAML, equals otherwise.
			sep := "="
			for _, c := range match[len(sub[1]):] {
				if c == ':' {
					sep = ": "
					break
				}
				if c == '=' {
					break
				}
			}
			result = append(result, sep...)
			if quoted {
				result = append(result, '"')
			}
			result = append(result, []byte("***REDACTED***")...)
			if quoted {
				result = append(result, '"')
			}
			return result
		})
	}
	// Ensure we always return a fresh slice so callers can't accidentally
	// mutate the original plan's backing array via the returned bytes.
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
			Before:     redactBytes(t.Before),
			After:      redactBytes(t.After),
			Mode:       t.Mode,
			BackupPath: t.BackupPath,
			Reason:     t.Reason,
		}
	}
	return out
}
