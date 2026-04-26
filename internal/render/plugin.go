package render

import (
	"path/filepath"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
	pluginpkg "github.com/Sharper-Flow/Opencode-Advance/internal/plugin"
)

// RenderPluginFragment returns flat string entry for opencode.json .plugin.
// Git plugins emit resolved path. npm plugins emit literal pkg@version form.
// Local plugins resolve path relative to repo root (cwd where oca apply runs).
func RenderPluginFragment(p cfg.Plugin) string {
	if p.IsNPMSource() {
		return pluginpkg.RenderLiteral(p.Source)
	}
	if p.IsLocalSource() {
		// local:plugins/oca → resolve plugins/oca/dist/index.js relative to cwd
		localDir := p.Source[6:] // strip "local:" prefix
		if p.Path != "" {
			return filepath.Join(localDir, p.Path)
		}
		// Default path for local plugins
		return filepath.Join(localDir, "dist", "index.js")
	}
	return p.Path
}
