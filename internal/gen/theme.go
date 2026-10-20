package gen

import (
	"cmp"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/ozzyozbourne/stellecss/internal/color"
	"github.com/ozzyozbourne/stellecss/internal/config"
)

// Tone is one ramp step: a background and the two text colours tuned for it.
type Tone struct {
	Bg, On, Dim color.OKLCH
}

// Role is one generated palette, such as primary or neutral.
type Role struct {
	Name        string
	Short       string
	Seed        color.OKLCH
	Light, Dark []Tone
}

// Candidate is one ranked mini palette offered in the Theme panel.
type Candidate struct {
	Name  string
	Mode  string
	Stars float64
	// Seeds holds primary, secondary and tertiary seed colours.
	Seeds [3]color.OKLCH
}

// ThemeResult is everything the theme generator derives from the config.
type ThemeResult struct {
	Chips      []color.OKLCH
	Candidates []Candidate
	Chosen     Candidate
	Roles      []Role
}

// harmonies gives each mode's secondary and tertiary hue offsets from primary.
var harmonies = map[string][2]float64{
	"complement": {180, 30},
	"analogous":  {30, -30},
	"triad":      {120, 240},
	"split":      {150, 210},
	"square":     {90, 180},
	"tetradic":   {60, 180},
}

// HarmonyModes orders the modes for the UI.
var HarmonyModes = []string{"complement", "analogous", "triad", "split", "square", "tetradic"}

// Theme builds the ranked candidates and the role ramps.
func Theme(t config.Theme) ThemeResult {
	var r ThemeResult
	for _, s := range t.Seeds {
		if c, err := color.ParseHex(s); err == nil {
			r.Chips = append(r.Chips, c.LCH())
		}
	}
	if len(r.Chips) == 0 {
		r.Chips = []color.OKLCH{color.MustHex("#3f6b78").LCH()}
	}
	r.Candidates = candidates(r.Chips, t.HarmonyMode)
	r.Chosen = r.Candidates[min(t.Palette, len(r.Candidates)-1)]
	p, s, te := r.Chosen.Seeds[0], r.Chosen.Seeds[1], r.Chosen.Seeds[2]
	seeds := []struct {
		name, short string
		seed        color.OKLCH
	}{
		{"primary", "P", p},
		{"secondary", "S", s},
		{"tertiary", "T", te},
		{"neutral", "N", color.OKLCH{L: p.L, C: t.NeutralChroma, H: p.H}},
		{"neutral-variant", "NV", color.OKLCH{L: p.L, C: t.NeutralVariantChroma, H: p.H}},
		{"error", "E", color.OKLCH{L: 0.58, C: 0.19, H: 27}},
		{"caution", "C", color.OKLCH{L: 0.75, C: 0.15, H: 75}},
	}
	for _, sd := range seeds {
		role := Role{Name: sd.name, Short: sd.short, Seed: sd.seed}
		role.Light = ramp(sd.seed, t, false)
		role.Dark = ramp(sd.seed, t, t.Invert)
		r.Roles = append(r.Roles, role)
	}
	return r
}

func candidates(chips []color.OKLCH, mode string) []Candidate {
	modes := HarmonyModes
	if mode != "best" && mode != "" {
		modes = []string{mode}
	}
	// Primary candidates are the most colourful chips; near-greys make poor brands.
	prim := slices.Clone(chips)
	slices.SortStableFunc(prim, func(a, b color.OKLCH) int { return cmp.Compare(b.C, a.C) })
	prim = prim[:min(len(prim), 10)]
	var out []Candidate
	for _, p := range prim {
		for _, m := range modes {
			off := harmonies[m]
			s, es := nearestHue(chips, p.H+off[0], p)
			te, et := nearestHue(chips, p.H+off[1], p)
			stars := 5 - (es+et)/24 - math.Max(0, 0.05-p.C)*40
			stars = math.Round(math.Max(1, math.Min(5, stars))*2) / 2
			out = append(out, Candidate{
				Name:  ColorName(p) + " - " + strings.ToUpper(m[:1]) + m[1:],
				Mode:  m,
				Stars: stars,
				Seeds: [3]color.OKLCH{p, s, te},
			})
		}
	}
	slices.SortStableFunc(out, func(a, b Candidate) int { return cmp.Compare(b.Stars, a.Stars) })
	return out[:min(len(out), 12)]
}

// nearestHue picks the chip closest to the target hue. When no chip is
// within 25° it synthesises one at the target hue with p's lightness and
// chroma, and reports the error as 25° so synthesised palettes rank lower.
func nearestHue(chips []color.OKLCH, target float64, p color.OKLCH) (color.OKLCH, float64) {
	target = math.Mod(target+360, 360)
	best, bestD := color.OKLCH{}, math.Inf(1)
	for _, c := range chips {
		if c.C < 0.02 {
			continue
		}
		if d := color.HueDelta(c.H, target); d < bestD {
			best, bestD = c, d
		}
	}
	if bestD > 25 {
		return color.OKLCH{L: p.L, C: math.Max(p.C, 0.04), H: target}, 25
	}
	return best, bestD
}

func ease(name string, t float64) float64 {
	switch name {
	case "in-quad":
		return t * t
	case "out-quad":
		return 1 - (1-t)*(1-t)
	case "in-out-quad":
		if t < 0.5 {
			return 2 * t * t
		}
		return 1 - math.Pow(-2*t+2, 2)/2
	case "in-out-cubic":
		if t < 0.5 {
			return 4 * t * t * t
		}
		return 1 - math.Pow(-2*t+2, 3)/2
	case "in-out-sine":
		return -(math.Cos(math.Pi*t) - 1) / 2
	}
	return t
}

// toneLightness gives step i's OKLCH lightness, light (i=0) to dark.
func toneLightness(t config.Theme, i int) float64 {
	hi, lo := t.Lightest/100, t.Darkest/100
	trim := (hi - lo) * t.ToneBias / 100
	hi, lo = hi-trim, lo+trim
	return hi - (hi-lo)*ease(t.Easing, float64(i)/float64(t.Steps-1))
}

// ramp builds Steps tones. Chroma peaks mid-ramp and fades toward the
// extremes, where sRGB cannot hold saturated colour anyway. mirrored puts
// the dark end first, for dark mode.
func ramp(seed color.OKLCH, t config.Theme, mirrored bool) []Tone {
	tones := make([]Tone, t.Steps)
	for i := range tones {
		j := i
		if mirrored {
			j = t.Steps - 1 - i
		}
		l := toneLightness(t, j)
		e := float64(j) / float64(t.Steps-1)
		bg := color.OKLCH{L: l, C: seed.C * (0.35 + 0.75*math.Sin(math.Pi*e)), H: seed.H}.InGamut()
		tones[i] = Tone{Bg: bg, On: color.OnColor(bg, t.OnTargetLc), Dim: color.OnColor(bg, t.DimTargetLc)}
	}
	return tones
}

func toneDecls(prefix string, tones []Tone) []Decl {
	var d []Decl
	for i, tn := range tones {
		n := "--" + prefix + "-" + strconv.Itoa(i+1)
		d = append(d,
			Decl{n, tn.Bg.CSS(), tn.Bg.Hex()},
			Decl{n + "-on", tn.On.CSS(), tn.On.Hex()},
			Decl{n + "-dim", tn.Dim.CSS(), tn.Dim.Hex()},
		)
	}
	return d
}

// schemeBlocks emits light and dark declarations with the configured strategy.
func schemeBlocks(scheme string, light, dark []Decl) []Block {
	if scheme == "light-dark" {
		pairs := make([]Decl, len(light))
		for i := range light {
			pairs[i] = Decl{light[i].Name, "light-dark(" + light[i].Value + ", " + dark[i].Value + ")", light[i].Fallback}
		}
		return []Block{{Selector: ":root", Decls: pairs, Raw: "color-scheme: light dark;"}}
	}
	return []Block{
		{Selector: ":root", Decls: light},
		{Media: "@media (prefers-color-scheme: dark)", Selector: ":root:not([data-theme=\"light\"])", Decls: dark},
		{Selector: "[data-theme=\"dark\"]", Decls: dark},
		{Selector: "[data-theme=\"light\"]", Decls: light},
	}
}

func themeSection(th ThemeResult, t config.Theme) Section {
	var light, dark []Decl
	for _, r := range th.Roles {
		light = append(light, toneDecls(r.Name, r.Light)...)
		dark = append(dark, toneDecls(r.Name, r.Dark)...)
	}
	blocks := schemeBlocks(t.Scheme, light, dark)
	if t.Scheme == "light-dark" {
		blocks = append(blocks,
			Block{Selector: "[data-theme=\"light\"]", Raw: "color-scheme: light;"},
			Block{Selector: "[data-theme=\"dark\"]", Raw: "color-scheme: dark;"})
	}
	return Section{ID: "colors-theme", Blocks: blocks}
}

// colorNames are evocative names for palette candidates, matched by nearest OKLab distance.
var colorNames = []struct{ name, hex string }{
	{"Nordic Noir", "#1f3d5c"}, {"Deep Slate Green", "#16343d"}, {"Real Brown", "#6b513f"},
	{"Silverbeet", "#5f6b4f"}, {"Fuchsia Berries", "#4a2a24"}, {"Black Kite", "#2b1a20"},
	{"Ash Tree", "#8a7d6b"}, {"Dolly", "#f0e48a"}, {"Amber Glass", "#a8823a"},
	{"Emerald Succulent", "#4f7563"}, {"Burnt Sienna", "#b5652a"}, {"Ocean Depths", "#1e4d4f"},
	{"Glacier Lake", "#5a9aa0"}, {"Olive Grove", "#857d4c"}, {"Midnight Ink", "#14232e"},
	{"Harbour Blue", "#3f7da3"}, {"Storm Cloud", "#4a5a60"}, {"Moss Bank", "#4a4426"},
	{"Copper Kettle", "#a06628"}, {"Deep Space", "#182a40"}, {"Moon Jellyfish", "#8ec5e8"},
	{"Calm Interlude", "#a7b8c7"}, {"Coral Reef", "#ff7f50"}, {"Rose Quartz", "#f7cac9"},
	{"Lavender Haze", "#b8a6d9"}, {"Royal Plum", "#6c3a7c"}, {"Crimson Tide", "#b0213a"},
	{"Saffron", "#f4c430"}, {"Tangerine Dream", "#f28500"}, {"Mint Julep", "#98d8b8"},
	{"Forest Floor", "#2e4a2b"}, {"Electric Indigo", "#5b3cff"}, {"Cobalt", "#0047ab"},
	{"Teal Lagoon", "#008080"}, {"Sandstone", "#d2b48c"}, {"Charcoal", "#36454f"},
	{"Pebble", "#9e9e96"}, {"Ivory Tower", "#fffff0"}, {"Raspberry", "#c72c5b"},
	{"Sky Harbor", "#4a90d9"}, {"Pine Needle", "#1d5e3b"}, {"Desert Rose", "#c08081"},
}

// ColorName returns the nearest evocative name for c.
func ColorName(c color.OKLCH) string {
	best, bestD := "", math.Inf(1)
	lab := c.Lab()
	for _, n := range colorNames {
		if d := color.Distance(lab, color.MustHex(n.hex).Lab()); d < bestD {
			best, bestD = n.name, d
		}
	}
	return best
}
