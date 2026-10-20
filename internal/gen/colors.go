package gen

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"strconv"

	"github.com/ozzyozbourne/stellecss/internal/color"
	"github.com/ozzyozbourne/stellecss/internal/config"
)

// NamedSwatch is one step of a named colour's relative palette.
type NamedSwatch struct {
	Step int
	Tone Tone
}

// NamedPalette derives the relative steps for one named colour. Step s
// shifts lightness by s·TonePerStep points, hue by s·HuePerStep degrees and
// chroma by s·ChromaPerStep percent, then maps into gamut.
func NamedPalette(n config.Named, base color.OKLCH) []NamedSwatch {
	var out []NamedSwatch
	for s := -n.NegativeSteps; s <= n.PositiveSteps; s++ {
		f := float64(s)
		bg := color.OKLCH{
			L: base.L + f*n.TonePerStep/100,
			C: base.C * (1 + f*n.ChromaPerStep/100),
			H: math.Mod(base.H+f*n.HuePerStep+360, 360),
		}.InGamut()
		out = append(out, NamedSwatch{s, Tone{bg, color.OnColor(bg, n.OnTargetLc), color.OnColor(bg, n.DimTargetLc)}})
	}
	return out
}

func namedSection(n config.Named) Section {
	var d []Decl
	for _, tok := range n.Tokens {
		c, err := color.ParseHex(tok.Color)
		if err != nil {
			continue
		}
		for _, sw := range NamedPalette(n, c.LCH()) {
			name := "--named-" + tok.Name + "-" + strconv.Itoa(sw.Step)
			d = append(d,
				Decl{name, sw.Tone.Bg.CSS(), sw.Tone.Bg.Hex()},
				Decl{name + "-on", sw.Tone.On.CSS(), sw.Tone.On.Hex()},
				Decl{name + "-dim", sw.Tone.Dim.CSS(), sw.Tone.Dim.Hex()})
		}
	}
	return Section{ID: "colors-named", Blocks: []Block{{Selector: ":root", Decls: d}}}
}

// ChartPalettes holds the qualitative series colours and diverging scales.
type ChartPalettes struct {
	Qualitative []Tone
	Diverging   [][]color.OKLCH
}

// Charts derives chart colours from the theme. Qualitative colours start
// from the theme's primary, secondary and tertiary hues, then add chips and
// golden-angle hues, always choosing the hue farthest from those already
// taken. All share one lightness so no series dominates.
func Charts(c config.Charts, th ThemeResult) ChartPalettes {
	var hues []float64
	for _, s := range th.Chosen.Seeds {
		hues = append(hues, s.H)
	}
	pool := slices.Clone(th.Chips)
	slices.SortStableFunc(pool, func(a, b color.OKLCH) int { return cmp.Compare(b.C, a.C) })
	for i := range 25 {
		pool = append(pool, color.OKLCH{H: math.Mod(th.Chosen.Seeds[0].H+float64(i)*137.508, 360), C: 0.1})
	}
	for len(hues) < 25 && len(pool) > 0 {
		bi, bd := 0, -1.0
		for i, p := range pool {
			md := 360.0
			for _, h := range hues {
				md = math.Min(md, color.HueDelta(p.H, h))
			}
			if md > bd {
				bi, bd = i, md
			}
		}
		hues = append(hues, pool[bi].H)
		pool = slices.Delete(pool, bi, bi+1)
	}
	var out ChartPalettes
	for i := range min(c.QualitativeCount, len(hues)) {
		bg := color.OKLCH{L: 0.64, C: 0.13, H: hues[i]}.InGamut()
		out.Qualitative = append(out.Qualitative, Tone{bg, color.OnColor(bg, c.OnTargetLc), color.OnColor(bg, c.DimTargetLc)})
	}
	// Diverging scales pair the most separated qualitative hues, so each
	// scale has a clear contrast across its neutral midpoint.
	type pair struct {
		a, b color.OKLCH
		d    float64
	}
	var pairs []pair
	for i, a := range out.Qualitative {
		for _, b := range out.Qualitative[i+1:] {
			pairs = append(pairs, pair{a.Bg, b.Bg, color.HueDelta(a.Bg.H, b.Bg.H)})
		}
	}
	slices.SortStableFunc(pairs, func(x, y pair) int { return cmp.Compare(y.d, x.d) })
	mid := color.OKLCH{L: 0.96, C: 0.008, H: th.Chosen.Seeds[0].H}.Lab()
	for i := range min(c.DivergingCount, len(pairs)) {
		a := color.OKLCH{L: 0.42, C: pairs[i].a.C, H: pairs[i].a.H}.InGamut().Lab()
		b := color.OKLCH{L: 0.42, C: pairs[i].b.C, H: pairs[i].b.H}.InGamut().Lab()
		steps := make([]color.OKLCH, c.ToneSteps)
		for s := range steps {
			t := float64(s) / float64(c.ToneSteps-1)
			if t <= 0.5 {
				steps[s] = color.Mix(a, mid, t*2).LCH()
			} else {
				steps[s] = color.Mix(mid, b, (t-0.5)*2).LCH()
			}
		}
		out.Diverging = append(out.Diverging, steps)
	}
	return out
}

func chartsSection(c config.Charts, cp ChartPalettes) Section {
	var d []Decl
	if c.Qualitative {
		for i, q := range cp.Qualitative {
			n := "--chart-qualitative-" + strconv.Itoa(i+1)
			d = append(d, Decl{n, q.Bg.CSS(), q.Bg.Hex()}, Decl{n + "-on", q.On.CSS(), q.On.Hex()}, Decl{n + "-dim", q.Dim.CSS(), q.Dim.Hex()})
		}
	}
	for i, steps := range cp.Diverging {
		if i < len(c.Diverging) && !c.Diverging[i] {
			continue
		}
		for s, col := range steps {
			d = append(d, Decl{fmt.Sprintf("--chart-diverging-%d-step-%d", i+1, s+1), col.CSS(), col.Hex()})
		}
	}
	return Section{ID: "colors-charts", Blocks: []Block{{Selector: ":root", Decls: d}}}
}

// CodeVariant is one syntax palette mix: which theme roles colour which tokens.
type CodeVariant struct {
	Name  string
	Stars float64
	// Keyword, String and Function index into the theme roles.
	Keyword, String, Function int
}

// CodePalette is a resolved set of --code-* colours for one scheme.
type CodePalette struct {
	Names  []string
	Colors []color.OKLCH
}

// CodeTokens lists the --code-* suffixes in output order.
var CodeTokens = []string{"bg", "inner-bg", "border", "border-subtle", "fg", "ln", "ln-current", "highlight-line", "selection", "comment", "keyword", "string", "number", "function", "type", "variable", "constant", "operator", "punctuation", "tag", "attr", "property", "regex", "added", "removed", "important", "strong", "emph"}

// CodeVariants ranks syntax role mixes by how well the three accent roles
// separate in hue.
func CodeVariants(th ThemeResult) []CodeVariant {
	accents := []int{0, 1, 2, 5, 6} // primary, secondary, tertiary, error, caution
	var out []CodeVariant
	for _, k := range accents {
		for _, s := range accents {
			for _, f := range accents {
				if k == s || s == f || k == f {
					continue
				}
				hk, hs, hf := th.Roles[k].Seed.H, th.Roles[s].Seed.H, th.Roles[f].Seed.H
				sep := math.Min(color.HueDelta(hk, hs), math.Min(color.HueDelta(hs, hf), color.HueDelta(hk, hf)))
				stars := math.Round(math.Max(1, math.Min(5, sep/24+1))*2) / 2
				out = append(out, CodeVariant{ColorName(th.Roles[k].Light[6].Bg), stars, k, s, f})
			}
		}
	}
	slices.SortStableFunc(out, func(a, b CodeVariant) int { return cmp.Compare(b.Stars, a.Stars) })
	return out[:min(len(out), 8)]
}

// towards walks c's lightness away from bg until |Lc| reaches target.
func towards(c, bg color.OKLCH, target float64) color.OKLCH {
	dir := -1.0
	if bg.L < 0.5 {
		dir = 1
	}
	for i := range 200 {
		x := color.OKLCH{L: c.L + dir*float64(i)*0.005, C: c.C, H: c.H}.InGamut()
		if x.L <= 0 || x.L >= 1 || math.Abs(color.Contrast(x.RGB(), bg.RGB())) >= target {
			return x
		}
	}
	return c
}

// Code resolves the syntax palette for a light (dark=false) or dark surface.
func Code(c config.Code, th ThemeResult, v CodeVariant, dark bool) CodePalette {
	roles := th.Roles
	pick := func(r, step int) color.OKLCH {
		tones := roles[r].Light
		if dark {
			tones = roles[r].Dark
		}
		return tones[min(step, len(tones)-1)].Bg
	}
	n := len(roles[0].Light)
	bg := pick(2, 0) // tertiary's palest tone tints the surface
	bg.C = math.Min(bg.C, 0.04)
	inner := pick(3, 1)
	accent := func(r int) color.OKLCH { return towards(pick(r, n/2), bg, (c.OnTargetLc+c.DimTargetLc)*0.66) }
	cols := map[string]color.OKLCH{
		"bg":             bg,
		"inner-bg":       inner,
		"border":         towards(pick(3, 3), bg, c.SurfaceTargetLc*0.5),
		"border-subtle":  towards(pick(3, 2), bg, c.SurfaceTargetLc*0.3),
		"fg":             towards(pick(3, n-2), bg, c.OnTargetLc),
		"ln":             towards(pick(3, n/2), bg, c.DimTargetLc),
		"ln-current":     towards(pick(3, n-3), bg, c.DimTargetLc*1.8),
		"highlight-line": pick(v.Keyword, 1),
		"selection":      pick(v.Keyword, 2),
		"comment":        towards(pick(4, n/2), bg, c.DimTargetLc*1.5),
		"keyword":        accent(v.Keyword),
		"string":         accent(v.String),
		"number":         accent(6),
		"function":       accent(v.Function),
		"type":           accent(v.String),
		"variable":       towards(pick(3, n-3), bg, c.OnTargetLc*0.8),
		"constant":       accent(6),
		"operator":       towards(pick(4, n-4), bg, c.OnTargetLc*0.7),
		"punctuation":    towards(pick(4, n-5), bg, c.OnTargetLc*0.6),
		"tag":            accent(v.Keyword),
		"attr":           accent(v.Function),
		"property":       accent(v.Function),
		"regex":          accent(v.String),
		"added":          towards(color.OKLCH{L: 0.6, C: 0.14, H: 145}, bg, 60),
		"removed":        towards(color.OKLCH{L: 0.6, C: 0.16, H: 27}, bg, 60),
		"important":      accent(5),
		"strong":         towards(pick(3, n-1), bg, c.OnTargetLc),
		"emph":           towards(pick(3, n-1), bg, c.OnTargetLc),
	}
	p := CodePalette{Names: CodeTokens}
	for _, t := range CodeTokens {
		p.Colors = append(p.Colors, cols[t])
	}
	return p
}

func codeSection(c config.Code, th ThemeResult, scheme string) Section {
	vs := CodeVariants(th)
	v := vs[min(c.Variant, len(vs)-1)]
	light, dark := Code(c, th, v, false), Code(c, th, v, true)
	if c.InvertLightDark {
		light, dark = dark, light
	}
	decls := func(p CodePalette) []Decl {
		var d []Decl
		for i, n := range p.Names {
			d = append(d, Decl{"--code-" + n, p.Colors[i].CSS(), p.Colors[i].Hex()})
		}
		return d
	}
	return Section{ID: "colors-code", Blocks: schemeBlocks(scheme, decls(light), decls(dark))}
}

// GradientStops returns the color-mix stop percentages: k·100/(stops−1),
// rounded half to even as Stellar does (12.5 → 12, 37.5 → 38).
func GradientStops(stops int) []int {
	out := make([]int, stops)
	for k := range out {
		out[k] = int(math.RoundToEven(float64(k) * 100 / float64(stops-1)))
	}
	return out
}

func gradientsSection(g config.Gradients) Section {
	var d []Decl
	for _, it := range g.Items {
		v := "linear-gradient(\n  " + strconv.Itoa(it.Angle) + "deg"
		for _, p := range GradientStops(g.Stops) {
			v += fmt.Sprintf(",\n  color-mix(in srgb, var(--%s) %d%%, var(--%s) %d%%) %d%%", it.From, 100-p, it.To, p, p)
		}
		d = append(d, Decl{Name: "--gradient-" + it.Name, Value: v + "\n)"})
	}
	return Section{ID: "colors-gradients", Blocks: []Block{{Selector: ":root", Decls: d}}}
}
