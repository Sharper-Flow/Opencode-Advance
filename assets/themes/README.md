# Themes

Theme assets for OpenCode Advance. Discovered by `oca theme list` (any `*.json` here).

## Inventory

| File                          | Purpose                                                                  |
| ----------------------------- | ------------------------------------------------------------------------ |
| `obsidian.json`               | Default OCA UI theme — indigo identity tuned to ayu-dark contrast feel   |
| `obsidian-mono.json`          | Neutral-gray variant — ayu-dark bg/panel grays with indigo accents       |
| `obsidian-classic.json`       | Pre-retune snapshot of `obsidian.json` (deeper bg, bigger panel jump)    |
| `ayu-dark.json`               | Vendored reference of upstream ayu-dark (the contrast jam target)        |
| `obsidian.tmux.conf`          | Tmux status bar + window theme (default)                                 |

Planned additions:

| `obsidian-light.tmux.conf`    | Light variant (future)                                                   |

## Background contrast ladder

All `obsidian-*` themes target the same bg → dialogue-panel relationship as ayu-dark: ~11% lightness base with a subtle (~+12 RGB) step to the panel. Differences are tint and accent only.

| Theme              | `background` | `backgroundPanel` | Identity                |
| ------------------ | ------------ | ----------------- | ----------------------- |
| `ayu-dark`         | `#1c1c1c`    | `#282828`         | Neutral, ayu accents    |
| `obsidian`         | `#1A1C20`    | `#262932`         | Subtle indigo lift      |
| `obsidian-mono`    | `#1c1c1c`    | `#282828`         | Neutral, indigo accents |
| `obsidian-classic` | `#0B0D10`    | `#1E2228`         | Deeper, bigger jump     |

## Obsidian theme spec

See [`../../docs/design/theme.md`](../../docs/design/theme.md) for the canonical Obsidian theme specification (palette, tmux row layout, CLI output conventions, boot splash behavior).

## Status

Phase 4 foundation populated `obsidian.json` and `obsidian.tmux.conf`. Light tmux variant deferred. `obsidian-mono`, `obsidian-classic`, and vendored `ayu-dark` added as switchable variants.
