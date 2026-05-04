# Archive: Install OCA umbrella plugin into operator's opencode.json — register plugins/oca/ so pane state and watchdog primitives activate, unblocking Pattern B (#1) and hibernation (#2).

**Change ID:** installOcaUmbrellaPlugin
**Archived:** 2026-05-04T18:20:08.030Z
**Created:** 2026-05-04T17:37:40.680Z

## Tasks Completed

- ✅ Change plugins/oca/package.json `main` field from `"dist/index.js"` to `"src/index.ts"` — LBP alignment with other 4 local plugins.
- ✅ Add /home/jrede/dev/opencodeadvance/plugins/oca to opencode.json plugin array (6th entry). Use exact string edit to preserve JSONC comments.
- ✅ Add [plugins.oca] section to stack.example.toml with path = "~/dev/opencodeadvance/plugins/oca". Place after [plugins.anthropic-auth], before [temporal].
- ✅ Post-edit verification: confirm opencode.json has 6 plugin entries, package.json main is "src/index.ts", stack.example.toml has [plugins.oca]. Run go test ./... to ensure no regressions.
- ✅ Run bun install in plugins/oca/ to resolve @opencode-ai/plugin peer dependency. node_modules does not exist — plugin cannot load without it.

## Specs Modified

