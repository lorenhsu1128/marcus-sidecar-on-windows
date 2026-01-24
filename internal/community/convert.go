package community

import (
	"fmt"
	"math"
	"sort"

	"github.com/marcus/sidecar/internal/styles"
)

// Convert maps a CommunityScheme to a full Sidecar ColorPalette.
func Convert(scheme *CommunityScheme) styles.ColorPalette {
	bg := scheme.Background
	fg := scheme.Foreground
	isDark := Luminance(bg) < 0.5

	bgSecondary := adjustBg(bg, 0.08, isDark)
	bgTertiary := scheme.SelectionBackground
	if bgTertiary == "" || ColorDistance(bg, bgTertiary) < 20 {
		bgTertiary = adjustBg(bg, 0.16, isDark)
	}

	return styles.ColorPalette{
		Primary:   scheme.Blue,
		Secondary: scheme.Cyan,
		Accent:    scheme.Yellow,

		Success: scheme.Green,
		Warning: scheme.Yellow,
		Error:   scheme.Red,
		Info:    scheme.Cyan,

		TextPrimary:   fg,
		TextSecondary: Blend(fg, bg, 0.25),
		TextMuted:     scheme.BrightBlack,
		TextSubtle:    Blend(scheme.BrightBlack, bg, 0.30),
		TextHighlight: scheme.BrightWhite,

		BgPrimary:   bg,
		BgSecondary: bgSecondary,
		BgTertiary:  bgTertiary,
		BgOverlay:   bg + "80",

		BorderNormal: scheme.BrightBlack,
		BorderActive: scheme.Blue,
		BorderMuted:  adjustBg(bg, 0.06, isDark),

		GradientBorderActive: []string{scheme.Blue, scheme.Purple},
		GradientBorderNormal: []string{scheme.BrightBlack, bgTertiary},
		GradientBorderAngle:  30,

		TabStyle:  "gradient",
		TabColors: deriveTabGradient(scheme),

		DiffAddFg:    scheme.Green,
		DiffAddBg:    Blend(bg, scheme.Green, 0.15),
		DiffRemoveFg: scheme.Red,
		DiffRemoveBg: Blend(bg, scheme.Red, 0.15),

		ButtonHover:     scheme.Purple,
		TabTextInactive: scheme.BrightBlack,
		Link:            scheme.BrightBlue,
		ToastSuccessText: contrastText(scheme.Green),
		ToastErrorText:   contrastText(scheme.Red),

		SyntaxTheme:   matchSyntaxTheme(bg),
		MarkdownTheme: markdownTheme(isDark),
	}
}

// PaletteToOverrides serializes a ColorPalette to the override map format for config.json.
func PaletteToOverrides(p styles.ColorPalette) map[string]interface{} {
	m := map[string]interface{}{
		"primary":          p.Primary,
		"secondary":        p.Secondary,
		"accent":           p.Accent,
		"success":          p.Success,
		"warning":          p.Warning,
		"error":            p.Error,
		"info":             p.Info,
		"textPrimary":      p.TextPrimary,
		"textSecondary":    p.TextSecondary,
		"textMuted":        p.TextMuted,
		"textSubtle":       p.TextSubtle,
		"textHighlight":    p.TextHighlight,
		"bgPrimary":        p.BgPrimary,
		"bgSecondary":      p.BgSecondary,
		"bgTertiary":       p.BgTertiary,
		"bgOverlay":        p.BgOverlay,
		"borderNormal":     p.BorderNormal,
		"borderActive":     p.BorderActive,
		"borderMuted":      p.BorderMuted,
		"diffAddFg":        p.DiffAddFg,
		"diffAddBg":        p.DiffAddBg,
		"diffRemoveFg":     p.DiffRemoveFg,
		"diffRemoveBg":     p.DiffRemoveBg,
		"buttonHover":      p.ButtonHover,
		"tabTextInactive":  p.TabTextInactive,
		"link":             p.Link,
		"toastSuccessText": p.ToastSuccessText,
		"toastErrorText":   p.ToastErrorText,
		"syntaxTheme":      p.SyntaxTheme,
		"markdownTheme":    p.MarkdownTheme,
		"tabStyle":         p.TabStyle,
	}

	if len(p.GradientBorderActive) > 0 {
		arr := make([]interface{}, len(p.GradientBorderActive))
		for i, c := range p.GradientBorderActive {
			arr[i] = c
		}
		m["gradientBorderActive"] = arr
	}
	if len(p.GradientBorderNormal) > 0 {
		arr := make([]interface{}, len(p.GradientBorderNormal))
		for i, c := range p.GradientBorderNormal {
			arr[i] = c
		}
		m["gradientBorderNormal"] = arr
	}
	if p.GradientBorderAngle != 0 {
		m["gradientBorderAngle"] = p.GradientBorderAngle
	}
	if len(p.TabColors) > 0 {
		arr := make([]interface{}, len(p.TabColors))
		for i, c := range p.TabColors {
			arr[i] = c
		}
		m["tabColors"] = arr
	}

	return m
}

// adjustBg lightens for dark themes, darkens for light themes.
func adjustBg(bg string, amount float64, isDark bool) string {
	if isDark {
		return Lighten(bg, amount)
	}
	return Darken(bg, amount)
}

// contrastText returns black or white for maximum contrast.
func contrastText(bg string) string {
	if Luminance(bg) > 0.5 {
		return "#000000"
	}
	return "#ffffff"
}

func markdownTheme(isDark bool) string {
	if isDark {
		return "dark"
	}
	return "light"
}

type colorInfo struct {
	hex string
	sat float64
	hue float64
}

// deriveTabGradient picks 3-4 saturated distinct colors sorted by hue.
func deriveTabGradient(scheme *CommunityScheme) []string {
	candidates := []string{
		scheme.Red, scheme.Green, scheme.Yellow, scheme.Blue,
		scheme.Purple, scheme.Cyan,
		scheme.BrightRed, scheme.BrightGreen, scheme.BrightYellow,
		scheme.BrightBlue, scheme.BrightPurple, scheme.BrightCyan,
	}

	// Filter to saturated colors
	var saturated []colorInfo
	for _, c := range candidates {
		if c == "" {
			continue
		}
		s := Saturation(c)
		if s > 0.3 {
			saturated = append(saturated, colorInfo{c, s, HueDegrees(c)})
		}
	}

	if len(saturated) < 3 {
		// Fallback: use primary colors directly
		return []string{scheme.Blue, scheme.Purple, scheme.Cyan}
	}

	// Sort by saturation descending, take top 6
	sort.Slice(saturated, func(i, j int) bool {
		return saturated[i].sat > saturated[j].sat
	})
	if len(saturated) > 6 {
		saturated = saturated[:6]
	}

	// Greedily pick 4 colors with maximum hue distance
	picked := greedyHuePick(saturated, 4)

	// Sort by hue for smooth gradient
	sort.Slice(picked, func(i, j int) bool {
		return HueDegrees(picked[i]) < HueDegrees(picked[j])
	})

	return picked
}

// greedyHuePick selects n colors with maximum mutual hue separation.
func greedyHuePick(candidates []colorInfo, n int) []string {
	if len(candidates) <= n {
		result := make([]string, len(candidates))
		for i, c := range candidates {
			result[i] = c.hex
		}
		return result
	}

	picked := []colorInfo{candidates[0]}
	used := map[int]bool{0: true}

	for len(picked) < n {
		bestIdx := -1
		bestMinDist := -1.0

		for i, c := range candidates {
			if used[i] {
				continue
			}
			minDist := math.MaxFloat64
			for _, p := range picked {
				dist := hueDist(c.hue, p.hue)
				if dist < minDist {
					minDist = dist
				}
			}
			if minDist > bestMinDist {
				bestMinDist = minDist
				bestIdx = i
			}
		}

		if bestIdx < 0 {
			break
		}
		picked = append(picked, candidates[bestIdx])
		used[bestIdx] = true
	}

	result := make([]string, len(picked))
	for i, c := range picked {
		result[i] = c.hex
	}
	return result
}

func hueDist(a, b float64) float64 {
	d := math.Abs(a - b)
	if d > 180 {
		d = 360 - d
	}
	return d
}

// Known Chroma theme backgrounds for matching.
var chromaThemes = map[string]string{
	"monokai":        "#272822",
	"dracula":        "#282a36",
	"nord":           "#2e3440",
	"solarized-dark": "#002b36",
	"github":         "#ffffff",
	"github-dark":    "#24292e",
	"onedark":        "#282c34",
	"gruvbox":        "#282828",
	"catppuccin-mocha": "#1e1e2e",
	"vs":             "#ffffff",
	"solarized-light": "#fdf6e3",
}

// matchSyntaxTheme finds the closest Chroma theme by background color.
func matchSyntaxTheme(bg string) string {
	isDark := Luminance(bg) < 0.5
	best := "monokai"
	if !isDark {
		best = "github"
	}
	bestDist := math.MaxFloat64

	for name, themeBg := range chromaThemes {
		// Only match dark themes to dark, light to light
		themeIsDark := Luminance(themeBg) < 0.5
		if themeIsDark != isDark {
			continue
		}
		dist := ColorDistance(bg, themeBg)
		if dist < bestDist {
			bestDist = dist
			best = name
		}
	}

	// If no close match, use safe defaults
	if bestDist > 100 {
		if isDark {
			return "monokai"
		}
		return "github"
	}
	return best
}

// FormatSchemeInfo returns a brief description for display.
func FormatSchemeInfo(scheme *CommunityScheme) string {
	lum := Luminance(scheme.Background)
	mode := "dark"
	if lum >= 0.5 {
		mode = "light"
	}
	return fmt.Sprintf("%s (%s)", scheme.Name, mode)
}
