# Color Palette Reference

Quick-reference swatches and usage guidelines for the OpenCode Advance palette. For the full brand specification see [`brand.md`](brand.md).

## Base palette (Obsidian / Slate / Graphite)

```
┌─────────────────────────────────────────────────────────────────────────┐
│  Obsidian    #0B0D10   ████  primary background (near-black, cool)     │
│  Slate       #1E2228   ████  elevated surfaces, panels, status bar     │
│  Graphite    #2D3138   ████  borders, dividers, inactive elements      │
│  Graphite+   #3D424B   ████  hover/active on graphite                  │
└─────────────────────────────────────────────────────────────────────────┘
```

## Text palette (Ivory family)

```
┌─────────────────────────────────────────────────────────────────────────┐
│  Ivory       #E8E6E3   ████  primary text (warm near-white)            │
│  Muted Ivory #A8A6A3   ████  secondary text, metadata                  │
│  Dim Ivory   #6A6866   ████  tertiary / disabled text                  │
└─────────────────────────────────────────────────────────────────────────┘
```

## Accent palette (Frosted Indigo)

```
┌─────────────────────────────────────────────────────────────────────────┐
│  Indigo      #6C7AB8   ████  signature accent (frosted muted violet)   │
│  Indigo+     #8B9FE0   ████  accent hover / active / highlight         │
│  Indigo-Glow #B4C4F5   ████  rare: wordmark glow, critical highlights  │
└─────────────────────────────────────────────────────────────────────────┘
```

## Status palette (muted, harmonious with base)

```
┌─────────────────────────────────────────────────────────────────────────┐
│  Success     #7A9B7A   ████  muted sage green                          │
│  Warning     #D4A843   ████  muted amber                               │
│  Error       #D47A7A   ████  muted coral                               │
└─────────────────────────────────────────────────────────────────────────┘
```

## Usage matrix

| Element                         | Color                    | Background  |
| ------------------------------- | ------------------------ | ----------- |
| Terminal background             | Obsidian                 | —           |
| Status bar background           | Slate                    | Obsidian    |
| Panel/box borders               | Graphite                 | Slate       |
| Primary text                    | Ivory                    | Obsidian    |
| Heading text                    | Ivory                    | Obsidian    |
| Metadata (timestamps, IDs)      | Muted Ivory              | Obsidian    |
| Inactive/disabled text          | Dim Ivory                | Obsidian    |
| Active session indicator        | Indigo                   | Slate       |
| Focused input field             | Indigo                   | Slate       |
| Selected menu item              | Indigo+                  | Slate       |
| Wordmark ("OpenCode" text)      | Ivory                    | Obsidian    |
| Wordmark ("ADVANCE" text)       | Indigo (top→Indigo+ grad) | Obsidian    |
| Progress bar fill               | Indigo                   | Graphite    |
| `✓` success marker                | Success                  | Obsidian    |
| `⚠` warning marker                | Warning                  | Obsidian    |
| `✗` error marker                  | Error                    | Obsidian    |
| Key/value separator             | Graphite                 | Obsidian    |
| Prompt `$` / `>`                    | Indigo                   | Obsidian    |
| Cursor block                    | Indigo+                  | Obsidian    |

## Forbidden combinations

| Do not use                                    | Reason                                   |
| --------------------------------------------- | ---------------------------------------- |
| Pure black (`#000000`) background               | Too harsh; use Obsidian                  |
| Pure white (`#FFFFFF`) text                     | Too bright; use Ivory                    |
| Multiple saturated colors at once             | Violates monochrome+accent rule          |
| Rainbow borders, edges, or dividers           | Explicitly removed from "chad" era       |
| Color-cycling or animated color transitions   | Only allowed during boot splash          |
| Red or orange as primary accent               | Indigo is the signature, nothing else    |
| Neon/electric colors                          | Feels cheap; we use muted frosted tones  |
| Gradient backgrounds                          | Flat colors only                         |

## Tmux / shell integration

When exporting these colors for shell scripts or tmux configs:

```bash
# Obsidian / Slate / Graphite / Ivory / Indigo (truecolor)
OCA_COLOR_OBSIDIAN="\e[38;2;11;13;16m"
OCA_COLOR_SLATE="\e[38;2;30;34;40m"
OCA_COLOR_GRAPHITE="\e[38;2;45;49;56m"
OCA_COLOR_IVORY="\e[38;2;232;230;227m"
OCA_COLOR_INDIGO="\e[38;2;108;122;184m"
OCA_COLOR_INDIGO_BRIGHT="\e[38;2;139;159;224m"
OCA_COLOR_INDIGO_GLOW="\e[38;2;180;196;245m"
OCA_COLOR_SUCCESS="\e[38;2;122;155;122m"
OCA_COLOR_WARNING="\e[38;2;212;168;67m"
OCA_COLOR_ERROR="\e[38;2;212;122;122m"
OCA_COLOR_RESET="\e[0m"

# Tmux-format equivalents
OCA_TMUX_OBSIDIAN="#0B0D10"
OCA_TMUX_SLATE="#1E2228"
OCA_TMUX_GRAPHITE="#2D3138"
OCA_TMUX_IVORY="#E8E6E3"
OCA_TMUX_INDIGO="#6C7AB8"
OCA_TMUX_INDIGO_BRIGHT="#8B9FE0"
```

A canonical version of these constants will live at `lib/palette.sh` once Phase 0 implementation begins.
