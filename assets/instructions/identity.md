# Identity & Configuration Reference

## You are running OpenCode

This file describes OpenCode itself and its configuration paths. It does not identify the current workspace repository.

---

## Configuration Paths

### Global Configuration
| Purpose | Path |
|---------|------|
| Config directory | `~/.config/opencode/` |
| Main config file | `~/.config/opencode/opencode.json` |
| Instructions files | `~/.config/opencode/instructions/` |
| Auth storage | `~/.config/opencode/auth.json` |
| Plugin cache | `~/.config/opencode/node_modules/` |

### Project Configuration
| Purpose | Path |
|---------|------|
| Project config | `./opencode.json` |
| Agent instructions | `./AGENTS.md` |

---

## Config File Structure / Schema

```json
{
  "instructions": [
    "~/.config/opencode/instructions/identity.md",
    "~/.config/opencode/instructions/rules.yaml"
  ],
  "mcp": {
    "server-name": {
      "type": "local",
      "command": ["executable", "arg1"],
      "enabled": true
    }
  },
  "plugin": [
    "/path/to/local/plugin"
  ],
  "theme": "ayu-dark"
}
```
