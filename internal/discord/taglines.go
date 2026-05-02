package discord

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

const DefaultTagline = "Spec-driven development"

type taglinesFile struct {
	Taglines []string `toml:"taglines"`
}

// LoadTaglines reads a TOML tagline file. Missing files and empty pools fall
// back to DefaultTagline so Discord presence never crashes on config drift.
func LoadTaglines(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []string{DefaultTagline}, nil
		}
		return []string{DefaultTagline}, err
	}

	var parsed taglinesFile
	if err := toml.Unmarshal(data, &parsed); err != nil {
		return []string{DefaultTagline}, fmt.Errorf("parse taglines %s: %w", path, err)
	}

	taglines := make([]string, 0, len(parsed.Taglines))
	for _, tagline := range parsed.Taglines {
		trimmed := strings.TrimSpace(tagline)
		if trimmed != "" {
			taglines = append(taglines, trimmed)
		}
	}
	if len(taglines) == 0 {
		return []string{DefaultTagline}, nil
	}
	return taglines, nil
}

// SelectTagline chooses a tagline and persists the chosen index so consecutive
// calls avoid repeating the same tagline when more than one option exists.
func SelectTagline(taglines []string, statePath string) (string, error) {
	if len(taglines) == 0 {
		taglines = []string{DefaultTagline}
	}

	last := readLastTaglineIndex(statePath)
	idx, tagline := chooseTagline(taglines, last, rand.N)
	if err := writeLastTaglineIndex(statePath, idx); err != nil {
		return tagline, err
	}
	return tagline, nil
}

func readLastTaglineIndex(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return -1
	}
	idx, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return -1
	}
	return idx
}

func writeLastTaglineIndex(path string, idx int) error {
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
	}
	return os.WriteFile(path, []byte(strconv.Itoa(idx)), 0o600)
}

func chooseTagline(taglines []string, last int, pick func(int) int) (int, string) {
	if len(taglines) == 0 {
		return 0, DefaultTagline
	}
	if len(taglines) == 1 {
		return 0, taglines[0]
	}

	for attempts := 0; attempts < len(taglines)*2; attempts++ {
		idx := pick(len(taglines))
		if idx != last {
			return idx, taglines[idx]
		}
	}

	idx := (last + 1) % len(taglines)
	return idx, taglines[idx]
}
