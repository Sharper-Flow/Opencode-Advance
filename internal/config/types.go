// Package config parses, validates, and resolves the OpenCode Advance
// stack.toml configuration file into typed Go structs.
//
// Phase 1 scope covers [meta] and [mcp.servers.*]. Known-but-unimplemented
// top-level sections (agents, session, discord, skills, formatters, commands,
// opencode) parse into a generic DeferredSections map without error.
// Truly unknown top-level sections produce a validation error.
//
// Phase 3 graduates providers, permissions, watcher, and lsp into typed
// structs with Extra map[string]any passthrough.
//
// See docs/design/phase1-mcp-apply.md for the full type design.
package config

import "fmt"

// Stack is the in-memory representation of a validated stack.toml file.
// Fields are populated by Parse, with validation run by Validate and
// variable expansion by Resolve. Use Load to run the full pipeline.
type Stack struct {
	Meta Meta       `toml:"meta"`
	MCP  MCPSection `toml:"mcp"`

	// Phase 2 typed sections
	Plugins      PluginsSection      `toml:"plugins"`
	Instructions InstructionsSection `toml:"instructions"`
	Temporal     *TemporalSection    `toml:"temporal"`

	// Phase 3 typed sections
	Providers   ProvidersSection   `toml:"providers"`
	Permissions PermissionsSection `toml:"permissions"`
	Watcher     WatcherSection     `toml:"watcher"`
	LSP         LSPSection         `toml:"lsp"`

	// Phase 3.5 typed sections
	Skills     SkillsSection     `toml:"skills"`
	Formatters FormattersSection `toml:"formatters"`
	Commands   CommandsSection   `toml:"commands"`
	OpenCode   OpenCodeSection   `toml:"opencode"`

	// DeferredSections holds known-but-unimplemented top-level sections
	// verbatim so that a complete stack.toml (including future-phase
	// sections) round-trips through Phase 1 without errors. Entries are
	// not rendered by Phase 1 target writers.
	DeferredSections map[string]any `toml:"-"`

	// Warnings are non-fatal issues surfaced during Load (e.g. env_file
	// path does not exist). Apply and Doctor surface these to the user.
	Warnings []Warning `toml:"-"`
}

// Meta is the [meta] table.
type Meta struct {
	Version     string `toml:"version"`
	Name        string `toml:"name,omitempty"`
	Description string `toml:"description,omitempty"`
}

// MCPSection is the [mcp] table. It currently only carries Servers, but
// is a table for forward-compat with future MCP-wide settings.
type MCPSection struct {
	Servers map[string]Server `toml:"servers"`
}

// Server mirrors Vision's internal/config.ServerConfig field-for-field
// plus OCA-only additions (Required is now in Vision post-V3; Source
// and Description are in Vision post-V2; EnvFile and Type="daemon" are
// OCA-only and stripped at render time when appropriate).
//
// Unknown Vision-side fields under [mcp.servers.<name>] land in
// ExtraFields as raw map entries so OCA passes them through to
// servers.yaml without interpretation (forward-compat with Vision
// schema additions).
type Server struct {
	// Core transport (Vision)
	Port      int               `toml:"port" yaml:"port"`
	Type      string            `toml:"type,omitempty" yaml:"-"`                        // OCA "daemon"|"stdio"|"http"|"sse"
	Transport string            `toml:"transport,omitempty" yaml:"transport,omitempty"` // Vision passthrough alias
	Command   string            `toml:"command,omitempty" yaml:"command,omitempty"`
	Args      []string          `toml:"args,omitempty" yaml:"args,omitempty"`
	Env       map[string]string `toml:"env,omitempty" yaml:"env,omitempty"`
	URL       string            `toml:"url,omitempty" yaml:"url,omitempty"`
	Headers   map[string]string `toml:"headers,omitempty" yaml:"headers,omitempty"`

	// Lifecycle (Vision)
	Autostart           bool   `toml:"autostart,omitempty" yaml:"autostart,omitempty"`
	RestartPolicy       string `toml:"restart_policy,omitempty" yaml:"restart_policy,omitempty"`
	MaxRestarts         int    `toml:"max_restarts,omitempty" yaml:"max_restarts,omitempty"`
	Stateful            bool   `toml:"stateful,omitempty" yaml:"stateful,omitempty"`
	AvailabilityProfile string `toml:"availability_profile,omitempty" yaml:"availability_profile,omitempty"`
	SessionTimeout      string `toml:"session_timeout,omitempty" yaml:"session_timeout,omitempty"`
	MaxSessions         int    `toml:"max_sessions,omitempty" yaml:"max_sessions,omitempty"`
	SessionTTL          string `toml:"session_ttl,omitempty" yaml:"session_ttl,omitempty"`
	HealthCheckInterval string `toml:"health_check_interval,omitempty" yaml:"health_check_interval,omitempty"`
	RequestTimeout      string `toml:"request_timeout,omitempty" yaml:"request_timeout,omitempty"`
	Timeout             int    `toml:"timeout,omitempty" yaml:"-"` // OCA-side alias (ms) → Vision request_timeout

	// Resilience (Vision)
	Retry          *RetryConfig    `toml:"retry,omitempty" yaml:"retry,omitempty"`
	CircuitBreaker *CircuitBreaker `toml:"circuit_breaker,omitempty" yaml:"circuit_breaker,omitempty"`

	// Sharing (Vision)
	SharedReadOnlyTools   []string `toml:"shared_read_only_tools,omitempty" yaml:"shared_read_only_tools,omitempty"`
	SharedResultCacheTTL  string   `toml:"shared_result_cache_ttl,omitempty" yaml:"shared_result_cache_ttl,omitempty"`
	SharedResultCacheSize int      `toml:"shared_result_cache_size,omitempty" yaml:"shared_result_cache_size,omitempty"`
	MaxInFlightRequests   int      `toml:"max_in_flight_requests,omitempty" yaml:"max_in_flight_requests,omitempty"`

	// OCA-respected / Vision accepts after V2+V3
	Required    bool   `toml:"required,omitempty" yaml:"required,omitempty"`
	Source      string `toml:"source,omitempty" yaml:"source,omitempty"`
	Description string `toml:"description,omitempty" yaml:"description,omitempty"`

	// Enabled is tri-state (pointer) so "enabled=false" is distinguishable
	// from "field absent (default true)" at render time.
	Enabled *bool `toml:"enabled,omitempty" yaml:"enabled,omitempty"`

	// OCA-side only — stripped at render time.
	EnvFile string `toml:"env_file,omitempty" yaml:"-"`
}

// RetryConfig configures retryable-failure behavior (Vision passthrough).
// Named RetryConfig (not Retry) so usage at call sites reads as a noun
// describing configuration data, not a verb that could be misread as an
// action. Field name on Server remains Retry; TOML/YAML keys remain "retry".
type RetryConfig struct {
	MaxAttempts     int      `toml:"max_attempts,omitempty" yaml:"max_attempts,omitempty"`
	InitialDelay    string   `toml:"initial_delay,omitempty" yaml:"initial_delay,omitempty"`
	MaxDelay        string   `toml:"max_delay,omitempty" yaml:"max_delay,omitempty"`
	RetryableErrors []string `toml:"retryable_errors,omitempty" yaml:"retryable_errors,omitempty"`
}

// CircuitBreaker configures fast-fail after repeated failures (Vision passthrough).
type CircuitBreaker struct {
	FailureThreshold int    `toml:"failure_threshold,omitempty" yaml:"failure_threshold,omitempty"`
	RecoveryTimeout  string `toml:"recovery_timeout,omitempty" yaml:"recovery_timeout,omitempty"`
}

// ProvidesCategory enumerates asset categories a plugin can declare it
// owns. When a plugin lists a category in provides, oca apply skips
// rendering or copying assets in that category from its own assets/ dir.
type ProvidesCategory string

const (
	ProvidesCommands     ProvidesCategory = "adv-commands"
	ProvidesAgents       ProvidesCategory = "adv-agents"
	ProvidesSkills       ProvidesCategory = "adv-skills"
	ProvidesOverlays     ProvidesCategory = "adv-overlays"
	ProvidesInstructions ProvidesCategory = "adv-instructions"
	ProvidesTemporal     ProvidesCategory = "adv-temporal" // Temporal config owned by OCA (Phase 5+)
)

// IsValid returns true if the category is a known provides value.
func (p ProvidesCategory) IsValid() bool {
	switch p {
	case ProvidesCommands, ProvidesAgents, ProvidesSkills,
		ProvidesOverlays, ProvidesInstructions, ProvidesTemporal:
		return true
	}
	return false
}

// Plugin describes a plugin declared in [plugins.<name>]. Plugins are
// either git-sourced (with checkout + optional subdir + build + sync)
// or npm-sourced (with a "npm:pkg@version" source string).
type Plugin struct {
	Source       string             `toml:"source"`             // git URL or "npm:pkg@version"
	Ref          string             `toml:"ref,omitempty"`      // branch/tag/SHA; default "trunk"
	Checkout     string             `toml:"checkout,omitempty"` // local clone path
	Subdir       string             `toml:"subdir,omitempty"`   // subdir within checkout
	Build        []string           `toml:"build,omitempty"`    // build commands
	Path         string             `toml:"path,omitempty"`     // load path, supports {checkout}/{subdir}
	Sync         string             `toml:"sync,omitempty"`     // post-build sync script
	Provides     []ProvidesCategory `toml:"provides,omitempty"`
	Instructions []string           `toml:"instructions,omitempty"` // instruction files
	Enabled      *bool              `toml:"enabled,omitempty"`      // default true
}

// IsGitSource returns true for git URL sources (not npm: prefixed).
func (p Plugin) IsGitSource() bool {
	return len(p.Source) >= 4 && p.Source[:4] != "npm:"
}

// IsNPMSource returns true for npm:-prefixed sources.
func (p Plugin) IsNPMSource() bool {
	return len(p.Source) >= 4 && p.Source[:4] == "npm:"
}

// IsEnabled returns the effective enabled state. Nil means true.
func (p Plugin) IsEnabled() bool {
	if p.Enabled == nil {
		return true
	}
	return *p.Enabled
}

// PluginsSection is the [plugins] table — a map of plugin name to Plugin.
type PluginsSection map[string]Plugin

// InstructionsSection is the [instructions] table.
type InstructionsSection struct {
	Order []string `toml:"order,omitempty"`
}

// TemporalSection is the [temporal] table. Phase 5 expands this from a
// reserved stub to a fully-validated config section with defaults.
// Validation (validateTemporal) fills defaults for Address and Namespace
// when enabled.
type TemporalSection struct {
	Enabled     *bool  `toml:"enabled,omitempty"`      // default true if absent
	Address     string `toml:"address,omitempty"`      // default "127.0.0.1:7233"
	Namespace   string `toml:"namespace,omitempty"`    // default "default"
	AllowRemote *bool  `toml:"allow_remote,omitempty"` // must be true for non-loopback
	NodePath    string `toml:"node_path,omitempty"`    // optional override for node binary
}

// IsEnabled returns the effective enabled state. Nil means true.
func (t *TemporalSection) IsEnabled() bool {
	if t == nil {
		return false
	}
	if t.Enabled == nil {
		return true
	}
	return *t.Enabled
}

// ---------------------------------------------------------------------------
// Phase 3 — graduated typed sections
// ---------------------------------------------------------------------------

// ProvidersSection is the [providers] table — a map of provider name to Provider.
type ProvidersSection map[string]Provider

// Provider describes a model provider declared in [providers.<name>].
type Provider struct {
	Models   map[string]ProviderModel `toml:"models,omitempty"`
	Options  map[string]any           `toml:"options,omitempty"`
	Variants map[string]any           `toml:"variants,omitempty"`
	Extra    map[string]any           `toml:"-"`
}

// UnmarshalTOML implements custom decoding to capture extra unknown fields
// (fields not in the Provider struct) into the Extra map, including at
// nested levels (per-model Extra via ProviderModel.UnmarshalTOML).
func (p *Provider) UnmarshalTOML(value any) error {
	m, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("expected table for provider, got %T", value)
	}

	// Manually decode known fields; unknown fields accumulate in extra.
	extra := make(map[string]any)
	for k, v := range m {
		switch k {
		case "models":
			models, err := decodeProviderModels(v)
			if err != nil {
				return fmt.Errorf("models: %w", err)
			}
			p.Models = models
		case "options":
			opts, err := decodeIntoMap(v)
			if err != nil {
				return fmt.Errorf("options: %w", err)
			}
			p.Options = opts
		case "variants":
			variants, err := decodeIntoMap(v)
			if err != nil {
				return fmt.Errorf("variants: %w", err)
			}
			p.Variants = variants
		default:
			extra[k] = v
		}
	}
	if len(extra) > 0 {
		p.Extra = extra
	}
	return nil
}

// decodeProviderModels decodes a raw TOML value into a map of ProviderModel.
// Each model value goes through ProviderModel.UnmarshalTOML to capture its
// own Extra fields.
func decodeProviderModels(raw any) (map[string]ProviderModel, error) {
	m, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected table, got %T", raw)
	}
	result := make(map[string]ProviderModel, len(m))
	for name, modelRaw := range m {
		model := ProviderModel{}
		if err := model.UnmarshalTOML(name, modelRaw); err != nil {
			return nil, fmt.Errorf("[models.%s]: %w", name, err)
		}
		result[name] = model
	}
	return result, nil
}

// ProviderModel describes a single model under a provider.
// Shape translation to opencode.json happens in the render layer.
type ProviderModel struct {
	Name     string         `toml:"name,omitempty"`
	Context  int            `toml:"context,omitempty"`
	Output   int            `toml:"output,omitempty"`
	Inputs   []string       `toml:"inputs,omitempty"`
	Outputs  []string       `toml:"outputs,omitempty"`
	Options  map[string]any `toml:"options,omitempty"`
	Variants map[string]any `toml:"variants,omitempty"`
	Extra    map[string]any `toml:"-"`
}

// modelName is passed explicitly because ProviderModel is identified by its
// map key in ProvidersSection, not by an internal name field for TOML decode.
func (m *ProviderModel) UnmarshalTOML(modelName string, value any) error {
	// ProviderModel is identified by its map key; the "name" field inside is
	// the provider-assigned display name, not the lookup key.
	_ = modelName // explicit signature; name used in error messages above
	pm, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("expected table, got %T", value)
	}

	extra := make(map[string]any)
	for k, v := range pm {
		switch k {
		case "name":
			if s, ok := v.(string); ok {
				m.Name = s
			}
		case "context":
			if i, ok := tomlInt(v); ok {
				m.Context = i
			}
		case "output":
			if i, ok := tomlInt(v); ok {
				m.Output = i
			}
		case "inputs":
			if ss, ok := v.([]any); ok {
				m.Inputs = toStringSlice(ss)
			}
		case "outputs":
			if ss, ok := v.([]any); ok {
				m.Outputs = toStringSlice(ss)
			}
		case "options":
			opts, err := decodeIntoMap(v)
			if err != nil {
				return fmt.Errorf("options: %w", err)
			}
			m.Options = opts
		case "variants":
			variants, err := decodeIntoMap(v)
			if err != nil {
				return fmt.Errorf("variants: %w", err)
			}
			m.Variants = variants
		default:
			extra[k] = v
		}
	}
	if len(extra) > 0 {
		m.Extra = extra
	}
	return nil
}

// PermissionsSection is the [permissions] table.
// Shape translation (default → .permission["*"]) happens in the render layer.
type PermissionsSection struct {
	Default           string            `toml:"default,omitempty"`
	DoomLoop          string            `toml:"doom_loop,omitempty"`
	ExternalDirectory map[string]string `toml:"external_directory,omitempty"`
	Bash              map[string]string `toml:"bash,omitempty"`
	Extra             map[string]any    `toml:"-"`
}

// WatcherSection is the [watcher] table.
type WatcherSection struct {
	Ignore []string       `toml:"ignore,omitempty"`
	Extra  map[string]any `toml:"-"`
}

// LSPSection is the [lsp] table — a map of LSP server name to LSP.
type LSPSection map[string]LSP

// LSP describes an LSP server declared in [lsp.<name>].
type LSP struct {
	Command    []string       `toml:"command,omitempty"`
	Extensions []string       `toml:"extensions,omitempty"`
	Disabled   bool           `toml:"disabled,omitempty"`
	Extra      map[string]any `toml:"-"`
}

// UnmarshalTOML implements custom decoding to capture extra unknown fields
// into the Extra map.
func (l *LSP) UnmarshalTOML(value any) error {
	m, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("expected table for LSP, got %T", value)
	}

	extra := make(map[string]any)
	for k, v := range m {
		switch k {
		case "command":
			if ss, ok := v.([]any); ok {
				l.Command = toStringSlice(ss)
			}
		case "extensions":
			if ss, ok := v.([]any); ok {
				l.Extensions = toStringSlice(ss)
			}
		case "disabled":
			if b, ok := v.(bool); ok {
				l.Disabled = b
			}
		default:
			extra[k] = v
		}
	}
	if len(extra) > 0 {
		l.Extra = extra
	}
	return nil
}

// ---------------------------------------------------------------------------
// Phase 3.5 — skills, formatters, commands, opencode toggles
// ---------------------------------------------------------------------------

// SkillsSection is the [skills] table.
type SkillsSection struct {
	Order []string `toml:"order,omitempty"`
}

// FormattersSection is the [formatters] table — a map of formatter name to Formatter.
type FormattersSection map[string]Formatter

// Formatter describes a code formatter declared in [formatters.<name>].
type Formatter struct {
	Command     []string          `toml:"command,omitempty"`
	Extensions  []string          `toml:"extensions,omitempty"`
	Disabled    bool              `toml:"disabled,omitempty"`
	Environment map[string]string `toml:"environment,omitempty"`
	Extra       map[string]any    `toml:"-"`
}

// UnmarshalTOML implements custom decoding to capture extra unknown fields
// into the Extra map.
func (f *Formatter) UnmarshalTOML(value any) error {
	m, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("expected table for formatter, got %T", value)
	}

	extra := make(map[string]any)
	for k, v := range m {
		switch k {
		case "command":
			if ss, ok := v.([]any); ok {
				f.Command = toStringSlice(ss)
			}
		case "extensions":
			if ss, ok := v.([]any); ok {
				f.Extensions = toStringSlice(ss)
			}
		case "disabled":
			if b, ok := v.(bool); ok {
				f.Disabled = b
			}
		case "environment":
			env, err := decodeStringMap(v)
			if err != nil {
				return fmt.Errorf("environment: %w", err)
			}
			f.Environment = env
		default:
			extra[k] = v
		}
	}
	if len(extra) > 0 {
		f.Extra = extra
	}
	return nil
}

// CommandsSection is the [commands] table — a map of command name to Command.
type CommandsSection map[string]Command

// Command describes a custom slash command declared in [commands.<name>].
type Command struct {
	Description string         `toml:"description"`
	Template    string         `toml:"template"`
	Agent       string         `toml:"agent,omitempty"`
	Model       string         `toml:"model,omitempty"`
	Subtask     *bool          `toml:"subtask,omitempty"`
	Extra       map[string]any `toml:"-"`
}

// UnmarshalTOML implements custom decoding to capture extra unknown fields
// into the Extra map.
func (c *Command) UnmarshalTOML(value any) error {
	m, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("expected table for command, got %T", value)
	}

	extra := make(map[string]any)
	for k, v := range m {
		switch k {
		case "description":
			if s, ok := v.(string); ok {
				c.Description = s
			}
		case "template":
			if s, ok := v.(string); ok {
				c.Template = s
			}
		case "agent":
			if s, ok := v.(string); ok {
				c.Agent = s
			}
		case "model":
			if s, ok := v.(string); ok {
				c.Model = s
			}
		case "subtask":
			if b, ok := v.(bool); ok {
				c.Subtask = &b
			}
		default:
			extra[k] = v
		}
	}
	if len(extra) > 0 {
		c.Extra = extra
	}
	return nil
}

// OpenCodeSection is the [opencode] table — top-level OpenCode behavior toggles.
type OpenCodeSection struct {
	Theme             string               `toml:"theme,omitempty"`
	DefaultAgent      string               `toml:"default_agent,omitempty"`
	Share             string               `toml:"share,omitempty"`
	Snapshot          *bool                `toml:"snapshot,omitempty"`
	Autoupdate        *AutoupdateValue     `toml:"autoupdate,omitempty"`
	Compaction        *CompactionSection   `toml:"compaction,omitempty"`
	DisabledProviders *ProviderListSection `toml:"disabled_providers,omitempty"`
	EnabledProviders  *ProviderListSection `toml:"enabled_providers,omitempty"`
	Extra             map[string]any       `toml:"-"`
}

// UnmarshalTOML implements custom decoding to capture extra unknown fields
// into the Extra map and handle the autoupdate bool/string union.
func (o *OpenCodeSection) UnmarshalTOML(value any) error {
	m, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("expected table for opencode, got %T", value)
	}

	extra := make(map[string]any)
	for k, v := range m {
		switch k {
		case "theme":
			if s, ok := v.(string); ok {
				o.Theme = s
			}
		case "default_agent":
			if s, ok := v.(string); ok {
				o.DefaultAgent = s
			}
		case "share":
			if s, ok := v.(string); ok {
				o.Share = s
			}
		case "snapshot":
			if b, ok := v.(bool); ok {
				o.Snapshot = &b
			}
		case "autoupdate":
			o.Autoupdate = parseAutoupdateValue(v)
		case "compaction":
			cs := &CompactionSection{}
			if err := decodeIntoWithExtra(v, cs); err != nil {
				return fmt.Errorf("compaction: %w", err)
			}
			o.Compaction = cs
		case "disabled_providers":
			pl := &ProviderListSection{}
			if err := decodeInto(v, pl); err != nil {
				return fmt.Errorf("disabled_providers: %w", err)
			}
			o.DisabledProviders = pl
		case "enabled_providers":
			pl := &ProviderListSection{}
			if err := decodeInto(v, pl); err != nil {
				return fmt.Errorf("enabled_providers: %w", err)
			}
			o.EnabledProviders = pl
		default:
			extra[k] = v
		}
	}
	if len(extra) > 0 {
		o.Extra = extra
	}
	return nil
}

// AutoupdateValue represents the autoupdate field which can be a bool
// (true/false) or a string ("notify"). Exactly one of Bool or Str is set.
type AutoupdateValue struct {
	Bool *bool
	Str  *string
}

// parseAutoupdateValue decodes the raw TOML value into an AutoupdateValue.
func parseAutoupdateValue(v any) *AutoupdateValue {
	switch val := v.(type) {
	case bool:
		return &AutoupdateValue{Bool: &val}
	case string:
		return &AutoupdateValue{Str: &val}
	}
	return nil
}

// CompactionSection is the [opencode.compaction] passthrough table.
type CompactionSection struct {
	Auto     *bool          `toml:"auto,omitempty"`
	Prune    *bool          `toml:"prune,omitempty"`
	Reserved int            `toml:"reserved,omitempty"`
	Extra    map[string]any `toml:"-"`
}

// ProviderListSection is used for [opencode.disabled_providers] and
// [opencode.enabled_providers] — each has a `list` field.
type ProviderListSection struct {
	List []string `toml:"list,omitempty"`
}

// decodeStringMap converts a raw TOML value to map[string]string.
func decodeStringMap(raw any) (map[string]string, error) {
	m, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected table, got %T", raw)
	}
	result := make(map[string]string, len(m))
	for k, v := range m {
		if s, ok := v.(string); ok {
			result[k] = s
		}
	}
	return result, nil
}

// decodeIntoMap converts a raw TOML value to map[string]any for storage
// in typed Extra/passthrough fields.
func decodeIntoMap(raw any) (map[string]any, error) {
	m, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected table, got %T", raw)
	}
	return m, nil
}

// toStringSlice converts a []any (from TOML decode) to []string,
// handling the common case of []any{string, ...}.
func toStringSlice(a []any) []string {
	if a == nil {
		return nil
	}
	out := make([]string, 0, len(a))
	for _, v := range a {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// tomlInt extracts an int64 from a TOML value (which decodes to int64
// for integer types, float64 for floats, etc.).
func tomlInt(v any) (int, bool) {
	switch n := v.(type) {
	case int64:
		return int(n), true
	case int:
		return n, true
	case float64:
		return int(n), true
	}
	return 0, false
}

// Warning carries a non-fatal diagnostic from Load. Surfaced by apply
// and doctor commands but does not abort operations.
type Warning struct {
	Path    string // e.g. "mcp.servers.kagi.env_file"
	Message string
	Hint    string // optional remediation suggestion
}

// IsEnabled returns the effective enabled state for a server. Absent
// (nil) is treated as true; explicit false disables.
func (s Server) IsEnabled() bool {
	if s.Enabled == nil {
		return true
	}
	return *s.Enabled
}
