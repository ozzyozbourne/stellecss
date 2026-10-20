package gen

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ozzyozbourne/stellecss/internal/config"
)

// Result is a generated bundle before minification, plus the derived data the
// editor previews.
type Result struct {
	Imports  []string
	Sections []Section
	Theme    ThemeResult
	Charts   ChartPalettes
	CSS      string
}

// Build generates every enabled section in bundle order. Size and font-size
// tokens need the viewport, so they are skipped when it is disabled; charts
// and code are derived from the theme, so they are skipped without it.
func Build(c *config.Config) *Result {
	r := &Result{}
	g, col, f, sf := c.General, c.Colors, c.Fonts, c.Surface
	vpOn := !g.Viewport.Disabled
	if c.Bundle.FontImports && !f.Import.Disabled {
		r.Imports = f.Import.URLs
	}
	add := func(on bool, s func() Section) {
		if on {
			r.Sections = append(r.Sections, s())
		}
	}
	add(!g.Normalize.Disabled, func() Section { return normalizeSection(g.Normalize) })
	if !col.Theme.Disabled {
		r.Theme = Theme(col.Theme)
		r.Sections = append(r.Sections, themeSection(r.Theme, col.Theme))
	}
	add(!col.Named.Disabled, func() Section { return namedSection(col.Named) })
	if !col.Theme.Disabled && !col.Charts.Disabled {
		r.Charts = Charts(col.Charts, r.Theme)
		r.Sections = append(r.Sections, chartsSection(col.Charts, r.Charts))
	}
	add(!col.Theme.Disabled && !col.Code.Disabled, func() Section { return codeSection(col.Code, r.Theme, col.Theme.Scheme) })
	add(!col.Gradients.Disabled, func() Section { return gradientsSection(col.Gradients) })
	add(!f.Families.Disabled, func() Section { return familiesSection(f.Families) })
	add(!f.Weights.Disabled, func() Section {
		return numberSection("fonts-weights", "font-weight", f.Weights.Weights, "")
	})
	add(vpOn, func() Section { return viewportSection(g.Viewport) })
	add(vpOn && !f.Sizes.Disabled, func() Section { return scaleSection("fonts-sizes", "font-size", f.Sizes, g.Viewport) })
	add(!f.LetterSpacing.Disabled, func() Section {
		return numberSection("fonts-letter-spacing", "letter-spacing", f.LetterSpacing.Values, "em")
	})
	add(!f.LineHeights.Disabled, func() Section {
		return numberSection("fonts-line-heights", "line-height", f.LineHeights.Values, "")
	})
	add(vpOn && !g.Size.Disabled, func() Section { return scaleSection("general-size", "size", g.Size, g.Viewport) })
	add(!g.AspectRatio.Disabled, func() Section { return aspectSection(g.AspectRatio) })
	add(!g.ZIndexes.Disabled, func() Section { return zSection(g.ZIndexes) })
	add(!sf.Radius.Disabled, func() Section { return radiusSection(sf.Radius) })
	add(!sf.Borders.Disabled, func() Section {
		return numberSection("surface-borders", "border-width", sf.Borders.Values, "px")
	})
	add(!sf.Shadows.Disabled, func() Section { return shadowsSection(sf.Shadows, col.Theme.Scheme) })
	add(!c.Motion.Disabled, func() Section { return motionSection(c.Motion, c.Bundle.ReducedMotion) })

	var b strings.Builder
	for _, u := range r.Imports {
		fmt.Fprintf(&b, "@import %q;\n", u)
	}
	if b.Len() > 0 {
		b.WriteByte('\n')
	}
	b.WriteString(Render(r.Sections, RenderOptions{Namespace: c.Bundle.Namespace, ColorFallbacks: c.Bundle.ColorFallbacks}))
	r.CSS = b.String()
	return r
}

func viewportSection(v config.Viewport) Section {
	return Section{ID: "general-viewport", Blocks: []Block{{Selector: ":root", Decls: []Decl{
		{Name: "--viewport-min", Value: num(v.Min, 3) + "px"},
		{Name: "--viewport-max", Value: num(v.Max, 3) + "px"},
		{Name: "--viewport-base-font-size", Value: num(v.BaseFontSize, 3) + "px"},
	}, Raw: "font-size: var(--viewport-base-font-size);"}}}
}

func normalizeSection(n config.Normalize) Section {
	var raw []string
	if n.BoxSizing {
		raw = append(raw, "*, *::before, *::after {\n  box-sizing: border-box;\n}")
	}
	if n.ResetMargins {
		raw = append(raw, "* {\n  margin: 0;\n}")
	}
	if n.MediaBlock {
		raw = append(raw, "img, picture, video, canvas, svg {\n  display: block;\n  max-inline-size: 100%;\n}")
	}
	if n.InheritFonts {
		raw = append(raw, "input, button, textarea, select {\n  font: inherit;\n}")
	}
	if n.TextWrap {
		raw = append(raw, "h1, h2, h3, h4, h5, h6 {\n  text-wrap: balance;\n}\np {\n  text-wrap: pretty;\n}")
	}
	if n.SmoothScroll {
		raw = append(raw, "@media (prefers-reduced-motion: no-preference) {\n  :root {\n    scroll-behavior: smooth;\n  }\n}")
	}
	return Section{ID: "general-normalize", Blocks: []Block{{Raw: strings.Join(raw, "\n")}}}
}

func familiesSection(f config.FontFamilies) Section {
	var d []Decl
	for _, fam := range f.Families {
		d = append(d, Decl{Name: "--font-" + fam.Name, Value: "\n" + strings.Join(fam.Stack, ",\n")})
	}
	return Section{ID: "fonts-families", Blocks: []Block{{Selector: ":root", Decls: d}}}
}

func numberSection(id, prefix string, vals []config.NamedNumber, unit string) Section {
	var d []Decl
	for _, v := range vals {
		s := num(v.Value, 4)
		if v.Value != 0 {
			s += unit
		}
		d = append(d, Decl{Name: "--" + prefix + "-" + v.Name, Value: s})
	}
	return Section{ID: id, Blocks: []Block{{Selector: ":root", Decls: d}}}
}

func radiusSection(r config.Radius) Section {
	var d []Decl
	for _, v := range r.Values {
		val := num(v.Value, 4) + "rem"
		switch {
		case v.Value >= 9999:
			val = "9999px"
		case v.Value == 0:
			val = "0"
		}
		d = append(d, Decl{Name: "--radius-" + v.Name, Value: val})
	}
	d = append(d, Decl{Name: "--radius-round", Value: "50%"})
	return Section{ID: "surface-radius", Blocks: []Block{{Selector: ":root", Decls: d}}}
}

func aspectSection(a config.AspectRatio) Section {
	var d []Decl
	for _, r := range a.Ratios {
		d = append(d, Decl{Name: "--aspect-ratio-" + r.Name, Value: num(r.Width/r.Height, 4)})
	}
	return Section{ID: "general-aspect-ratio", Blocks: []Block{{Selector: ":root", Decls: d}}}
}

func zSection(z config.ZIndexes) Section {
	var d []Decl
	for _, l := range z.Levels {
		d = append(d, Decl{Name: "--z-" + l.Name, Value: strconv.Itoa(l.Value)})
	}
	return Section{ID: "general-zindex", Blocks: []Block{{Selector: ":root", Decls: d}}}
}

// shadowsSection layers progressively larger, softer shadows. The tint and
// strength are themselves tokens so dark mode only swaps those two.
func shadowsSection(s config.Shadows, scheme string) Section {
	d := []Decl{}
	for i := 1; i <= s.Steps; i++ {
		f := float64(i)
		layers := []string{}
		for l := 1; l <= min(i+1, 4); l++ {
			lf := float64(l)
			layers = append(layers, fmt.Sprintf("0 %spx %spx %spx color-mix(in oklch, var(--shadow-color) calc(var(--shadow-strength) + %s%%), transparent)",
				num(lf*f*0.6, 2), num(lf*f*1.6, 2), num(-lf*0.4, 2), num(9-lf*2, 1)))
		}
		d = append(d, Decl{Name: "--shadow-" + strconv.Itoa(i), Value: strings.Join(layers, ",\n")})
	}
	light := []Decl{
		{Name: "--shadow-color", Value: "oklch(22% 0.03 " + num(s.Hue, 2) + ")"},
		{Name: "--shadow-strength", Value: num(s.Strength, 2) + "%"},
	}
	dark := []Decl{
		{Name: "--shadow-color", Value: "oklch(6% 0.02 " + num(s.Hue, 2) + ")"},
		{Name: "--shadow-strength", Value: num(min(s.Strength*2.6, 90), 2) + "%"},
	}
	var blocks []Block
	if scheme == "light-dark" {
		// light-dark() only accepts colours, so the strength stays one value
		// and the colour pair carries the dark-mode weight.
		blocks = []Block{{Selector: ":root", Decls: []Decl{
			{Name: "--shadow-color", Value: "light-dark(" + light[0].Value + ", " + dark[0].Value + ")"},
			light[1],
		}}}
	} else {
		blocks = schemeBlocks(scheme, light, dark)
	}
	blocks = append(blocks, Block{Selector: ":root", Decls: d})
	return Section{ID: "surface-shadows", Blocks: blocks}
}

func motionSection(m config.Motion, reduced bool) Section {
	var d, zero []Decl
	for _, v := range m.Durations {
		d = append(d, Decl{Name: "--anim-duration-" + v.Name, Value: num(v.Value, 2) + "ms"})
		zero = append(zero, Decl{Name: "--anim-duration-" + v.Name, Value: "0ms"})
	}
	for _, e := range m.Easings {
		d = append(d, Decl{Name: "--ease-" + e.Name, Value: e.Value})
	}
	blocks := []Block{{Selector: ":root", Decls: d}}
	if reduced {
		blocks = append(blocks, Block{Media: "@media (prefers-reduced-motion: reduce)", Selector: ":root", Decls: zero})
	}
	return Section{ID: "motion", Blocks: blocks}
}
