package render

import (
	"encoding/json"
	"fmt"
)

// MergeMCP merges declared fragments into existing opencode.json bytes.
// Top-level keys and non-declared .mcp entries are preserved.
func MergeMCP(existing []byte, declared map[string]Fragment) ([]byte, error) {
	root := map[string]any{}
	if len(existing) > 0 {
		if err := json.Unmarshal(existing, &root); err != nil {
			return nil, fmt.Errorf("parse existing opencode.json: %w", err)
		}
	}
	var mcp map[string]any
	if raw, ok := root["mcp"]; ok {
		cast, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("existing .mcp is not an object")
		}
		mcp = cast
	} else {
		mcp = map[string]any{}
	}
	for name, frag := range declared {
		mcp[name] = frag
	}
	root["mcp"] = mcp
	b, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}
