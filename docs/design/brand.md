# Brand Identity

OpenCode Advance has a deliberate, consistent visual identity across its CLI output, documentation, status bar, boot splash, and terminal theming. This document is the source of truth for all brand elements.

## Name

| Context                     | Form                           |
| --------------------------- | ------------------------------ |
| Full product name           | **OpenCode Advance**               |
| Short name                  | **OCA**                            |
| CLI command (primary)       | `oca`                            |
| CLI command (full form)     | `opencode-advance`               |
| Repo name                   | `Sharper-Flow/Opencode-Advance`  |
| Local directory convention  | `~/dev/opencodeadvance`          |
| Display name in status bars | `OCA` or `oca`                     |
| Environment variable prefix | `OCA_*`                          |
| Cache directory name        | `opencode-advance`               |
| Shell profile marker        | `OPENCODE-ADVANCE BEGIN/END`     |
| Tmux theme marker           | `OPENCODE-ADVANCE THEME`         |

## Wordmark

The OpenCode Advance wordmark pairs a standard "OpenCode" setting with a stylized "Advance" treatment inspired by the Game Boy Advance logo. "Advance" is rendered in a chunky italic-slanted sans-serif with a confident, forward-leaning posture — echoing the hardware whose name it borrows.

Full specification in [`wordmark.md`](wordmark.md).

Short summary:

- **"OpenCode"**: rendered in standard sans-serif, no slant, obsidian color on light backgrounds or ivory on dark backgrounds
- **"ADVANCE"**: rendered in block-style ASCII/italic capitals, slanted forward, in the signature frosted indigo (`#6C7AB8`) with optional brighter accent (`#8B9FE0`) for highlights or animation
- **Relationship**: "Advance" is always visually dominant. "OpenCode" sits above it at smaller weight, framing it

## Color Palette

| Name         | Hex       | RGB              | Role                                                                |
| ------------ | --------- | ---------------- | ------------------------------------------------------------------- |
| **Obsidian**     | `#0B0D10`   | 11, 13, 16       | Primary background. Near-black with subtle cool tint.               |
| **Slate**        | `#1E2228`   | 30, 34, 40       | Elevated surfaces, status bar background, panel backgrounds.       |
| **Graphite**     | `#2D3138`   | 45, 49, 56       | Borders, dividers, inactive elements.                              |
| **Graphite+**    | `#3D424B`   | 61, 66, 75       | Hover/active states on graphite elements.                          |
| **Ivory**        | `#E8E6E3`   | 232, 230, 227    | Primary text. Warm near-white, not pure white.                     |
| **Muted Ivory**  | `#A8A6A3`   | 168, 166, 163    | Secondary text, metadata, timestamps.                              |
| **Dim Ivory**    | `#6A6866`   | 106, 104, 102    | Tertiary/disabled text.                                            |
| **Indigo**       | `#6C7AB8`   | 108, 122, 184    | **Signature accent.** Frosted muted indigo/violet.                     |
| **Indigo+**      | `#8B9FE0`   | 139, 159, 224    | Accent hover/active/highlight. Brighter, more energy.              |
| **Indigo-Glow**  | `#B4C4F5`   | 180, 196, 245    | Rare: used only for wordmark glow effect and critical highlights.  |
| Success      | `#7A9B7A`   | 122, 155, 122    | Muted sage green. Passing checks, healthy state.                   |
| Warning      | `#D4A843`   | 212, 168, 67     | Muted amber. Non-fatal warnings, degraded state.                   |
| Error        | `#D47A7A`   | 212, 122, 122    | Muted coral. Failures, critical errors.                            |

### Accent palette philosophy

The accent color is **frosted indigo** — visually described as muted, slightly desaturated, with a subtle cool-blue undertone. It echoes:

- The Game Boy Advance SP's purple cartridge aesthetic
- Frosted glass / obsidian glass physical textures
- Electric twilight sky
- Premium technical product branding

It is NOT:

- Neon purple / electric magenta (too loud, feels cheap)
- Royal purple (too regal, wrong connotation)
- Lavender (too soft, lacks authority)

### Usage rules

1. **Background**: always Obsidian (`#0B0D10`). Obsidian is the default. Slate is only for elevated surfaces.
2. **Text**: default Ivory (`#E8E6E3`) on Obsidian. Muted Ivory for secondary information.
3. **Accent**: Indigo (`#6C7AB8`) is the ONLY saturated color in normal UI. It marks active elements, highlights, the wordmark, and focus state.
4. **Status colors** (success/warning/error): only appear in doctor output, check results, and error messages. They are muted to stay harmonious with the base palette.
5. **No rainbows**: synthwave edges, multi-color borders, and color-cycling animations are forbidden. Monochrome-plus-accent only.
6. **Frosted treatment**: the indigo accent MAY be rendered with a slight gradient transition from Indigo to Indigo+ (e.g., in the wordmark) to evoke the frosted glass effect. In terminals without truecolor support, it falls back to a single solid Indigo.

### ANSI truecolor escape codes

For terminal rendering, use 24-bit color escape sequences:

| Color     | Escape sequence (foreground)   |
| --------- | ------------------------------ |
| Obsidian  | `\e[38;2;11;13;16m`              |
| Slate     | `\e[38;2;30;34;40m`              |
| Graphite  | `\e[38;2;45;49;56m`              |
| Ivory     | `\e[38;2;232;230;227m`           |
| Indigo    | `\e[38;2;108;122;184m`           |
| Indigo+   | `\e[38;2;139;159;224m`           |
| Success   | `\e[38;2;122;155;122m`           |
| Warning   | `\e[38;2;212;168;67m`            |
| Error     | `\e[38;2;212;122;122m`           |

For backgrounds, swap `38;2` → `48;2`.

## Typography (CLI + docs)

- **Code/terminal**: monospace, user-chosen font. OCA does not impose a font.
- **Wordmark in terminal**: block-character ASCII art (see `wordmark.md`)
- **Wordmark in docs**: monospace ASCII art in fenced code blocks, or SVG when a rendered image is appropriate
- **Headings**: standard Markdown
- **Emphasis**: **bold** for key terms, *italic* for product names referenced for the first time, `monospace` for commands and file paths

## Tone of voice

Professional, direct, confident. Not playful. Not academic. Not salesy.

| Good                                                                                   | Bad                                                                             |
| -------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------- |
| "Declarative source of truth for your OpenCode stack."                                 | "Supercharge your AI dev workflow with the ultimate OpenCode power-up!"         |
| "`oca apply` renders stack.toml into the config files OpenCode reads."                     | "Just run oca apply and watch the magic happen ✨"                               |
| "MCP server ports are verified against the running daemon before reporting healthy."  | "Our proprietary health check algorithm ensures bulletproof reliability."      |
| "Requires Advance as a declared dependency."                                           | "Works seamlessly with our friend Advance!"                                     |

Avoid:

- Exclamation marks (use sparingly in docs, never in error messages)
- Emoji in CLI output except where functionally useful (✓ ✗ ⚠ for status)
- First-person plural ("we", "our") in documentation — prefer neutral voice
- Marketing superlatives ("best", "most powerful", "revolutionary")
- "chad"-era wordplay of any kind

## Attribution

The "Advance" wordmark stylization pays homage to Nintendo's Game Boy Advance logo (2001) as a cultural reference. OpenCode Advance is not affiliated with, endorsed by, or derivative of Nintendo. The wordmark is a re-creation in ASCII/Unicode for terminal rendering, not a copy or derivative of the Nintendo logo asset.
