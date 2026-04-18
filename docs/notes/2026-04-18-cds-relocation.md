# Note to future sessions — cds relocation

_Date: 2026-04-18_

Heads up: the `cds` command is no longer a zsh alias and is no longer defined
inside open-chad either. It now has its own canonical project.

## tl;dr

- **Canonical repo:** https://github.com/JRedeker/cds
- **Local clone:** `~/dev/smallsoft/cds`
- **Installed via:** `~/.local/bin/cds → ~/dev/smallsoft/cds/cds`
- **Full behavior + fork notes:** `docs/cds-location.md` in this repo.

## What changed in this session

Two ADV changes landed, both archived:

1. `tidyScratchWorkspace` — moved `cds` out of `~/.zshrc` into a real PATH
   command living in the canonical repo, added tests, installer, README, and
   hygiene updates in `~/scratch`.
2. `initDailyFolderRepos` — taught `cds` to initialize each
   `~/scratch/YYYY-MM-DD` folder as its own git repo and seed `.gitignore` +
   `AGENTS.md` from canonical `templates/`. Parent `~/scratch` now ignores
   top-level date dirs so nested repos are not noisy.

Open-chad's `bin/cds` fork was also updated to match repo-bootstrap behavior,
with its own `templates/cds/` directory and extended tests. Launcher stays
`openchad` there; canonical launches `oc`.

## Why this note exists

If a later session starts thinking of `cds` as "just a scratch alias" or tries
to edit it inside open-chad first, read `docs/cds-location.md` and work in
`~/dev/smallsoft/cds` unless explicitly changing the open-chad fork.
