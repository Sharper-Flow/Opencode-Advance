// Package render produces and applies opencode.json and vision servers.yaml
// render plans from a resolved stack configuration.
//
// The package is organized around three concerns:
//
//   - Planning: PlanMCP builds a deterministic Plan describing the before/after
//     byte content of opencode.json and vision/servers.yaml for a given Stack.
//   - Merging: MergeMCP folds declared MCP fragments into an existing
//     opencode.json while preserving user-owned keys outside the managed scope.
//   - Applying: WriteAtomic performs temp-file + rename writes with optional
//     .bak.<UnixNano> backups so partial failures can be rolled back.
//
// Redact is exported so doctor / health surfaces can scrub secret-bearing
// strings (API keys, URL credentials, env-var assignments) before they are
// embedded in user-visible output.
package render
