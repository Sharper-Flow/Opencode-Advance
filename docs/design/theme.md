# Obsidian Theme Specification

The default OpenCode Advance theme is **Obsidian** — a cohesive dark theme that applies the OCA brand palette to current primary client surfaces and tmux/session surfaces:

1. The current OpenCode chat/input UI target (`~/.config/opencode/themes/obsidian.json`)
2. The tmux status bar and window decorations (`~/dev/opencodeadvance/lib/obsidian.tmux.conf`)
3. CLI output from `oca` itself
4. The boot splash and doctor output

This document specifies each of those surfaces. The canonical implementation lives in `assets/themes/` and is referenced by templates in `templates/`.

## Philosophy

- **Monochrome-plus-accent.** The base is Obsidian/Slate/Graphite/Ivory. Indigo is the only saturated color in normal use. Status colors (success/warning/error) only appear where semantically meaningful.
- **No gradients.** Flat colors. The only exception is the wordmark during boot splash, which may transition from Indigo to Indigo+ as a subtle frosted glass effect.
- **Readable before beautiful.** Contrast ratios meet WCAG AA for text. The theme never sacrifices readability for aesthetics.
- **Calm at rest, focused on action.** Idle UI is quiet. Active elements stand out through the Indigo accent, not through color noise.

## Current primary client theme (`obsidian.json`)

Current v1 planning assumes OpenCode uses a JSON theme file referenced in `opencode.json` via `"theme": "obsidian"`. The file will live at `assets/themes/obsidian.json` in this repo and be installed to `~/.config/opencode/themes/obsidian.json` by `oca apply`.

Canonical structure (to be finalized in Phase 4):

```json
{
  "name": "obsidian",
  "description": "OpenCode Advance signature theme — obsidian base with frosted indigo accent",
  "author": "Sharper Flow",
  "colors": {
    "background": "#0B0D10",
    "foreground": "#E8E6E3",
    "cursor": "#8B9FE0",
    "selection": "#2D3138",
    "selectionForeground": "#E8E6E3",
    "border": "#2D3138",
    "panelBackground": "#1E2228",
    "panelForeground": "#E8E6E3",
    "mutedForeground": "#A8A6A3",
    "dimForeground": "#6A6866",
    "accent": "#6C7AB8",
    "accentBright": "#8B9FE0",
    "success": "#7A9B7A",
    "warning": "#D4A843",
    "error": "#D47A7A"
  },
  "syntax": {
    "comment": "#6A6866",
    "string": "#9FC19F",
    "keyword": "#8B9FE0",
    "function": "#E8E6E3",
    "variable": "#A8A6A3",
    "number": "#D4A843",
    "type": "#B4C4F5",
    "constant": "#8B9FE0",
    "operator": "#A8A6A3",
    "delimiter": "#6A6866"
  }
}
```

The exact schema OpenCode expects will be verified during Phase 4 research before the file is generated. If OpenCode's theme JSON format differs from this draft, the draft is updated to match.

## Tmux theme (`obsidian.tmux.conf`)

A clean two-row status bar, inspired in spirit by the open-chad layout but without synthwave edges, color-cycling, or per-session palette randomization.

### Row layout

```
Row 0 (top):
  left       current session title + ADV change indicator (if active)
  right      repo / branch / worktree marker

Row 1 (bottom):
  left       current window name
  right      metrics (CPU% RAM% LOAD) + LLM fuel gauges (if enabled) + clock
```

### Colors

| Element                      | Color       | Notes                                                  |
| ---------------------------- | ----------- | ------------------------------------------------------ |
| Status bar background        | Slate       | Both rows                                              |
| Status bar foreground        | Ivory       | Default text                                           |
| Inactive window name         | Muted Ivory | Dimmed                                                 |
| Active window name           | Indigo      | Bold, no background change                             |
| Session title                | Ivory       | Left of row 0                                          |
| ADV change indicator         | Indigo      | When an ADV change is active; uses gate progress glyph |
| Repo / branch                | Muted Ivory | Right of row 0                                         |
| Worktree marker              | Indigo+     | Only when current pane is in a worktree                |
| CPU/RAM/LOAD metrics         | Muted Ivory | Not color-coded                                        |
| LLM fuel gauge ≥50%          | Success     |                                                        |
| LLM fuel gauge 20–49%        | Warning     |                                                        |
| LLM fuel gauge <20%          | Error       |                                                        |
| Clock                        | Muted Ivory |                                                        |
| Pane border (inactive)       | Graphite    |                                                        |
| Pane border (active)         | Indigo      | Subtle focus indicator                                 |
| Message text (tmux messages) | Ivory       |                                                        |
| Message background           | Slate       |                                                        |

### No more

- No per-session randomized border colors (removed from open-chad era)
- No synthwave edges / block glyphs in corners
- No color-cycling animations
- No agent palette (blue/yellow/pink/green) in the status bar
- No retro-styled separators between sections

### Section separators

Use thin Graphite vertical bars as dividers where sections meet: `│`. No angular "powerline" arrows. No colored fills.

## CLI output styling

When `oca` commands produce output, follow these conventions:

```
oca apply
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
 rendering stack.toml
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  ✓  MCP servers            9 declared, 9 rendered
  ✓  Plugins                6 declared, 6 resolved
  ✓  Instructions          11 declared, 11 resolved
  ✓  Providers              3 declared, 3 rendered
  ✓  Agents                 6 mapped
  ✓  Permissions            rendered
  ⚠  svelte-mcp             port 6278 not responding
  ✓  opencode.json          patched (backup: .bak.1712345678)
  ✓  vision/servers.yaml    rendered

  1 warning, 0 errors. apply complete.
```

Rules:

- Horizontal rules use `━` (box-drawing heavy horizontal), rendered in Graphite
- Section headers have 1 space leading, sentence case, rendered in Ivory
- Each status line has 2-space indent
- Status glyph: `✓` (Success), `⚠` (Warning), `✗` (Error) — colored
- Status glyph followed by 2 spaces, then the label (Ivory), then additional context (Muted Ivory)
- Columns are aligned by left-padding, not by tabs
- Summary footer is 1 blank line after the last item, then `N warnings, M errors. <verb> <past-tense>.`

## Doctor output styling

`oca doctor` uses the same conventions as `oca apply` but adds per-check explanations on failure:

```
oca doctor
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
 OpenCode Advance health check
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

 Vision daemon
  ✓  binary on PATH         /home/jrede/.local/bin/vision
  ✓  daemon running         PID 12345
  ✓  config readable        ~/.config/vision/servers.yaml

 MCP servers (9 declared)
  ✓  vision                 port 6275  →  200 OK   (2ms)
  ✓  context7               port 6276  →  200 OK   (4ms)
  ⚠  svelte-mcp             port 6278  →  refused  (connection failed)
  ✓  kagi                   port 6279  →  200 OK   (8ms)
  ...

 Plugins (6 declared)
  ✓  advance                ~/dev/oc-plugins/advance  (built)
  ...

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
 summary: 1 warning, 0 errors
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

 Warning: svelte-mcp (port 6278) is declared in stack.toml but not
 responding. This MCP server is autostart=true; Vision should have
 started it. Check: vision daemon logs | grep svelte-mcp
```

## Boot splash

The boot splash renders the full wordmark once per session on session creation. Opt-out via `OCA_BOOT_SPLASH=0` or `oca session new --no-splash`.

Sequence:

1. Clear screen
2. Brief pause (100ms)
3. Render "OpenCode" row in Ivory (instant)
4. 150ms pause
5. Render "ADVANCE" row with slant, in Indigo, with a brief (200ms) pulse of Indigo-Glow on the outer edges
6. Fade to stable Indigo
7. Print compact version line below: `v1.0.0 · ~/dev/opencodeadvance`
8. 500ms hold
9. Clear and hand off to tmux session

Total elapsed: roughly 1 second. No sound effects. No typewriter text. Calm and confident.

When `OCA_BOOT_SPLASH=0`, skip the entire sequence and drop directly into the session.

## Future variants

Not in v1.0 scope, but planned for later versions:

- **Obsidian Light** — inverted palette for users who prefer light themes (ivory background, obsidian text, same indigo accent)
- **High Contrast** — WCAG AAA variant for accessibility
- **Colorblind-safe** — swap Success/Warning/Error for colorblind-distinguishable alternatives

These live in `assets/themes/` as separate `.json` and `.tmux.conf` files and are selectable via `oca theme set <name>`.
