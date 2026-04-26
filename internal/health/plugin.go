package health

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
	"github.com/Sharper-Flow/Opencode-Advance/internal/render"
	"github.com/Sharper-Flow/Opencode-Advance/internal/subprocess"
)

func init() {
	registerBuiltin("plugins", CheckPlugins)
}

func CheckPlugins(ctx context.Context, stack *cfg.Stack, opts Options) ([]Check, error) {
	paths := cfg.ResolvePaths()
	pluginEntries, instructionEntries, err := readOpencodeArrays(paths.OpencodeJSON())
	if err != nil {
		return nil, err
	}

	checks := make([]Check, 0)
	declaredFragments := map[string]string{}
	names := make([]string, 0, len(stack.Plugins))
	for name := range stack.Plugins {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		plugin := stack.Plugins[name]
		if !plugin.IsEnabled() {
			continue
		}

		fragment := render.RenderPluginFragment(plugin)
		declaredFragments[name] = fragment

		if plugin.IsNPMSource() {
			checks = append(checks,
				Check{Name: "plugins." + name + ".checkout_exists", Status: StatusPass, Message: "npm source - no checkout required"},
				Check{Name: "plugins." + name + ".built_artifact", Status: StatusPass, Message: "npm source - no local artifact required"},
				Check{Name: "plugins." + name + ".git_ref", Status: StatusPass, Message: "npm source - no git ref required"},
			)
		} else if plugin.IsLocalSource() {
			localDir := plugin.Source[6:] // strip "local:" prefix
			checks = append(checks, statCheck("plugins."+name+".local_dir_exists", localDir, "local directory present", "local directory missing"))
			checks = append(checks, statCheck("plugins."+name+".built_artifact", fragment, "built artifact present", "built artifact missing"))
			checks = append(checks, Check{Name: "plugins." + name + ".git_ref", Status: StatusPass, Message: "local source - no git ref required"})
		} else {
			checks = append(checks, statCheck("plugins."+name+".checkout_exists", plugin.Checkout, "checkout present", "checkout missing"))
			checks = append(checks, statCheck("plugins."+name+".built_artifact", plugin.Path, "built artifact present", "built artifact missing"))
			checks = append(checks, gitRefCheck(ctx, name, plugin))
		}

		if containsString(pluginEntries, fragment) {
			checks = append(checks, Check{Name: "plugins." + name + ".path_in_opencode_json", Status: StatusPass, Message: "plugin entry present in opencode.json"})
			checks = append(checks, Check{Name: "plugins.drift.declared_and_present." + name, Status: StatusPass, Message: "declared plugin entry present"})
		} else {
			checks = append(checks, Check{Name: "plugins." + name + ".path_in_opencode_json", Status: StatusWarn, Message: "declared plugin entry missing from opencode.json", Hint: "run oca apply --target plugins"})
		}

		if providesInstructionsPatched(plugin, instructionEntries) {
			checks = append(checks, Check{Name: "plugins.drift.provides_patched." + name, Status: StatusPass, Message: "plugin-provided instructions intentionally omitted from opencode.json (provides-patched)"})
		}
	}

	for _, entry := range pluginEntries {
		if containsDeclaredFragment(declaredFragments, entry) {
			continue
		}
		status := StatusPass
		msg := "user-added plugin entry preserved"
		if strings.Contains(entry, "/opencode/worktree/") {
			status = StatusWarn
			msg = "unexpected plugin entry in opencode.json"
			checks = append(checks, Check{Name: "plugins.drift.unexpected." + entry, Status: status, Message: msg, Hint: "run oca apply --target plugins to prune stale worktree entries"})
			continue
		}
		checks = append(checks, Check{Name: "plugins.drift.user_added." + entry, Status: status, Message: msg})
	}

	return checks, nil
}

func readOpencodeArrays(path string) ([]string, []string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("read opencode.json: %w", err)
	}
	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, nil, fmt.Errorf("parse opencode.json: %w", err)
	}
	return readStringArray(root["plugin"]), readStringArray(root["instructions"]), nil
}

func readStringArray(raw any) []string {
	arr, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func statCheck(name, path, okMsg, failMsg string) Check {
	if path == "" {
		return Check{Name: name, Status: StatusWarn, Message: failMsg}
	}
	if _, err := os.Stat(path); err != nil {
		return Check{Name: name, Status: StatusWarn, Message: failMsg, Hint: path}
	}
	return Check{Name: name, Status: StatusPass, Message: okMsg}
}

func gitRefCheck(ctx context.Context, name string, plugin cfg.Plugin) Check {
	ref := plugin.Ref
	if ref == "" {
		ref = "trunk"
	}
	res, err := subprocess.Run(ctx, subprocess.Cmd{Name: "git", Args: []string{"rev-parse", ref}, Dir: plugin.Checkout})
	if err != nil {
		return Check{Name: "plugins." + name + ".git_ref", Status: StatusWarn, Message: "git ref not resolvable", Hint: string(render.Redact(res.Output))}
	}
	return Check{Name: "plugins." + name + ".git_ref", Status: StatusPass, Message: "git ref resolves locally"}
}

func providesInstructionsPatched(plugin cfg.Plugin, instructionEntries []string) bool {
	provided := false
	for _, category := range plugin.Provides {
		if category == cfg.ProvidesInstructions {
			provided = true
			break
		}
	}
	if !provided {
		return false
	}
	for _, instruction := range plugin.Instructions {
		if strings.HasSuffix(instruction, "ADV_INSTRUCTIONS.md") && !containsString(instructionEntries, instruction) {
			return true
		}
	}
	return false
}

func containsDeclaredFragment(declared map[string]string, entry string) bool {
	for _, fragment := range declared {
		if fragment == entry {
			return true
		}
	}
	return false
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
