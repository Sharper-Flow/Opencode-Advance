# Install OpenCode Advance

This guide installs `oca`, applies your `stack.toml`, and verifies the environment.

## Prerequisites

- Go 1.22+ for development builds
- tmux 3.4+ for OCA-managed OpenCode sessions
- git for plugin checkouts and updates
- OpenCode installed and authenticated
- Node.js/Bun only when required by managed plugins such as Advance
- Temporal CLI when `[temporal].enabled = true`

Check local tools:

```bash
go version
tmux -V
git --version
opencode --version
```

## Install from release binary

1. Download the correct `oca` archive from GitHub Releases.
2. Verify `SHA256SUMS.txt`.
3. Install the binary somewhere on `PATH`.

```bash
chmod +x oca
mkdir -p ~/.local/bin
mv oca ~/.local/bin/oca
oca version
```

## Install from source

```bash
git clone https://github.com/Sharper-Flow/Opencode-Advance.git
cd Opencode-Advance
make build
./bin/oca version
```

Optional local install:

```bash
mkdir -p ~/.local/bin
cp ./bin/oca ~/.local/bin/oca
```

## First stack.toml

Create a starter config:

```bash
oca migrate init
```

Or migrate from the older environment:

```bash
oca migrate from-open-chad
```

Review and edit `stack.toml` before applying.

## Install managed shell/profile integration

Run:

```bash
oca install
```

`oca install` adds OCA-managed blocks to your shell profile and tmux integration. It does not own unrelated user content.

After install, reload your shell profile:

```bash
source ~/.zshrc
# or
source ~/.bashrc
```

## First apply

Preview writes:

```bash
oca apply --dry-run
```

Apply when the plan looks right:

```bash
oca apply
```

Verify:

```bash
oca doctor
```

## Start services

If Temporal is enabled:

```bash
oca temporal start
oca temporal status
```

If Discord Rich Presence is desired:

```bash
oca discord enable
oca discord status
```

Start an OpenCode session:

```bash
oca session new
```

## Development isolation

When developing or testing OCA, never point commands at live config directories. Use isolated overrides:

```bash
export OCA_OPENCODE_CONFIG_DIR="$PWD/.dev/opencode"
export OCA_VISION_CONFIG_DIR="$PWD/.dev/vision"
export OCA_PLUGIN_CHECKOUT_ROOT="$PWD/.dev/plugins"
export OCA_CACHE_DIR="$PWD/.dev/cache"
```

Then run:

```bash
oca apply --dry-run
oca doctor
```

## Troubleshooting

| Symptom | Check |
|---|---|
| `oca doctor` reports plugin build failure | Run `oca update`, then `oca apply` |
| Temporal unreachable | Run `oca temporal status`, then `oca temporal start` |
| tmux session missing theme | Run `oca apply`, then start a new `oca session new` |
| Discord not showing presence | Ensure the Discord desktop client is running; run `oca discord status` |
| OpenCode config drift | Run `oca diff`, then `oca apply` |

## Uninstall

Remove OCA-managed shell/config blocks:

```bash
oca uninstall
```

User-owned files and plugin checkouts are not deleted unless explicitly managed by the command output.
