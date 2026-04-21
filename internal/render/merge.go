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
	b, err := jsonMarshalIndent(root)
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

// MergeObject performs a recursive deep merge of declared into existing JSON.
// For each top-level key in declared:
//   - If both existing and declared values are maps, recursively merge.
//   - Otherwise overwrite with the declared value.
//
// All existing top-level keys not in declared are preserved.
// Arrays are overwritten (not merged) to match MergeArray behavior.
//
// This is the primitive underlying PlanProviders, PlanPermissions, PlanLSP
// and the per-target merge steps in composeApplyPlan.
func MergeObject(existing []byte, declared map[string]any) ([]byte, error) {
	root := map[string]any{}
	if len(existing) > 0 {
		if err := json.Unmarshal(existing, &root); err != nil {
			return nil, fmt.Errorf("parse existing: %w", err)
		}
	}
	for key, decVal := range declared {
		decMap, decIsMap := decVal.(map[string]any)
		existingVal, hasExisting := root[key]
		existingMap, existingIsMap := existingVal.(map[string]any)

		if hasExisting && existingIsMap && decIsMap {
			// Recursive merge: declared wins on conflict, existing user keys preserved.
			merged, err := mergeMap(existingMap, decMap)
			if err != nil {
				return nil, fmt.Errorf("merge %s: %w", key, err)
			}
			root[key] = merged
		} else {
			// Overwrite with declared value.
			root[key] = decVal
		}
	}

	b, err := jsonMarshalIndent(root)
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

// mergeMap merges src into dst recursively. src wins on conflict; dst keys
// not in src are preserved.
func mergeMap(dst, src map[string]any) (map[string]any, error) {
	for key, srcVal := range src {
		srcMap, srcIsMap := srcVal.(map[string]any)
		dstVal, dstHas := dst[key]
		dstMap, dstIsMap := dstVal.(map[string]any)

		if dstHas && dstIsMap && srcIsMap {
			merged, err := mergeMap(dstMap, srcMap)
			if err != nil {
				return nil, err
			}
			dst[key] = merged
		} else {
			dst[key] = srcVal
		}
	}
	return dst, nil
}

// jsonMarshalIndent produces deterministic JSON with sorted keys and 2-space indentation.
func jsonMarshalIndent(v any) ([]byte, error) {
	b, err := json.Marshal(v) // json.Marshal sorts map keys
	if err != nil {
		return nil, err
	}
	var raw any
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, err
	}
	return json.MarshalIndent(raw, "", "  ")
}

// MergeWatcherIgnore merges declared ignore entries into .watcher.ignore while
// preserving user-added ignore entries and other watcher object keys.
func MergeWatcherIgnore(existing []byte, declared []string) ([]byte, error) {
	root := map[string]any{}
	if len(existing) > 0 {
		if err := json.Unmarshal(existing, &root); err != nil {
			return nil, fmt.Errorf("parse existing: %w", err)
		}
	}

	var watcher map[string]any
	if raw, ok := root["watcher"]; ok {
		cast, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("existing .watcher is not an object")
		}
		watcher = cast
	} else {
		watcher = map[string]any{}
	}

	var existingIgnore []string
	if raw, ok := watcher["ignore"]; ok {
		switch v := raw.(type) {
		case []any:
			for _, item := range v {
				if s, ok := item.(string); ok {
					existingIgnore = append(existingIgnore, s)
				}
			}
		case []string:
			existingIgnore = v
		default:
			return nil, fmt.Errorf("existing .watcher.ignore is not an array")
		}
	}

	declaredSet := make(map[string]bool, len(declared))
	for _, d := range declared {
		declaredSet[canonicalPath(d)] = true
	}

	mergedIgnore := make([]any, 0, len(declared)+len(existingIgnore))
	for _, d := range declared {
		mergedIgnore = append(mergedIgnore, d)
	}
	for _, e := range existingIgnore {
		if declaredSet[canonicalPath(e)] {
			continue
		}
		mergedIgnore = append(mergedIgnore, e)
	}

	watcher["ignore"] = mergedIgnore
	root["watcher"] = watcher
	b, err := jsonMarshalIndent(root)
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}
