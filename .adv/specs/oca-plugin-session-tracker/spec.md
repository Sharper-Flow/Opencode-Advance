# OCA Plugin Session Tracker

Capability spec for per-pane opencode TUI restart with session continuity.

## Requirements

### rq-opst-plugin-scaffold01: Plugin scaffolding

The `plugins/oca/` directory exists with a valid bun-based TypeScript project structure.

**Given** the OCA repository at any commit on the change branch  
**When** `cd plugins/oca && bun test` is run  
**Then** all tests pass and `bun run build` produces `dist/index.js`

### rq-opst-session-tracker01: Session tracker feature module

The OCA plugin captures opencode session lifecycle events and persists per-pane state.

**Given** an OCA tmux session with `TMUX_PANE=%42` and the OCA plugin installed and enabled  
**When** opencode creates a new session with ID `ses_abc123` in that pane  
**Then** the file `$XDG_STATE_HOME/oca/panes/oca/42.json` contains `{"sessionID":"ses_abc123","directory":"...","ts":...}`  

**Given** the same pane has state file `42.json` with `sessionID = "ses_abc123"`  
**When** opencode deletes session `ses_abc123`  
**Then** the state file `42.json` is removed  

**Given** no `TMUX_PANE` environment variable is set  
**When** the plugin's `sessionCreated` hook fires  
**Then** no state file is written (graceful no-op)  

### rq-opst-pane-restart01: Pane restart command

`oca pane restart-tui` restarts opencode in the current tmux pane with session continuity.

**Given** pane `%42` is dead and has state file with `sessionID = "ses_abc123"`  
**When** `oca pane restart-tui` runs inside that pane  
**Then** tmux respawns the pane running `opencode -s ses_abc123`  

**Given** pane `%42` is alive and running opencode  
**When** `oca pane restart-tui` runs without `--force`  
**Then** the command exits with code 1 and message "pane is running opencode. Use --force to restart."  

**Given** pane `%42` is alive and running opencode  
**When** `oca pane restart-tui --force` runs  
**Then** tmux kills and respawns the pane running `opencode -s ses_abc123`  

**Given** pane `%42` is alive and running `bash` (not opencode)  
**When** `oca pane restart-tui --force` runs  
**Then** the command exits with code 1 and refuses to respawn  

**Given** pane `%42` has no state file  
**When** `oca pane restart-tui` runs  
**Then** the command warns and falls back to `opencode --continue`  

**Given** `TMUX_PANE` is not set in the environment  
**When** `oca pane restart-tui` runs  
**Then** the command exits with code 1 and message "TMUX_PANE not set"  

### rq-opst-apply-integration01: Apply render integration

The `[plugins.oca]` section in `stack.toml` flows through the existing render pipeline.

**Given** a `stack.toml` containing `[plugins.oca]` with `source = "local:plugins/oca"` and `enabled = true`  
**When** `oca apply --target plugins` runs from the repo root  
**Then** the rendered `opencode.json` contains the plugin path `plugins/oca/dist/index.js` in the `.plugin` array  

**Given** the same stack.toml but with `enabled = false`  
**When** `oca apply --target plugins` runs  
**Then** the OCA plugin entry is removed from `opencode.json` `.plugin` array  

**Given** `oca doctor --scope plugins` runs  
**When** the OCA plugin is declared and enabled  
**Then** the check `plugins.oca.local_dir_exists` passes if the directory exists  

### rq-opst-tmux-keybind01: Tmux keybind

The Obsidian tmux theme includes a guarded prefix+r keybind for pane restart.

**Given** an OCA tmux session named `oca-project-0`  
**When** the user presses prefix+r  
**Then** `oca pane restart-tui` executes in the current pane  

**Given** a non-OCA tmux session named `work`  
**When** the user presses prefix+r  
**Then** the literal character `r` is sent to the pane (default tmux behavior)  
