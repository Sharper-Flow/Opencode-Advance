# Capability: phase4-foundation

Phase 4 foundation capability spec for Obsidian theme assets, session lifecycle primitives, managed-block tmux conf template, and isolation requirements.

## Requirements

- `rq-p4f-theme-json01` — `assets/themes/obsidian.json` uses the real OpenCode theme schema (flat `theme.*` keys: primary, secondary, accent, text, textMuted, background) with Obsidian palette values: background=#0B0D10, text=#E8E6E3, accent=#6C7AB8, secondary=#8B9FE0, textMuted=#A8A6A3, border=#2D3138, plus semantic colors error=#D47A7A, warning=#D4A843, success=#7A9B7A.
- `rq-p4f-tmux-conf01` — `assets/themes/obsidian.tmux.conf` is a static tmux 3.4+ conf with `status 2` for 2-row layout, `status-format[0]` (session name left, repo/branch right), `status-format[1]` (window name left, clock right), using Obsidian palette colors for backgrounds, foregrounds, active/inactive panes, and message styling.
- `rq-p4f-tmux-template01` — `templates/tmux.conf.block.gotmpl` wraps tmux conf content with managed-block markers (`# >>> OCA Managed Block >>>` / `# <<< OCA Managed Block <<<`) and accepts `.ConfContent` for the body. Generates a standalone conf file — no `~/.tmux.conf` injection.
- `rq-p4f-session-package01` — `internal/session/` package provides Manager struct with NewManager(socket, repoRoot) that validates tmux binary exists, Create(name, workingDir, tmuxConfPath) that runs tmux via subprocess.Run, List() that parses tmux list-sessions output, and NextSessionName(repoSlug) that returns `oca-<slug>-<n>` with sequential numbering.
- `rq-p4f-session-cli01` — `oca session new [--name <name>] [--no-splash]` creates a tmux session with auto-generated or custom name and triggers boot splash via `tmux send-keys` unless `--no-splash`. `oca session list [--output text|json]` enumerates OCA-managed sessions filtered by `oca-*` prefix.
- `rq-p4f-session-naming01` — Session names follow `oca-<repo-slug>-<n>` pattern where slug is the directory basename lowercased and n is the next sequential integer starting from 0.
- `rq-p4f-shell-helpers01` — `lib/session_lifecycle.sh` provides `session_new` (thin shell wrapper sourcing palette.sh and boot_splash.sh, accepts socket/name/dir/version args) and `session_list` (placeholder). Uses `OCA_TMUX_SOCKET` env var. Exit codes: 0 success, 1 tmux error, 2 invalid args.
- `rq-p4f-boot-splash-ext01` — `lib/boot_splash.sh` supports `OCA_SPLASH_DIR` env var: when set, displays `v{version} · {dir}` below the wordmark instead of just version.
- `rq-p4f-isolation01` — Session creation and listing honor `OCA_TMUX_SOCKET` for isolated tmux servers, `OCA_OPENCODE_CONFIG_DIR`, `OCA_CACHE_DIR`, and `OCA_ASSETS_ROOT` overrides. No writes to live `~/.config/opencode/` or `~/.tmux.conf`.
- `rq-p4f-subprocess01` — All tmux commands in `internal/session/` use `internal/subprocess.Run` with timeout and exit classification — no direct `exec.Command` calls outside the subprocess package.
