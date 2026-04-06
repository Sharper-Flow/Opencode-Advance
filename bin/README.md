# bin/ — Shell entry points

Shell-level entry scripts installed onto the user's PATH. These are thin wrappers; the real logic lives in the Go binary at `cmd/oca/`.

## Inventory (target state at v1.0)

| Script        | Purpose                                                         |
| ------------- | --------------------------------------------------------------- |
| `oca`           | Main entry point (thin wrapper around the Go binary)           |
| `oc`            | Short alias (retains muscle memory from open-chad)             |
| `cds`           | Date-stamped scratch directory launcher                        |
| `ocashell.sh`   | Shell completion bootstrap                                     |

## Status

Empty — populated in Phase 5 (installer phase). The Go binary itself is built to `./bin/oca` during development but those dev builds are gitignored.

## Not committed

- `./bin/oca` — the built Go binary (gitignored; produced by `make build`)
- `./bin/oca-*` — any additional Go-built binaries
