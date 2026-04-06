# Wordmark Specification

The OpenCode Advance wordmark is the project's primary visual identity element. It appears in the README, the CLI `oca version` output, the boot splash, the doctor output, and anywhere else the product needs to identify itself visually.

## Design rationale

The "Advance" half of the wordmark is stylized after the Game Boy Advance logo (Nintendo, 2001) — a chunky italic-slanted sans-serif that reads as confident, forward-leaning, and unmistakably technical. Paired with an understated "OpenCode" setting above it, the wordmark establishes the project as:

1. **Technical** — the block/slab letterforms evoke hardware and system-level tools
2. **Confident** — the forward slant carries momentum without being aggressive
3. **Retro-professional** — a reference to iconic hardware, but rendered in muted tones for a mature audience
4. **Frosted** — rendered in muted indigo with optional gradient, not loud primary colors

## Full wordmark (terminal ASCII)

The canonical rendering uses Unicode box-drawing characters with a leftward progressive indent to create the slant:

```
   ██████╗ ██████╗ ███████╗███╗   ██╗ ██████╗ ██████╗ ██████╗ ███████╗
  ██╔═══██╗██╔══██╗██╔════╝████╗  ██║██╔════╝██╔═══██╗██╔══██╗██╔════╝
  ██║   ██║██████╔╝█████╗  ██╔██╗ ██║██║     ██║   ██║██║  ██║█████╗
  ██║   ██║██╔═══╝ ██╔══╝  ██║╚██╗██║██║     ██║   ██║██║  ██║██╔══╝
  ╚██████╔╝██║     ███████╗██║ ╚████║╚██████╗╚██████╔╝██████╔╝███████╗
   ╚═════╝ ╚═╝     ╚══════╝╚═╝  ╚═══╝ ╚═════╝ ╚═════╝ ╚═════╝ ╚══════╝

           █████╗ ██████╗ ██╗   ██╗ █████╗ ███╗   ██╗ ██████╗███████╗
          ██╔══██╗██╔══██╗██║   ██║██╔══██╗████╗  ██║██╔════╝██╔════╝
         ███████║██║  ██║██║   ██║███████║██╔██╗ ██║██║     █████╗
        ██╔══██║██║  ██║╚██╗ ██╔╝██╔══██║██║╚██╗██║██║     ██╔══╝
       ██║  ██║██████╔╝ ╚████╔╝ ██║  ██║██║ ╚████║╚██████╗███████╗
      ╚═╝  ╚═╝╚═════╝   ╚═══╝  ╚═╝  ╚═╝╚═╝  ╚═══╝ ╚═════╝╚══════╝
```

### Anatomy

- **Row 1** (OPENCODE): 6 lines tall, using the "ANSI Shadow" block letterform style, rendered in Ivory (`#E8E6E3`) on obsidian backgrounds
- **Gap**: 1 blank line separating the two halves
- **Row 2** (ADVANCE): 6 lines tall, same letterform style, but with a 2-column-per-line leftward indent progression (row 0 = 0 indent, row 5 = -5 columns... rendered as progressive left-padding) to create the forward slant
- **Color**: "ADVANCE" rendered in Indigo (`#6C7AB8`), with optional per-row gradient toward Indigo+ (`#8B9FE0`) on the top rows to create the frosted glass effect
- **Optional glow**: when rendered during the boot splash animation, a brief pulse of Indigo-Glow (`#B4C4F5`) can accent the outline before settling into Indigo

## Slant progression

The slant is achieved by varying the leading whitespace per line. For the "ADVANCE" block, starting from the top:

| Line | Leading spaces | Slant effect                |
| ---- | -------------- | --------------------------- |
| 1    | 11             | baseline reference          |
| 2    | 10             | −1 column                   |
| 3    | 9              | −2 columns                  |
| 4    | 8              | −3 columns                  |
| 5    | 7              | −4 columns                  |
| 6    | 6              | −5 columns (maximum slant)  |

This creates the forward-leaning italic posture of the original GBA wordmark without requiring actual italic ASCII characters.

## Compact single-line variant

For contexts where the full 13-line wordmark is too large (status bar, `oca version` short form, terminal prompts), use the compact form:

```
OpenCode ADVANCE
```

- "OpenCode" in Ivory
- " " space in default color
- "ADVANCE" in Indigo (`#6C7AB8`), with `\e[3m` italic if terminal supports it

Or the even shorter:

```
OCA
```

- All three characters in Indigo (`#6C7AB8`)
- Use only where a single-token identifier is needed (tmux session titles, compact status)

## Rendering rules

1. **Never show the wordmark without proper color rendering.** If a terminal does not support 24-bit color, fall back to 256-color approximations (see below). Never show it in default white/no-color.
2. **Never animate colors during normal display.** The boot splash is the ONLY context where color transitions are allowed on the wordmark.
3. **Never rotate, skew beyond the specified slant, or apply custom effects.** The wordmark is a fixed asset.
4. **The OpenCode/ADVANCE relationship is non-negotiable.** ADVANCE is always visually dominant. OpenCode frames it from above.
5. **Do not use the wordmark on backgrounds other than Obsidian, Slate, or Graphite.** If displaying on a user-chosen theme with different backgrounds, fall back to the compact single-line variant.

## 256-color fallback

For terminals that lack truecolor support:

| Design color | 256-color code | Approximate hex |
| ------------ | -------------- | --------------- |
| Ivory        | `255`            | `#EEEEEE`         |
| Indigo       | `103`            | `#8787AF`         |
| Indigo+      | `147`            | `#AFAFD7`         |
| Obsidian     | `232`            | `#080808`         |
| Slate        | `235`            | `#262626`         |

Use `\e[38;5;<code>m` for foreground, `\e[48;5;<code>m` for background.

## Monochrome fallback

For terminals with no color at all:

- Full wordmark: render as-is, no color
- Compact: `OpenCode ADVANCE` with the word `ADVANCE` surrounded by asterisks: `OpenCode *ADVANCE*`

## File references

The wordmark is rendered by:

- `cmd/oca/main.go` — `oca version` output
- `lib/boot_splash.sh` — tmux session boot splash
- `internal/render/wordmark.go` — canonical Go implementation
- `docs/design/wordmark.txt` — reference ASCII source file

## Trademark note

"Game Boy" and "Game Boy Advance" are trademarks of Nintendo. OpenCode Advance is an independent project with no affiliation to Nintendo. The wordmark is an original ASCII art composition inspired by the GBA logo as a cultural reference; it is not a copy or derivative of any Nintendo asset.
