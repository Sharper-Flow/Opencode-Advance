# Templates

Go `text/template` files that `oca apply` renders into user config files.

## Inventory (target state at v1.0)

| Template                     | Renders into                                     | Phase |
| ---------------------------- | ------------------------------------------------ | ----- |
| `opencode.json.gotmpl`         | `~/.config/opencode/opencode.json` (merged)        | 1-3   |
| `vision-servers.yaml.gotmpl`   | `~/.config/vision/servers.yaml`                    | 1     |
| `tmux.conf.block.gotmpl`       | Block injected into `~/.tmux.conf`                 | 4     |
| `shell_profile.block.gotmpl`   | Block injected into `~/.zshrc` / `~/.bashrc`         | 5     |

## Fragment templates

For partial renders (e.g., `oca apply --target mcp`), sub-templates render individual sections:

| Fragment                         | Slice of opencode.json |
| -------------------------------- | ---------------------- |
| `opencode.json.mcp.gotmpl`         | `.mcp`                   |
| `opencode.json.plugins.gotmpl`     | `.plugin`                |
| `opencode.json.instructions.gotmpl`| `.instructions`          |
| `opencode.json.providers.gotmpl`   | `.provider`              |
| `opencode.json.agents.gotmpl`      | `.agent`                 |
| `opencode.json.permissions.gotmpl` | `.permission`            |
| `opencode.json.watcher.gotmpl`     | `.watcher`               |
| `opencode.json.lsp.gotmpl`         | `.lsp`                   |

## Merge semantics

The top-level `opencode.json.gotmpl` does NOT overwrite the entire `opencode.json`. It produces a set of fragments that are merged into the existing file using the idempotent merge algorithm in `internal/render/merge.go`:

- Array fields: append if missing (deduped by identity)
- Object fields: OCA-managed keys overwrite; non-managed keys preserved
- Nested objects: recurse

This preserves user-added fields that are not declared in stack.toml.

## Status

Empty — populated in Phase 1 onward as each slice of coverage comes online.
