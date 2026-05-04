package migrate

import (
	"fmt"
	"time"
)

// EmitInit generates a minimal starter stack.toml for new users.
// Not derived from open-chad state — produces a hardcoded baseline.
func EmitInit() (string, error) {
	return fmt.Sprintf(`# stack.toml — OpenCode Advance starter stack
# Generated: %s
#
# Edit this file to declare your MCP servers, plugins, providers, and config.
# Then run: oca apply

[meta]
version = "1.0.0"
name = "default"
description = "Personal OpenCode Advance stack"

# ─── MCP servers ─────────────────────────────────────────────────────────────
# Declared here once. `+"`oca apply`"+` renders into both:
#   ~/.config/opencode/opencode.json  (.mcp section)
#   ~/.config/vision/servers.yaml     (Vision daemon registry)

[mcp.servers.vision]
port        = 6275
type        = "daemon"
required    = true
source      = "https://github.com/Sharper-Flow/vision"
description = "MCP daemon — manages lifecycle of all other MCP servers"

[mcp.servers.context7]
port        = 6276
command     = "npx"
args        = ["-y", "@upstash/context7-mcp@latest"]
timeout     = 10000
autostart   = true
source      = "https://github.com/upstash/context7"
description = "Library and API documentation lookup"

# ─── Plugins ─────────────────────────────────────────────────────────────────
# Git-sourced plugins are cloned, built, and wired automatically.
# NPM-sourced plugins use "npm:package@version" syntax.

[plugins.advance]
source   = "https://github.com/Sharper-Flow/Advance.git"
checkout = "{checkout}/advance"
path     = "plugin"
build    = ["cd plugin && pnpm install && pnpm build"]
provides = ["adv-commands", "adv-agents", "adv-skills", "adv-overlays"]
sync     = "scripts/sync-global.sh --fix"

# ─── Instructions ────────────────────────────────────────────────────────────
# OCA-owned instructions are copied from assets/instructions/.
# Plugin-provided instructions are merged in declared order.

[instructions]
order = [
  "{assets}/instructions/identity.md",
  "{assets}/instructions/rules.yaml",
]

# ─── Providers ───────────────────────────────────────────────────────────────
# Model providers with per-model limits and modality flags.

[providers.google]

[providers.google.models."gemini-2.5-flash"]
name    = "gemini-2.5-flash"
context = 32000
inputs  = ["text", "image"]
outputs = ["text"]

# ─── OpenCode toggles ────────────────────────────────────────────────────────
# Theme, default agent, and behavior flags.

[opencode]
theme = "obsidian"
`, time.Now().UTC().Format(time.RFC3339)), nil
}
