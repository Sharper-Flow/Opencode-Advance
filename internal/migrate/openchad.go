// Package migrate reads legacy configuration and produces valid stack.toml.
package migrate

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// tomlSafeNamePattern matches plugin names that are safe to use as unquoted
// TOML table keys. The OCA stack schema treats plugin names as Go identifiers
// downstream, so we restrict the alphabet to a conservative subset.
var tomlSafeNamePattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// translateMCPType maps legacy/external MCP transport names to OCA's schema
// enum, mirroring the URL-suffix logic in internal/config/validate.go's
// inferTransport (lines 474-491) so the emitted stack passes validation
// directly without relying on load-time inference (validate.go:206 rejects
// empty type values).
//
// Mappings (input type, input url) → (translated, warning):
//
//   - "remote", "<url ending /mcp>"     → "http", "translated remote → http"
//   - "remote", "<url not ending /mcp>" → "sse",  "translated remote → sse"
//   - "remote", ""                      → "",     "remote type with no url; cannot infer transport"
//   - "stdio"|"http"|"sse"|"daemon", *  → passthrough, no warning
//   - "", *                             → "", no warning (downstream may fail validation)
//   - any other, *                      → unchanged, warning describing the unknown type
func translateMCPType(typeIn, urlIn string) (translated string, warning string) {
	switch typeIn {
	case "":
		return "", ""
	case "stdio", "http", "sse", "daemon":
		return typeIn, ""
	case "remote":
		if urlIn == "" {
			return "", fmt.Sprintf("type %q with no url; cannot infer transport (set type manually)", typeIn)
		}
		if strings.HasSuffix(urlIn, "/mcp") {
			return "http", fmt.Sprintf("translated type %q → \"http\" (url ends /mcp)", typeIn)
		}
		return "sse", fmt.Sprintf("translated type %q → \"sse\" (url does not end /mcp)", typeIn)
	default:
		return typeIn, fmt.Sprintf("unknown type %q; passing through (will likely fail validation)", typeIn)
	}
}

// portFromURL parses an explicit numeric port out of a URL. Returns 0 when no
// explicit port is present (e.g., bare https://host without :443). Schemes
// without an explicit port (like https://mcp.grep.app) leave port=0; for sse
// transport this is acceptable because the schema only requires url.
func portFromURL(rawURL string) int {
	if rawURL == "" {
		return 0
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return 0
	}
	if u.Port() == "" {
		return 0
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil {
		return 0
	}
	return port
}

// classifyPluginEntry inspects an opencode.json plugin entry string and returns
// the PluginState fields appropriate for the source kind.
//
// Recognized forms (detection priority order):
//
//   - "<path with separators>" (contains '/' or '\\')
//     → local checkout: Source="", Checkout=<entry>, Name=basename
//     (npm-scoped form "@scope/pkg[@spec]" is detected within this branch via
//     leading '@' before path separator)
//
//   - "name@spec" (no path separator, '@' at position > 0)
//     → npm unscoped: Source="npm:<entry>", Name=<entry-without-spec>, Checkout=""
//
//   - bare name (no '@', no path separator)
//     → local checkout (preserves existing behavior): Source="",
//     Checkout=<entry>, Name=<entry>
//
// Returns an error if the derived Name contains TOML-unsafe characters that
// cannot be sanitized further. Callers should append a warning and skip the
// entry.
func classifyPluginEntry(entry string) (PluginState, error) {
	if entry == "" {
		return PluginState{}, fmt.Errorf("empty plugin entry")
	}

	var ps PluginState

	switch {
	case strings.ContainsAny(entry, `/\`):
		// Path-like form. Distinguish scoped npm ("@scope/pkg[@spec]") from a
		// real filesystem path by leading '@'.
		if strings.HasPrefix(entry, "@") {
			pkg := entry
			// Strip the trailing "@spec" suffix only if '@' appears beyond the
			// leading position (i.e., a version qualifier, not the scope marker).
			if at := strings.LastIndex(entry, "@"); at > 0 {
				pkg = entry[:at]
			}
			ps = PluginState{
				Name:     filepath.Base(pkg), // "@scope/pkg" → "pkg"
				Source:   "npm:" + entry,
				Checkout: "",
			}
			break
		}
		// Real filesystem path. Strip any trailing "@spec" from the basename
		// defensively (a path like "/abs/path/pkg@latest" should not leak '@'
		// into a TOML key).
		base := filepath.Base(entry)
		if at := strings.Index(base, "@"); at > 0 {
			base = base[:at]
		}
		ps = PluginState{
			Name:     base,
			Source:   "",
			Checkout: entry,
		}

	case strings.Contains(entry, "@") && !strings.HasPrefix(entry, "@"):
		// "name@spec" — npm unscoped with version pin.
		at := strings.Index(entry, "@")
		ps = PluginState{
			Name:     entry[:at],
			Source:   "npm:" + entry,
			Checkout: "",
		}

	default:
		// Bare name (no '@', no path separator). Preserve existing semantics:
		// treat as a checkout-like reference. OpenCode itself resolves these
		// against its plugin search path.
		ps = PluginState{
			Name:     entry,
			Source:   "",
			Checkout: entry,
		}
	}

	if !tomlSafeNamePattern.MatchString(ps.Name) {
		return PluginState{}, fmt.Errorf("plugin name %q contains TOML-unsafe characters", ps.Name)
	}
	return ps, nil
}

// ReaderConfig controls where ReadOpenChadState looks for source data.
type ReaderConfig struct {
	OpenChadRepo      string // path to open-chad checkout (e.g. ~/dev/open-chad)
	OpenCodeConfigDir string // path to opencode config (e.g. ~/.config/opencode)
	VisionConfigDir   string // path to vision config (e.g. ~/.config/vision)
	OcPluginsDir      string // path to plugin checkouts (e.g. ~/dev/oc-plugins)
}

// DefaultReaderConfig returns sensible defaults based on the environment.
func DefaultReaderConfig() ReaderConfig {
	home, _ := os.UserHomeDir()
	return ReaderConfig{
		OpenChadRepo:      filepath.Join(home, "dev", "open-chad"),
		OpenCodeConfigDir: filepath.Join(home, ".config", "opencode"),
		VisionConfigDir:   filepath.Join(home, ".config", "vision"),
		OcPluginsDir:      filepath.Join(home, "dev", "oc-plugins"),
	}
}

// OpenChadState is the intermediate representation of all discovered open-chad
// configuration before emission to stack.toml.
type OpenChadState struct {
	MCPServers     map[string]MCPServerState
	SlotGroups     map[string]SlotGroupState
	Plugins        []PluginState
	Instructions   []string
	Providers      map[string]ProviderState
	Permissions    *PermissionsState
	Watcher        *WatcherState
	LSP            map[string]LSPEntryState
	Skills         []string
	Formatters     map[string]FormatterState
	Commands       map[string]CommandState
	OpenCode       *OpenCodeState
	Session        *SessionState
	Warnings       []string
	SkippedSources []string
}

type MCPServerState struct {
	Type      string
	URL       string
	Port      int
	Command   string
	Args      []string
	Env       map[string]string
	EnvFile   string
	Autostart bool
	Required  bool
	Source    string
	Timeout   int
}

type SlotGroupState struct {
	Servers   []string
	GroupPort int
	Template  string
	// BasePort + Count match the canonical Vision YAML shape and OCA's
	// stack schema (validate.go:362,372). The legacy MinSlots/MaxSlots
	// fields never matched the real Vision YAML tags and were always zero.
	BasePort int
	Count    int
}

type PluginState struct {
	Name     string
	Source   string
	Ref      string
	Checkout string
	Build    []string
	Provides []string
}

type ProviderState struct {
	Name   string
	Models []ModelState
}

type ModelState struct {
	Name       string
	Limit      *LimitState
	Modalities []string
}

type LimitState struct {
	Tokens *int
	TPM    *int
	RPM    *int
	Image  *int
	Audio  *int
}

type PermissionsState struct {
	Default     string
	DoomLoop    string
	ExternalDir string
	Bash        string
}

type WatcherState struct {
	Ignore []string
}

type LSPEntryState struct {
	Command string
	Args    []string
}

type FormatterState struct {
	Command string
	Args    []string
}

type CommandState struct {
	Script  string
	Args    []string
	Env     map[string]string
	Timeout int
}

type SessionState struct {
	Prefix string
	Theme  string
}

type OpenCodeState struct {
	Theme             string
	DefaultAgent      string
	Share             *bool
	Snapshot          *bool
	Autoupdate        any // string or bool
	Compaction        *bool
	DisabledProviders []string
	EnabledProviders  []string
}

// ReadOpenChadState discovers and reads open-chad managed state from the
// filesystem. Each source is optional — missing sources produce warnings.
func ReadOpenChadState(cfg ReaderConfig) (*OpenChadState, error) {
	state := &OpenChadState{
		MCPServers:     make(map[string]MCPServerState),
		SlotGroups:     make(map[string]SlotGroupState),
		Providers:      make(map[string]ProviderState),
		LSP:            make(map[string]LSPEntryState),
		Formatters:     make(map[string]FormatterState),
		Commands:       make(map[string]CommandState),
		Warnings:       []string{},
		SkippedSources: []string{},
	}

	// 1. Read opencode.json (live OpenCode config)
	opencodePath := filepath.Join(cfg.OpenCodeConfigDir, "opencode.json")
	if _, err := os.Stat(opencodePath); err == nil {
		if err := readOpenCodeJSON(opencodePath, state); err != nil {
			state.Warnings = append(state.Warnings, fmt.Sprintf("opencode.json: %v", err))
		}
	} else {
		state.SkippedSources = append(state.SkippedSources, "opencode.json (not found)")
	}

	// 2. Read vision/servers.yaml
	visionPath := filepath.Join(cfg.VisionConfigDir, "servers.yaml")
	if _, err := os.Stat(visionPath); err == nil {
		if err := readVisionServers(visionPath, state); err != nil {
			state.Warnings = append(state.Warnings, fmt.Sprintf("vision/servers.yaml: %v", err))
		}
	} else {
		state.SkippedSources = append(state.SkippedSources, "vision/servers.yaml (not found)")
	}

	// 3. Discover plugins from oc-plugins directory
	if _, err := os.Stat(cfg.OcPluginsDir); err == nil {
		if err := discoverPlugins(cfg.OcPluginsDir, state); err != nil {
			state.Warnings = append(state.Warnings, fmt.Sprintf("plugin discovery: %v", err))
		}
	} else {
		state.SkippedSources = append(state.SkippedSources, "oc-plugins dir (not found)")
	}

	// 4. Read bundled instructions from open-chad repo
	instructionsDir := filepath.Join(cfg.OpenChadRepo, "config", "opencode", "instructions")
	if _, err := os.Stat(instructionsDir); err == nil {
		if err := readInstructions(instructionsDir, state); err != nil {
			state.Warnings = append(state.Warnings, fmt.Sprintf("instructions: %v", err))
		}
	}

	// 6. Read bundled skills from open-chad repo
	skillsDir := filepath.Join(cfg.OpenChadRepo, "config", "opencode", "skills")
	if _, err := os.Stat(skillsDir); err == nil {
		if err := readSkills(skillsDir, state); err != nil {
			state.Warnings = append(state.Warnings, fmt.Sprintf("skills: %v", err))
		}
	}

	// 7. Fill MCP server ports from URL when Vision didn't provide one. This
	// must run after both readOpenCodeJSON (URL recorded) and readVisionServers
	// (Port may have been set from Vision YAML).
	for name, mcs := range state.MCPServers {
		if mcs.Port == 0 && mcs.URL != "" {
			if p := portFromURL(mcs.URL); p > 0 {
				mcs.Port = p
				state.MCPServers[name] = mcs
			}
		}
	}

	return state, nil
}

// readOpenCodeJSON parses the live opencode.json for MCP servers, plugins,
// providers, permissions, instructions, and theme.
func readOpenCodeJSON(path string, state *OpenChadState) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// MCP servers — translate legacy `type` values into OCA's schema enum at
	// read time so the emitted stack validates directly.
	if mcpRaw, ok := raw["mcp"].(map[string]any); ok {
		for name, srvRaw := range mcpRaw {
			srv, ok := srvRaw.(map[string]any)
			if !ok {
				continue
			}
			var mcs MCPServerState
			typeRaw, _ := srv["type"].(string)
			urlRaw, _ := srv["url"].(string)
			translated, warn := translateMCPType(typeRaw, urlRaw)
			mcs.Type = translated
			mcs.URL = urlRaw
			if warn != "" {
				state.Warnings = append(state.Warnings, fmt.Sprintf("mcp.%s: %s", name, warn))
			}
			if e, ok := srv["enabled"].(bool); ok {
				mcs.Autostart = e
			}
			state.MCPServers[name] = mcs
		}
	}

	// Plugins — classify each entry by source kind (path / npm-scoped /
	// npm-unscoped / bare name). See classifyPluginEntry.
	if pluginsRaw, ok := raw["plugin"].([]any); ok {
		for _, pRaw := range pluginsRaw {
			entry, ok := pRaw.(string)
			if !ok {
				continue
			}
			ps, err := classifyPluginEntry(entry)
			if err != nil {
				state.Warnings = append(state.Warnings,
					fmt.Sprintf("plugin entry %q: %v; skipped", entry, err))
				continue
			}
			state.Plugins = append(state.Plugins, ps)
		}
	}

	// Instructions
	if instrRaw, ok := raw["instructions"].([]any); ok {
		for _, iRaw := range instrRaw {
			if path, ok := iRaw.(string); ok {
				state.Instructions = append(state.Instructions, path)
			}
		}
	}

	// Theme
	if theme, ok := raw["theme"].(string); ok {
		if state.OpenCode == nil {
			state.OpenCode = &OpenCodeState{}
		}
		state.OpenCode.Theme = theme
	}

	// Providers (complex nested structure)
	if provRaw, ok := raw["provider"].([]any); ok {
		for _, pRaw := range provRaw {
			p, ok := pRaw.(map[string]any)
			if !ok {
				continue
			}
			name, _ := p["name"].(string)
			if name == "" {
				continue
			}
			var ps ProviderState
			ps.Name = name
			if modelsRaw, ok := p["models"].([]any); ok {
				for _, mRaw := range modelsRaw {
					m, ok := mRaw.(map[string]any)
					if !ok {
						continue
					}
					ms := ModelState{Name: fmt.Sprint(m["name"])}
					if limRaw, ok := m["limit"].(map[string]any); ok {
						var ls LimitState
						if v, ok := limRaw["tokens"].(float64); ok {
							ti := int(v)
							ls.Tokens = &ti
						}
						ms.Limit = &ls
					}
					if modsRaw, ok := m["modalities"].([]any); ok {
						for _, modRaw := range modsRaw {
							if mod, ok := modRaw.(string); ok {
								ms.Modalities = append(ms.Modalities, mod)
							}
						}
					}
					ps.Models = append(ps.Models, ms)
				}
			}
			state.Providers[name] = ps
		}
	}

	return nil
}

// readVisionServers parses vision/servers.yaml and enriches MCP server state
// with command/args/port details.
func readVisionServers(path string, state *OpenChadState) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var doc struct {
		Servers map[string]struct {
			Port      int      `yaml:"port"`
			Command   string   `yaml:"command"`
			Args      []string `yaml:"args"`
			Autostart bool     `yaml:"autostart"`
			Source    string   `yaml:"source"`
			Timeout   int      `yaml:"timeout"`
		} `yaml:"servers"`
		SlotGroups map[string]struct {
			Servers   []string `yaml:"servers"`
			GroupPort int      `yaml:"group_port"`
			Template  string   `yaml:"template"`
			BasePort  int      `yaml:"base_port"`
			Count     int      `yaml:"count"`
		} `yaml:"slot_groups"`
	}

	if err := yaml.Unmarshal(data, &doc); err != nil {
		return err
	}

	for name, srv := range doc.Servers {
		mcs := state.MCPServers[name]
		mcs.Port = srv.Port
		mcs.Command = srv.Command
		mcs.Args = srv.Args
		mcs.Autostart = srv.Autostart
		mcs.Source = srv.Source
		mcs.Timeout = srv.Timeout
		state.MCPServers[name] = mcs
	}

	for name, sg := range doc.SlotGroups {
		state.SlotGroups[name] = SlotGroupState{
			Servers:   sg.Servers,
			GroupPort: sg.GroupPort,
			Template:  sg.Template,
			BasePort:  sg.BasePort,
			Count:     sg.Count,
		}
	}

	return nil
}

// discoverPlugins scans the oc-plugins directory for git checkouts and
// extracts source URLs and current refs.
func discoverPlugins(dir string, state *OpenChadState) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		checkout := filepath.Join(dir, entry.Name())
		gitDir := filepath.Join(checkout, ".git")
		if _, err := os.Stat(gitDir); err != nil {
			continue // not a git checkout
		}

		// Find matching plugin in state by checkout path
		found := false
		for i := range state.Plugins {
			if pluginCheckoutMatches(state.Plugins[i].Checkout, entry.Name()) {
				state.Plugins[i].Checkout = checkout
				// Try to read remote URL
				if url, err := readGitRemoteURL(checkout); err == nil {
					state.Plugins[i].Source = url
				}
				// Try to read current ref (HEAD SHA)
				if ref, err := readGitHEAD(checkout); err == nil {
					state.Plugins[i].Ref = ref
				}
				found = true
				break
			}
		}
		if !found {
			// Plugin exists on disk but not in opencode.json — add as orphan
			var ps PluginState
			ps.Name = entry.Name()
			ps.Checkout = checkout
			if url, err := readGitRemoteURL(checkout); err == nil {
				ps.Source = url
			}
			if ref, err := readGitHEAD(checkout); err == nil {
				ps.Ref = ref
			}
			state.Plugins = append(state.Plugins, ps)
		}
	}

	return nil
}

func pluginCheckoutMatches(checkout, name string) bool {
	clean := filepath.Clean(checkout)
	if filepath.Base(clean) == name {
		return true
	}
	return filepath.Base(clean) == "plugin" && filepath.Base(filepath.Dir(clean)) == name
}

func readGitRemoteURL(dir string) (string, error) {
	cmdPath := filepath.Join(dir, ".git", "config")
	data, err := os.ReadFile(cmdPath)
	if err != nil {
		return "", err
	}
	// Simple parse: look for [remote "origin"] → url = ...
	lines := strings.Split(string(data), "\n")
	inOrigin := false
	for _, line := range lines {
		if strings.TrimSpace(line) == `[remote "origin"]` {
			inOrigin = true
			continue
		}
		if inOrigin && strings.HasPrefix(strings.TrimSpace(line), "url = ") {
			return strings.TrimPrefix(strings.TrimSpace(line), "url = "), nil
		}
		if inOrigin && strings.HasPrefix(line, "[") {
			break
		}
	}
	return "", fmt.Errorf("remote origin not found")
}

func readGitHEAD(dir string) (string, error) {
	headPath := filepath.Join(dir, ".git", "HEAD")
	data, err := os.ReadFile(headPath)
	if err != nil {
		return "", err
	}
	ref := strings.TrimSpace(string(data))
	// If HEAD is a symbolic ref, read the actual commit
	if strings.HasPrefix(ref, "ref: ") {
		refFile := strings.TrimPrefix(ref, "ref: ")
		refPath := filepath.Join(dir, ".git", refFile)
		data, err := os.ReadFile(refPath)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(data)), nil
	}
	return ref, nil
}

// readInstructions discovers instruction files in a directory, skipping stale
// instruction names.
func readInstructions(dir string, state *OpenChadState) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if isStaleInstruction(name) {
			state.Warnings = append(state.Warnings, fmt.Sprintf("skipped stale instruction: %s", name))
			continue
		}
		path := filepath.Join(dir, name)
		state.Instructions = append(state.Instructions, path)
	}

	return nil
}

// readSkills discovers skill directories in a directory.
func readSkills(dir string, state *OpenChadState) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if isStaleSkill(name) {
			state.Warnings = append(state.Warnings, fmt.Sprintf("skipped stale skill: %s", name))
			continue
		}
		state.Skills = append(state.Skills, name)
	}

	return nil
}
