package render

import (
	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
	pluginpkg "github.com/Sharper-Flow/Opencode-Advance/internal/plugin"
)

// RenderPluginFragment returns flat string entry for opencode.json .plugin.
// Git plugins emit resolved path. npm plugins emit literal pkg@version form.
func RenderPluginFragment(p cfg.Plugin) string {
	if p.IsNPMSource() {
		return pluginpkg.RenderLiteral(p.Source)
	}
	return p.Path
}
