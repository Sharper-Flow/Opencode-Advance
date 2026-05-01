// Package migrate reads legacy configuration (open-chad state, existing
// opencode.json, etc.) and produces a valid stack.toml representing the
// current state.
//
// The migration pipeline:
//
//	ReadOpenChadState(cfg) → OpenChadState
//	EmitTOML(state)        → stack.toml string
//	EmitInit()             → minimal starter stack.toml
//
// Each source is optional — missing sources produce warnings, not errors.
// The emitted stack.toml is never auto-applied; the user reviews it first.
package migrate
