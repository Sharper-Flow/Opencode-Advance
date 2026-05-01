package migrate

// Stale agent names that were renamed/consolidated by Advance.
// These should not be emitted into the migrated stack.toml.
var staleAgents = map[string]bool{
	"scout.md":  true,
	"refine.md": true,
}

// Stale instruction names that are no longer shipped.
var staleInstructions = map[string]bool{
	"post_install_verification.md": true,
	"criteria-prioritizer.md":      true,
}

// Stale skill names that are no longer shipped.
var staleSkills = map[string]bool{
	"mcp-selection": true, // superseded by mcp-selection skill in OCA
}

func isStaleAgent(name string) bool {
	return staleAgents[name]
}

func isStaleInstruction(name string) bool {
	return staleInstructions[name]
}

func isStaleSkill(name string) bool {
	return staleSkills[name]
}
