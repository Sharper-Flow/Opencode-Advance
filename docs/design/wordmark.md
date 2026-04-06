# Wordmark Specification

The OpenCode Advance wordmark is the project's primary visual identity element. It appears in the README, the CLI `oca version` output, the boot splash, the doctor output, and anywhere the product identifies itself visually.

## Canonical rendering (compact — 3 lines)

```
░█▀█░█▀█░█▀▀░█▀█░█▀▀░█▀█░█▀▄░█▀▀   ░▟█▙░█▀▄░█░█░▟█▙░█▀█░█▀▀░█▀▀
░█░█░█▀▀░█▀▀░█░█░█░░░█░█░█░█░█▀▀   ░█▀█░█░█░▀▄▀░█▀█░█░█░█░░░█▀▀
░▀▀▀░▀░░░▀▀▀░▀░▀░▀▀▀░▀▀▀░▀▀░░▀▀▀   ░█░█░▀▀░░░▀░░█░█░▀░▀░▀▀▀░▀▀▀
```

- **Left half**: `OpenCode` in **Ivory** (`#E8E6E3`)
- **Right half**: `ADVANCE` in **Indigo** (`#6C7AB8`), optionally with frosted gradient to **Indigo+** (`#8B9FE0`) on the top row
- **Separator**: three spaces between halves

## Plain-text variants

For contexts where even the 3-line form is too large (tmux session titles, CLI prompts, compact status):

- `OpenCode ADVANCE` — Ivory + Indigo, with `\e[3m` italic on ADVANCE if terminal supports it
- `OCA` — all three characters in Indigo, single-token identifier only

## Font

The wordmark uses the **pagga** font from the `toilet` figlet font collection (`/usr/share/figlet/pagga.tlf`). This is a 3-line Unicode-block font using `░` (light shade) stipple for frosted anti-aliased edges.

### Stylized A

Each "A" in ADVANCE has two substitutions from the raw pagga output:

| Row    | Raw pagga | Stylized | Effect                        |
|--------|-----------|----------|-------------------------------|
| Top    | `░█▀█`    | `░▟█▙`   | Flat-top trapezoid (GBA apex) |
| Bottom | `░▀░▀`    | `░█░█`   | Full-block legs               |
| Middle | `░█▀█`    | `░█▀█`   | Unchanged (crossbar)          |

### Regenerating

```bash
toilet -f pagga "OpenCode"
toilet -f pagga "ADVANCE"
```

Then apply the two substitutions to each A. Do NOT reformat, trim, or edit the raw output.

## Color rendering

The wordmark MUST always render in color when the terminal supports it. Never show in default white/no-color.

| Context       | OpenCode         | ADVANCE                          |
|---------------|------------------|----------------------------------|
| Default       | Ivory `#E8E6E3`  | Indigo `#6C7AB8`                 |
| Boot splash   | Ivory            | Indigo+ `#8B9FE0` → Indigo      |
| Monochrome    | bold             | normal + asterisks `*ADVANCE*`   |

### 256-color fallback

| Design color | 256-color code | Approximate hex |
|--------------|----------------|-----------------|
| Ivory        | `255`          | `#EEEEEE`       |
| Indigo       | `103`          | `#8787AF`       |
| Indigo+      | `147`          | `#AFAFD7`       |

## Rendering rules

1. **Never modify the character layout.** Whitespace, stipple positions, and block arrangement are the design.
2. **Always pair OpenCode with ADVANCE.** Never show ADVANCE alone as a wordmark.
3. **Color hierarchy is non-negotiable.** OpenCode is Ivory, ADVANCE is Indigo.
4. **The stipple edge is sacred.** Do not replace `░` with other characters.
5. **No animation outside the boot splash.** Normal CLI output uses stable Indigo.
6. **Background must be dark.** Designed for Obsidian/Slate/Graphite backgrounds. On light backgrounds, fall back to plain-text form.

## File references

- `docs/design/wordmark.txt` — canonical ASCII source (this file's companion)
- `lib/wordmark.sh` — bash rendering function (to be implemented)
- `internal/render/wordmark.go` — Go implementation (to be implemented)
- `lib/boot_splash.sh` — boot splash renderer (to be implemented)
- `cmd/oca/version.go` — `oca version` command output (to be implemented)

## Trademark note

"Game Boy" and "Game Boy Advance" are trademarks of Nintendo. OpenCode Advance is an independent project with no affiliation to Nintendo. The wordmark is an original Unicode composition inspired by the GBA aesthetic as a cultural reference; it is not a copy or derivative of any Nintendo asset.
