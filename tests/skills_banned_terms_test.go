package tests

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// TestSkillsHaveNoBannedTerms walks every OCA-owned skill markdown file under
// assets/skills/ and asserts no banned legacy terms appear. Each category has
// an optional allow-list of file paths (relative to assets/skills/) where the
// term is permitted for documented reasons (e.g., explicit historical notes).
//
// Locks the M6 enforcement gap: previously the only check was a one-off test
// for the worktree skill. This walking test catches future drift across the
// entire OCA-owned skill set.
//
// Categories:
//   - legacy-openchad: pre-cutover environment terminology
//   - legacy-mcp-names: stale MCP function names (pre-schema-correct forms)
//   - deleted-adv-skills: skill names that ADV inlined into commands + deleted;
//     OCA's README has an explicit "inlined-and-deleted" note that uses these
//     names, so README.md is allow-listed
func TestSkillsHaveNoBannedTerms(t *testing.T) {
	type category struct {
		name  string
		terms []string
		// allowFiles is a set of file paths (relative to assets/skills/) where
		// the banned terms are permitted. Used for explicit historical-marker
		// content like the README's inlined-and-deleted note.
		allowFiles map[string]bool
	}
	categories := []category{
		{
			name: "legacy-openchad",
			terms: []string{
				"openchad",
				"open-chad",
				"oc switch",
			},
		},
		{
			name: "legacy-mcp-names",
			terms: []string{
				"kagi_search_fetch",
				"firecrawl_scrape",
				"context7_resolve_library_id",
			},
		},
		{
			name: "deleted-adv-skills",
			terms: []string{
				"adv-review-methodology",
				"adv-apply-methodology",
				"adv-harden-methodology",
				"adv-discover-methodology",
				"adv-prep-methodology",
			},
			allowFiles: map[string]bool{
				"README.md": true,
			},
		},
	}

	// Compile each banned term as a word-boundary regex. This avoids false
	// positives where a banned legacy term is a substring of a current,
	// schema-correct name (e.g., `kagi_search_fetch` inside `kagi_kagi_search_fetch`).
	type bannedPattern struct {
		category string
		term     string
		re       *regexp.Regexp
	}
	var patterns []bannedPattern
	for _, cat := range categories {
		for _, term := range cat.terms {
			patterns = append(patterns, bannedPattern{
				category: cat.name,
				term:     term,
				re:       regexp.MustCompile(`\b` + regexp.QuoteMeta(term) + `\b`),
			})
		}
	}

	skillsRoot := filepath.Join("..", "assets", "skills")
	type hit struct {
		category string
		term     string
		relPath  string
		line     int
		text     string
	}
	var hits []hit

	walkErr := filepath.Walk(skillsRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".md") {
			return nil
		}
		rel, err := filepath.Rel(skillsRoot, path)
		if err != nil {
			return err
		}
		// Normalize separators for cross-platform comparison against allow lists.
		rel = filepath.ToSlash(rel)

		contentBytes, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		// Per-file allow-list lookup by category.
		fileAllowed := func(catName string) bool {
			for _, cat := range categories {
				if cat.name == catName {
					return cat.allowFiles[rel]
				}
			}
			return false
		}

		scanner := bufio.NewScanner(bytes.NewReader(contentBytes))
		scanner.Buffer(make([]byte, 64*1024), 1024*1024)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()
			for _, p := range patterns {
				if fileAllowed(p.category) {
					continue
				}
				if p.re.MatchString(line) {
					hits = append(hits, hit{
						category: p.category,
						term:     p.term,
						relPath:  rel,
						line:     lineNum,
						text:     strings.TrimSpace(line),
					})
				}
			}
		}
		return scanner.Err()
	})
	if walkErr != nil {
		t.Fatalf("walk assets/skills: %v", walkErr)
	}

	if len(hits) == 0 {
		return
	}

	// Sort hits deterministically for stable failure output.
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].category != hits[j].category {
			return hits[i].category < hits[j].category
		}
		if hits[i].relPath != hits[j].relPath {
			return hits[i].relPath < hits[j].relPath
		}
		return hits[i].line < hits[j].line
	})

	var b strings.Builder
	b.WriteString("banned terms found in OCA-owned skills:\n")
	for _, h := range hits {
		b.WriteString("  [")
		b.WriteString(h.category)
		b.WriteString("] ")
		b.WriteString(h.relPath)
		b.WriteString(":")
		// fmt would pull in another import; keep it minimal
		b.WriteString(itoa(h.line))
		b.WriteString(" — ")
		b.WriteString(h.term)
		b.WriteString("\n    ")
		b.WriteString(h.text)
		b.WriteByte('\n')
	}
	t.Fatal(b.String())
}

// itoa avoids importing strconv for a single conversion.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
