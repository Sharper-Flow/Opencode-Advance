package install

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/Sharper-Flow/Opencode-Advance/internal/render"
)

const (
	// SentinelStart marks the beginning of an OCA-managed block in shell rc files.
	SentinelStart = "# >>> OCA Managed Block >>>"

	// SentinelEnd marks the end of an OCA-managed block in shell rc files.
	SentinelEnd = "# <<< OCA Managed Block <<<"
)

const shellProfileTemplate = `# This block is managed by OpenCode Advance (oca).
# Do not edit manually — changes will be overwritten on next ` + "`oca apply`" + `.
# To remove: ` + "`oca uninstall`" + ` or delete this block manually.

# Add OCA to PATH (resolved at install time)
_oca_bindir="{{.OCABinDir}}"
case ":$PATH:" in
    *":$_oca_bindir:"*) ;;
    *) export PATH="$_oca_bindir:$PATH" ;;
esac
unset _oca_bindir`

// ReadBlock extracts the managed block content (between sentinels) from a file.
// Returns ("", nil) if file doesn't exist or has no block.
// Returns error if the block structure is malformed (e.g., multiple blocks,
// start without end).
func ReadBlock(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	return extractBlock(string(data))
}

// extractBlock finds the content between sentinels in a string.
func extractBlock(content string) (string, error) {
	startIdx := strings.Index(content, SentinelStart)
	if startIdx == -1 {
		return "", nil
	}

	afterStart := content[startIdx+len(SentinelStart):]
	// Trim leading newline after start sentinel
	if len(afterStart) > 0 && afterStart[0] == '\n' {
		afterStart = afterStart[1:]
	}
	endIdx := strings.Index(afterStart, SentinelEnd)
	if endIdx == -1 {
		return "", fmt.Errorf("malformed managed block: start sentinel found but no end sentinel")
	}

	blockContent := afterStart[:endIdx]

	// Trim trailing newline before end sentinel for canonical comparison
	blockContent = strings.TrimRight(blockContent, "\n")

	// Check for a second block
	afterEnd := afterStart[endIdx+len(SentinelEnd):]
	if strings.Contains(afterEnd, SentinelStart) {
		return "", fmt.Errorf("multiple managed blocks found in file; expected at most one")
	}

	return blockContent, nil
}

// WriteBlock writes a managed block to the file. If a block already exists,
// it is replaced in-place. Otherwise, the block is appended.
// The full file is written atomically via WriteAtomic with maxBackups=5.
func WriteBlock(path string, content string) error {
	var existing string
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if data != nil {
		existing = string(data)
	}

	newContent := replaceOrAppendBlock(existing, content)

	if _, err := render.WriteAtomic(path, []byte(newContent), 0o644, 5); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// replaceOrAppendBlock replaces an existing managed block or appends a new one.
func replaceOrAppendBlock(fileContent string, blockContent string) string {
	block := SentinelStart + "\n" + blockContent + "\n" + SentinelEnd

	startIdx := strings.Index(fileContent, SentinelStart)
	if startIdx == -1 {
		// No existing block — append
		if fileContent == "" {
			return block + "\n"
		}
		if !strings.HasSuffix(fileContent, "\n") {
			fileContent += "\n"
		}
		return fileContent + block + "\n"
	}

	// Replace existing block
	endIdx := strings.Index(fileContent[startIdx:], SentinelEnd)
	if endIdx == -1 {
		// Shouldn't happen since WriteBlock reads after ReadBlock, but handle gracefully
		return fileContent
	}
	endIdx += startIdx + len(SentinelEnd)

	return fileContent[:startIdx] + block + fileContent[endIdx:]
}

// BlockDiff holds the result of comparing a managed block against expected content.
type BlockDiff struct {
	HasBlock bool
	Matches  bool
	Diff     string
}

// DiffBlock compares the managed block in the file against expected content.
func DiffBlock(path string, expected string) (BlockDiff, error) {
	actual, err := ReadBlock(path)
	if err != nil {
		return BlockDiff{}, err
	}

	if actual == "" {
		return BlockDiff{HasBlock: false, Matches: false, Diff: ""}, nil
	}

	if actual == expected {
		return BlockDiff{HasBlock: true, Matches: true, Diff: ""}, nil
	}

	diff := unifiedDiff("expected", "actual", expected, actual)
	return BlockDiff{HasBlock: true, Matches: false, Diff: diff}, nil
}

// IsEdited returns true if the file has a managed block that differs from expected.
func IsEdited(path string, expected string) (bool, error) {
	diff, err := DiffBlock(path, expected)
	if err != nil {
		return false, err
	}
	return diff.HasBlock && !diff.Matches, nil
}

// RemoveBlock removes the managed block from the file.
// If the file doesn't exist or has no block, it is a no-op.
func RemoveBlock(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read %s: %w", path, err)
	}

	content := string(data)
	startIdx := strings.Index(content, SentinelStart)
	if startIdx == -1 {
		return nil
	}

	endIdx := strings.Index(content[startIdx:], SentinelEnd)
	if endIdx == -1 {
		return nil
	}
	endIdx += startIdx + len(SentinelEnd)

	// Consume trailing newline after end sentinel to avoid double newlines
	removeEnd := endIdx
	if removeEnd < len(content) && content[removeEnd] == '\n' {
		removeEnd++
	}

	newContent := content[:startIdx] + content[removeEnd:]
	// Ensure file ends with exactly one newline
	if len(newContent) > 0 {
		newContent = strings.TrimRight(newContent, "\n") + "\n"
	}

	if _, err := render.WriteAtomic(path, []byte(newContent), 0o644, 5); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// RenderShellProfile renders the shell profile block with the given OCA binary
// directory path. The returned string includes sentinel markers.
func RenderShellProfile(ocaBinPath string) string {
	binDir := filepath.Dir(ocaBinPath)
	tmpl, err := template.New("shell_profile").Parse(shellProfileTemplate)
	if err != nil {
		// Template is a compile-time constant — this should never fail
		panic(fmt.Sprintf("parse shell profile template: %v", err))
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]string{"OCABinDir": binDir}); err != nil {
		panic(fmt.Sprintf("execute shell profile template: %v", err))
	}
	return buf.String()
}

// unifiedDiff produces a minimal line-based diff between two strings.
func unifiedDiff(labelA, labelB, a, b string) string {
	linesA := strings.Split(a, "\n")
	linesB := strings.Split(b, "\n")

	var diff strings.Builder
	maxLen := len(linesA)
	if len(linesB) > maxLen {
		maxLen = len(linesB)
	}

	for i := 0; i < maxLen; i++ {
		lineA, lineB := "", ""
		if i < len(linesA) {
			lineA = linesA[i]
		}
		if i < len(linesB) {
			lineB = linesB[i]
		}
		if lineA != lineB {
			if lineA != "" {
				diff.WriteString(fmt.Sprintf("- %s[%d]: %s\n", labelA, i+1, lineA))
			}
			if lineB != "" {
				diff.WriteString(fmt.Sprintf("+ %s[%d]: %s\n", labelB, i+1, lineB))
			}
		}
	}
	return diff.String()
}
