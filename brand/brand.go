package brand

import (
	"bufio"
	"embed"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

//go:embed assets/*
var assetsFS embed.FS

type ColorMode string

const (
	ColorModeTruecolor ColorMode = "truecolor"
	ColorMode256       ColorMode = "256"
	ColorModeMono      ColorMode = "mono"
)

type Variant string

const (
	VariantFull   Variant = "full"
	VariantMedium Variant = "medium"
	VariantShort  Variant = "short"
)

type Environment struct {
	IsTTY     bool
	NoColor   bool
	Term      string
	ColorTerm string
}

type RGB struct {
	R int
	G int
	B int
}

type ColorSpec struct {
	Hex     string
	Ansi256 int
	RGB     RGB
}

type Palette struct {
	IVORY         ColorSpec
	INDIGO        ColorSpec
	INDIGO_BRIGHT ColorSpec
	INDIGO_GLOW   ColorSpec
}

type Assets struct {
	FullLines []string
	Medium    string
	Short     string
	Palette   Palette
}

var (
	loadOnce    sync.Once
	loaded      Assets
	loadedError error
)

func MustAssets() Assets {
	loadOnce.Do(func() {
		loaded, loadedError = loadAssets()
	})
	if loadedError != nil {
		panic(loadedError)
	}
	return loaded
}

func DetectColorMode(env Environment) ColorMode {
	if !env.IsTTY || env.NoColor {
		return ColorModeMono
	}

	colorTerm := strings.ToLower(env.ColorTerm)
	if strings.Contains(colorTerm, "truecolor") || strings.Contains(colorTerm, "24bit") {
		return ColorModeTruecolor
	}

	if strings.Contains(strings.ToLower(env.Term), "256color") {
		return ColorMode256
	}

	return ColorModeMono
}

func Render(variant Variant, mode ColorMode) string {
	assets := MustAssets()

	switch variant {
	case VariantFull:
		return renderFull(assets, mode)
	case VariantShort:
		return renderSingle(assets.Short, assets.Palette.INDIGO, mode)
	case VariantMedium:
		fallthrough
	default:
		return renderMedium(assets, mode)
	}
}

func loadAssets() (Assets, error) {
	full, err := readLines("assets/full.txt")
	if err != nil {
		return Assets{}, err
	}

	medium, err := readTrimmed("assets/medium.txt")
	if err != nil {
		return Assets{}, err
	}

	short, err := readTrimmed("assets/short.txt")
	if err != nil {
		return Assets{}, err
	}

	palette, err := readPalette("assets/palette.env")
	if err != nil {
		return Assets{}, err
	}

	return Assets{FullLines: full, Medium: medium, Short: short, Palette: palette}, nil
}

func readLines(name string) ([]string, error) {
	raw, err := assetsFS.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", name, err)
	}

	trimmed := strings.TrimRight(string(raw), "\n")
	return strings.Split(trimmed, "\n"), nil
}

func readTrimmed(name string) (string, error) {
	raw, err := assetsFS.ReadFile(name)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", name, err)
	}
	return strings.TrimSpace(string(raw)), nil
}

func readPalette(name string) (Palette, error) {
	raw, err := assetsFS.ReadFile(name)
	if err != nil {
		return Palette{}, fmt.Errorf("read %s: %w", name, err)
	}
	return parsePalette(raw)
}

func parsePalette(raw []byte) (Palette, error) {

	entries := map[string]ColorSpec{}
	scanner := bufio.NewScanner(strings.NewReader(string(raw)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return Palette{}, fmt.Errorf("invalid palette line %q", line)
		}

		vals := strings.Split(parts[1], ",")
		if len(vals) != 5 {
			return Palette{}, fmt.Errorf("invalid palette values for %s", parts[0])
		}

		ansi256, err := strconv.Atoi(vals[1])
		if err != nil {
			return Palette{}, fmt.Errorf("invalid ansi256 for %s: %w", parts[0], err)
		}
		r, err := strconv.Atoi(vals[2])
		if err != nil {
			return Palette{}, fmt.Errorf("invalid red channel for %s: %w", parts[0], err)
		}
		g, err := strconv.Atoi(vals[3])
		if err != nil {
			return Palette{}, fmt.Errorf("invalid green channel for %s: %w", parts[0], err)
		}
		b, err := strconv.Atoi(vals[4])
		if err != nil {
			return Palette{}, fmt.Errorf("invalid blue channel for %s: %w", parts[0], err)
		}

		entries[parts[0]] = ColorSpec{Hex: vals[0], Ansi256: ansi256, RGB: RGB{R: r, G: g, B: b}}
	}

	if err := scanner.Err(); err != nil {
		return Palette{}, fmt.Errorf("scan palette: %w", err)
	}

	palette := Palette{
		IVORY:         entries["IVORY"],
		INDIGO:        entries["INDIGO"],
		INDIGO_BRIGHT: entries["INDIGO_BRIGHT"],
		INDIGO_GLOW:   entries["INDIGO_GLOW"],
	}

	if err := validatePalette(palette); err != nil {
		return Palette{}, err
	}

	return palette, nil
}

func validatePalette(p Palette) error {
	required := map[string]ColorSpec{
		"IVORY":         p.IVORY,
		"INDIGO":        p.INDIGO,
		"INDIGO_BRIGHT": p.INDIGO_BRIGHT,
		"INDIGO_GLOW":   p.INDIGO_GLOW,
	}

	for name, spec := range required {
		if spec.Hex == "" {
			return fmt.Errorf("missing required color entry %s", name)
		}
	}

	return nil
}

func renderMedium(assets Assets, mode ColorMode) string {
	if mode == ColorModeMono {
		return "OpenCode *ADVANCE*"
	}

	return renderSingle("OpenCode", assets.Palette.IVORY, mode) + " " + renderSingle("ADVANCE", assets.Palette.INDIGO, mode)
}

func renderFull(assets Assets, mode ColorMode) string {
	if mode == ColorModeMono {
		return strings.Join(assets.FullLines, "\n")
	}

	lines := make([]string, 0, len(assets.FullLines))
	for _, line := range assets.FullLines {
		left, right, ok := splitWordmarkLine(line)
		if !ok {
			lines = append(lines, renderSingle(line, assets.Palette.IVORY, mode))
			continue
		}

		lines = append(lines, renderSingle(left, assets.Palette.IVORY, mode)+"   "+renderSingle(right, assets.Palette.INDIGO, mode))
	}

	return strings.Join(lines, "\n")
}

func splitWordmarkLine(line string) (left string, right string, ok bool) {
	parts := strings.SplitN(line, "   ", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func renderSingle(text string, color ColorSpec, mode ColorMode) string {
	if mode == ColorModeMono {
		return text
	}

	return colorSequence(color, mode) + text + "\x1b[0m"
}

func colorSequence(color ColorSpec, mode ColorMode) string {
	switch mode {
	case ColorModeTruecolor:
		return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", color.RGB.R, color.RGB.G, color.RGB.B)
	case ColorMode256:
		return fmt.Sprintf("\x1b[38;5;%dm", color.Ansi256)
	default:
		return ""
	}
}
