package health

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

func init() {
	registerBuiltin("context-budget", CheckContextBudget)
}

// tokenEstimateChars is the approximate characters-per-token ratio for
// instruction and agent prompt content. Conservative estimate; actual
// tokenization varies by model but this provides a consistent budget metric.
const tokenEstimateChars = 4

// budgetThresholdBytes defines soft thresholds for instruction size warnings.
const (
	budgetWarnPerFileBytes = 50 * 1024 // 50KB per file
	budgetWarnTotalBytes   = 200 * 1024 // 200KB total baseline
)

// fileEntry tracks a single file's contribution to the context budget.
type fileEntry struct {
	Path  string
	Size  int64
	Owner string // "instructions", "agents", "skills", "commands"
}

// CheckContextBudget audits the estimated token cost of always-loaded
// configuration files (instructions, agents, skills, commands). It reports
// per-category sizes, identifies the largest contributors, and warns when
// the baseline context budget exceeds soft thresholds.
func CheckContextBudget(ctx context.Context, stack *cfg.Stack, opts Options) ([]Check, error) {
	var checks []Check

	if opts.ConfigDir == "" {
		return []Check{{
			Name:    "context-budget.config",
			Status:  StatusWarn,
			Message: "ConfigDir not set — context budget audit skipped",
		}}, nil
	}

	var entries []fileEntry

	// 1. Instruction files (from stack.toml [instructions].order or filesystem scan)
	entries = append(entries, scanInstructionFiles(stack, opts)...)

	// 2. Agent files
	entries = append(entries, scanDirEntries(filepath.Join(opts.ConfigDir, "agents"), "agents")...)

	// 3. Skill files (top-level SKILL.md only per skill)
	entries = append(entries, scanDirEntries(filepath.Join(opts.ConfigDir, "skills"), "skills")...)

	// 4. Command files
	entries = append(entries, scanDirEntries(filepath.Join(opts.ConfigDir, "command"), "commands")...)

	if len(entries) == 0 {
		checks = append(checks, Check{
			Name:    "context-budget",
			Status:  StatusPass,
			Message: "No deployed configuration files found — budget audit skipped",
		})
		return checks, nil
	}

	// Aggregate by owner
	ownerTotals := make(map[string]int64)
	var totalBytes int64
	for _, e := range entries {
		ownerTotals[e.Owner] += e.Size
		totalBytes += e.Size
	}

	// Report per-category sizes
	for _, owner := range sortedKeys(ownerTotals) {
		bytes := ownerTotals[owner]
		tokens := bytes / tokenEstimateChars
		checks = append(checks, Check{
			Name:    fmt.Sprintf("context-budget.%s", owner),
			Status:  StatusPass,
			Message: fmt.Sprintf("%s: %d files, ~%d bytes (~%d tokens)", owner, countOwner(entries, owner), bytes, tokens),
		})
	}

	// Report total
	totalTokens := totalBytes / tokenEstimateChars
	checks = append(checks, Check{
		Name:   "context-budget.total",
		Status: StatusPass,
		Message: fmt.Sprintf("Total baseline: %d files, ~%d bytes (~%d tokens)",
			len(entries), totalBytes, totalTokens),
	})

	// Warn on total budget
	if totalBytes > budgetWarnTotalBytes {
		checks = append(checks, Check{
			Name:    "context-budget.total.threshold",
			Status:  StatusWarn,
			Message: fmt.Sprintf("Baseline context exceeds %dKB (~%d tokens) — consider lazy-loading phase-specific guidance", budgetWarnTotalBytes/1024, totalTokens),
			Hint:    "move phase-specific methodology from always-loaded instructions to skills/commands",
		})
	}

	// Identify top-5 largest files
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Size > entries[j].Size
	})
	topN := 5
	if len(entries) < topN {
		topN = len(entries)
	}
	for i := 0; i < topN; i++ {
		e := entries[i]
		status := StatusPass
		hint := ""
		if e.Size > budgetWarnPerFileBytes {
			status = StatusWarn
			hint = "consider splitting into always-loaded core + lazy-loaded detail"
		}
		checks = append(checks, Check{
			Name:    fmt.Sprintf("context-budget.largest.%d", i+1),
			Status:  status,
			Message: fmt.Sprintf("%s (%s): ~%dKB", filepath.Base(e.Path), e.Owner, e.Size/1024),
			Hint:    hint,
		})
	}

	return checks, nil
}

// scanInstructionFiles scans instruction paths from stack.toml order or falls
// back to scanning the instructions directory.
func scanInstructionFiles(stack *cfg.Stack, opts Options) []fileEntry {
	var entries []fileEntry
	instrDir := filepath.Join(opts.ConfigDir, "instructions")

	// If stack.toml declares instruction order, scan those paths
	if len(stack.Instructions.Order) > 0 {
		for _, path := range stack.Instructions.Order {
			resolved := resolvePath(path, opts)
			info, err := os.Stat(resolved)
			if err != nil {
				continue
			}
			if !info.IsDir() {
				entries = append(entries, fileEntry{
					Path:  resolved,
					Size:  info.Size(),
					Owner: "instructions",
				})
			}
		}
		// Also scan directory for files not in the order list
		declared := make(map[string]bool)
		for _, path := range stack.Instructions.Order {
			declared[filepath.Base(resolvePath(path, opts))] = true
		}
		extraEntries := scanDirEntries(instrDir, "instructions")
		for _, e := range extraEntries {
			if !declared[filepath.Base(e.Path)] {
				entries = append(entries, e)
			}
		}
	} else {
		// Fall back to directory scan
		entries = append(entries, scanDirEntries(instrDir, "instructions")...)
	}

	return entries
}

// scanDirEntries returns file entries for all regular files in dir.
func scanDirEntries(dir, owner string) []fileEntry {
	var entries []fileEntry
	dirents, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	for _, d := range dirents {
		if d.IsDir() {
			// Recurse one level for skills (skill-name/SKILL.md pattern)
			if owner == "skills" {
				subEntries := scanDirEntries(filepath.Join(dir, d.Name()), owner)
				// Only include SKILL.md or .md files for skills
				for _, se := range subEntries {
					base := filepath.Base(se.Path)
					if base == "SKILL.md" || strings.HasSuffix(base, ".md") {
						entries = append(entries, se)
					}
				}
			}
			continue
		}
		info, err := d.Info()
		if err != nil {
			continue
		}
		entries = append(entries, fileEntry{
			Path:  filepath.Join(dir, d.Name()),
			Size:  info.Size(),
			Owner: owner,
		})
	}
	return entries
}

func sortedKeys(m map[string]int64) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func countOwner(entries []fileEntry, owner string) int {
	n := 0
	for _, e := range entries {
		if e.Owner == owner {
			n++
		}
	}
	return n
}
