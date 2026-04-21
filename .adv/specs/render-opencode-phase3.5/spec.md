# Capability: render-opencode-phase3.5

Phase 3.5 capability spec for skills, commands, formatters, and OpenCode toggles rendering.

## Requirements

- `rq-phase3.5-skills-copy01` — OCA copies OCA-owned skill directories from `assets/skills/` to `~/.config/opencode/skills/` respecting `[skills].order`. When order is omitted, all non-reserved OCA-owned skills are copied. File writes are atomic (WriteAtomic) and idempotent (noop when bytes match).
- `rq-phase3.5-skills-ownership01` — Skills with `adv-*` namespace are reserved for the Advance plugin. Config validation rejects `adv-*` entries in `[skills].order`. OCA never copies or creates `adv-*` directories in the target. Advance-owned `adv-*` directories already in the target are ignored by health checks.
- `rq-phase3.5-commands-render01` — OCA renders `[commands.*]` into `opencode.json` `.command` section, translating each command entry (description, template, agent) and preserving Extra map fields for forward compatibility.
- `rq-phase3.5-formatters-render01` — OCA renders `[formatters.*]` into `opencode.json` `.formatter` section, translating command ([]string → []any), extensions, disabled flag, environment map, and preserving Extra fields.
- `rq-phase3.5-toggles-render01` — OCA renders `[opencode]` toggles into `opencode.json` root-level keys: theme, default_agent, share, snapshot, autoupdate (bool or string union), plus nested compaction, disabled_providers, and enabled_providers sections.
- `rq-phase3.5-doctor-skills01` — `oca doctor --scope skills` checks: declared skills have source asset directories, reserved namespace enforcement, declared skills present in target dir, extra OCA-owned skill dirs produce warnings, and `adv-*` directories in target are ignored.
- `rq-phase3.5-apply-all01` — `oca apply` without `--target` includes skills, commands, formatters, and toggles in the composed apply plan after Phase 3 targets. Per-target `--target skills|commands|formatters|toggles` applies each independently with parity to composed output.
