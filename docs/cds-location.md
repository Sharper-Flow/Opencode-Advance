# `cds` — Canonical Location

The canonical `cds` command (Create Date-stamped Scratch dir → git init → seed
templates → launch editor) lives in its own repo:

- **Repo:** https://github.com/JRedeker/cds
- **Local clone:** `~/dev/smallsoft/cds`
- **Installed symlink:** `~/.local/bin/cds → ~/dev/smallsoft/cds/cds`

## Behavior summary

On every run, `cds`:

1. resolves the target daily folder `~/scratch/YYYY-MM-DD` (today by default; accepts explicit `YYYY-MM-DD` arg)
2. creates it if missing
3. initializes it as a git repo if not already one (plain `git init -q`, respects local `init.defaultBranch`)
4. seeds `.gitignore` and `AGENTS.md` from `templates/` if missing (never overwrites existing files)
5. `cd`s into it
6. `exec`s `oc` to launch the editor (skippable with `--no-launch`)

## Related forks

- `~/dev/open-chad/bin/cds` is a sibling fork maintained inside open-chad.
  - Same repo-bootstrap behavior and seeded templates.
  - Launches `openchad` instead of `oc`.
  - Installed by `~/dev/open-chad/install.sh`, which may re-point
    `~/.local/bin/cds` — decline that prompt if the smallsoft canonical copy
    should remain installed.

## Historical note

`cds` was originally a zsh alias in `~/.zshrc`. The archived ADV changes
`tidyScratchWorkspace` and `initDailyFolderRepos` moved it out of the shell rc,
into a real PATH command, then taught it to bootstrap per-day git repos with
seeded continuity files.
