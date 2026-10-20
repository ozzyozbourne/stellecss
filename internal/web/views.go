package web

import (
	"encoding/json"
	"fmt"
	"html"
	"html/template"
	"image"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ozzyozbourne/stellecss/internal/bundle"
	"github.com/ozzyozbourne/stellecss/internal/color"
	"github.com/ozzyozbourne/stellecss/internal/config"
	"github.com/ozzyozbourne/stellecss/internal/gen"
)

// view is the data every editor template receives. Control helpers are
// methods so templates can bind an input with only a label and a path.
type view struct {
	Groups  []group
	Sec     section
	Cfg     *config.Config
	B       *bundle.Bundle
	Abs     string
	Version string
	cfgDir  string
	tree    map[string]any
}

func newView(sec section, c *config.Config, b *bundle.Bundle, cfgPath string) *view {
	abs, _ := filepath.Abs(filepath.Join(filepath.Dir(cfgPath), c.Bundle.Output))
	return &view{Groups: groups, Sec: sec, Cfg: c, B: b, Abs: abs, Version: version, cfgDir: filepath.Dir(cfgPath), tree: toTree(c)}
}

// Signals is the initial data-signals value: the whole config under cfg, so
// every @post carries a complete config the server can validate.
func (v *view) Signals() string {
	b, _ := json.Marshal(map[string]any{"cfg": v.tree})
	return string(b)
}

func (v *view) val(path string) any {
	x, err := lookup(v.tree, path)
	if err != nil {
		panic(fmt.Sprintf("template path %s: %v", path, err))
	}
	return x
}

func attr(s string) string { return html.EscapeString(s) }

func fmtVal(x any) string {
	switch t := x.(type) {
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case nil:
		return ""
	}
	return fmt.Sprint(x)
}

func fieldID(path string) string { return "f-" + strings.ReplaceAll(path, ".", "-") }

// Num renders a labelled number input bound to cfg.<path>.
func (v *view) Num(label, path string, step string) template.HTML {
	return v.input(label, path, "number", `step="`+attr(step)+`"`)
}

// Text renders a labelled text input bound to cfg.<path>.
func (v *view) Text(label, path string) template.HTML {
	return v.input(label, path, "text", "")
}

// Color renders a colour picker bound to cfg.<path>.
func (v *view) Color(label, path string) template.HTML {
	return v.input(label, path, "color", "")
}

func (v *view) input(label, path, typ, extra string) template.HTML {
	id := fieldID(path)
	lab := ""
	if label != "" {
		lab = `<label for="` + id + `">` + attr(label) + `</label>`
	} else {
		extra += ` aria-label="` + attr(path[strings.LastIndexByte(path, '.')+1:]) + `"`
	}
	return template.HTML(`<div class="field">` + lab + `<input id="` + id + `" type="` + typ + `" value="` + attr(fmtVal(v.val(path))) + `" data-bind="cfg.` + path + `" ` + extra + `></div>`)
}

// Range renders a slider with its live value.
func (v *view) Range(label, path, lo, hi, step, unit string) template.HTML {
	id := fieldID(path)
	return template.HTML(`<div class="field range"><label for="` + id + `">` + attr(label) + `</label><input id="` + id + `" type="range" min="` + lo + `" max="` + hi + `" step="` + step + `" value="` + attr(fmtVal(v.val(path))) + `" data-bind="cfg.` + path + `"><output data-text="$cfg.` + path + ` + '` + unit + `'">` + attr(fmtVal(v.val(path))) + unit + `</output></div>`)
}

// Select renders a select; options are "value" or "value=Label" strings.
func (v *view) Select(label, path string, options ...string) template.HTML {
	id := fieldID(path)
	cur := fmtVal(v.val(path))
	var b strings.Builder
	b.WriteString(`<div class="field"><label for="` + id + `">` + attr(label) + `</label><select id="` + id + `" data-bind="cfg.` + path + `">`)
	for _, o := range options {
		val, lab, ok := strings.Cut(o, "=")
		if !ok {
			lab = val
		}
		sel := ""
		if val == cur {
			sel = " selected"
		}
		b.WriteString(`<option value="` + attr(val) + `"` + sel + `>` + attr(lab) + `</option>`)
	}
	b.WriteString(`</select></div>`)
	return template.HTML(b.String())
}

// Toggle renders a switch-style checkbox with an optional description.
func (v *view) Toggle(label, desc, path string) template.HTML {
	id := fieldID(path)
	checked := ""
	if b, _ := v.val(path).(bool); b {
		checked = " checked"
	}
	d := ""
	if desc != "" {
		d = `<small>` + attr(desc) + `</small>`
	}
	return template.HTML(`<div class="toggle"><label for="` + id + `">` + attr(label) + d + `</label><input id="` + id + `" type="checkbox" role="switch"` + checked + ` data-bind="cfg.` + path + `"></div>`)
}

// RatioSelect is a [view.Select] of the musical scales, e.g. "Minor Third (1.2)".
func (v *view) RatioSelect(label, path string) template.HTML {
	var opts []string
	for _, n := range config.RatioNames {
		opts = append(opts, n+"="+n+" ("+strconv.FormatFloat(config.Ratios[n], 'f', -1, 64)+")")
	}
	return v.Select(label, path, opts...)
}

// SectionCSS is this section's slice of the readable bundle.
func (v *view) SectionCSS() string {
	if v.Sec.Group == "export" {
		return ""
	}
	id := v.Sec.Group + "-" + v.Sec.Slug
	if v.Sec.Group == "motion" {
		id = "motion"
	}
	for _, s := range v.B.Result.Sections {
		if s.ID == id {
			return gen.Render([]gen.Section{s}, gen.RenderOptions{Namespace: v.Cfg.Bundle.Namespace, ColorFallbacks: v.Cfg.Bundle.ColorFallbacks, Hex: v.Cfg.Bundle.ColorFormat == "hex"})
		}
	}
	return "/* section disabled */"
}

// scaleRow is one fluid token in the size and font-size previews.
type scaleRow struct {
	Name     string
	Min, Max float64
}

// Expr is a Datastar expression giving the token's rem value at the preview
// width, so the slider updates rows without a round trip.
func (r scaleRow) Expr(v config.Viewport) string {
	return fmt.Sprintf("(%g + %g * Math.min(1, Math.max(0, ($_pw - %g) / %g)))", r.Min, r.Max-r.Min, v.Min, v.Max-v.Min)
}

func (v *view) SizeRows() []scaleRow { return rows(gen.Steps("size", v.Cfg.General.Size)) }
func (v *view) FontRows() []scaleRow { return rows(gen.Steps("font-size", v.Cfg.Fonts.Sizes)) }
func (v *view) VP() config.Viewport  { return v.Cfg.General.Viewport }

func rows(steps []gen.Step) []scaleRow {
	out := make([]scaleRow, len(steps))
	for i, s := range steps {
		out[i] = scaleRow{s.Name, s.Min, s.Max}
	}
	return out
}

func (v *view) Theme() gen.ThemeResult { return v.B.Result.Theme }
func (v *view) Charts() gen.ChartPalettes {
	return v.B.Result.Charts
}
func (v *view) CodeVariants() []gen.CodeVariant {
	if len(v.B.Result.Theme.Roles) == 0 {
		return nil
	}
	return gen.CodeVariants(v.B.Result.Theme)
}

// Named returns each token's swatches for the Named preview.
func (v *view) Named() []namedRow {
	var out []namedRow
	for _, t := range v.Cfg.Colors.Named.Tokens {
		c, err := color.ParseHex(t.Color)
		if err != nil {
			continue
		}
		out = append(out, namedRow{t.Name, gen.NamedPalette(v.Cfg.Colors.Named, c.LCH())})
	}
	return out
}

type namedRow struct {
	Name     string
	Swatches []gen.NamedSwatch
}

func (v *view) Stops() []int { return gen.GradientStops(v.Cfg.Colors.Gradients.Stops) }

// WheelPos places a colour on the theme's hue/chroma wheel (radius 100).
func WheelPos(c color.OKLCH) [2]float64 {
	r := math.Min(c.C/0.25, 1) * 92
	a := c.H * math.Pi / 180
	return [2]float64{math.Round(r*math.Cos(a)*100) / 100, math.Round(-r*math.Sin(a)*100) / 100}
}

// Stars renders a 0–5 rating in half stars.
func Stars(s float64) string {
	full := int(s)
	out := strings.Repeat("★", full)
	if s-float64(full) >= 0.5 {
		out += "⯪"
		full++
	}
	return out + strings.Repeat("☆", 5-full)
}

func (v *view) KiB(n int) string { return bundle.KiB(n) }

// Minify names the minify flags the way the Bundle preview lists them.
func (v *view) Minify() string {
	b := v.Cfg.Bundle
	var parts []string
	if b.MinifyWhitespace {
		parts = append(parts, "Whitespace")
	}
	if b.MinifySyntax {
		parts = append(parts, "Syntax")
	}
	if b.MinifyIdentifiers {
		parts = append(parts, "Identifiers")
	}
	if len(parts) == 0 {
		return "None"
	}
	return strings.Join(parts, ", ")
}

// CSSHead is the first lines of the bundle for the Bundle preview.
func (v *view) CSSHead() string {
	s := string(v.B.CSS)
	if len(s) > 6000 {
		s = s[:6000] + "\n…"
	}
	return s
}

var funcs = template.FuncMap{
	"css":     func(c color.OKLCH) template.CSS { return template.CSS(c.CSS()) },
	"hex":     func(c color.OKLCH) string { return c.Hex() },
	"pct":     func(f float64) string { return strconv.FormatFloat(f*100, 'f', 1, 64) + "%" },
	"wheel":   WheelPos,
	"stars":   Stars,
	"add":     func(a, b int) int { return a + b },
	"js":      func(s string) template.JS { return template.JS(s) },
	"fmt":     func(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) },
	"modes":   func() []string { return append([]string{"best"}, gen.HarmonyModes...) },
	"title":   func(s string) string { return strings.ToUpper(s[:1]) + s[1:] },
	"presets": func() []string { return config.FontPresetNames },
	"seq": func(n int) []int {
		out := make([]int, n)
		for i := range out {
			out[i] = i
		}
		return out
	},
	"stack": func(s []string) string { return strings.Join(s, ", ") },
	"list":  func(xs ...any) []any { return xs },
	"wheelEdge": func(i int) [2]float64 {
		a := float64(i) * 10 * math.Pi / 180
		return [2]float64{math.Round(96*math.Cos(a)*100) / 100, math.Round(-96*math.Sin(a)*100) / 100}
	},
	"wheelHue": func(i int) string { return color.OKLCH{L: 0.72, C: 0.13, H: float64(i)*10 + 5}.InGamut().Hex() },
	"dict": func(kv ...any) map[string]any {
		m := map[string]any{}
		for i := 0; i+1 < len(kv); i += 2 {
			m[kv[i].(string)] = kv[i+1]
		}
		return m
	},
}

// Part renders this section's "settings/<group>/<slug>" or
// "preview/<group>/<slug>" template.
func (v *view) Part(kind string) (template.HTML, error) {
	var b strings.Builder
	if err := tmpl.ExecuteTemplate(&b, kind+"/"+v.Sec.Key(), v); err != nil {
		return "", err
	}
	return template.HTML(b.String()), nil
}

// ImageURL is the theme's source image; without an uploaded one it is the
// bundled nebula the default seeds came from.
func (v *view) ImageURL() string {
	if v.Cfg.Colors.Theme.Image != "" {
		return "/theme-image?v=" + v.B.Hash
	}
	return "/static/img/nebula.jpg"
}

type chipDot struct {
	X, Y  string
	Color color.OKLCH
}

// ChipDots places each extracted chip where it occurs in the image.
func (v *view) ChipDots() []chipDot {
	var r io.ReadCloser
	var err error
	if v.Cfg.Colors.Theme.Image != "" {
		r, err = os.Open(filepath.Join(v.cfgDir, filepath.Base(v.Cfg.Colors.Theme.Image)))
	} else {
		r, err = assets.Open("static/img/nebula.jpg")
	}
	if err != nil {
		return nil
	}
	defer r.Close()
	img, _, err := image.Decode(r)
	if err != nil {
		return nil
	}
	chips := v.B.Result.Theme.Chips
	rgb := make([]color.RGB, len(chips))
	for i, c := range chips {
		rgb[i] = c.RGB()
	}
	var out []chipDot
	for i, p := range color.Locate(img, rgb) {
		out = append(out, chipDot{strconv.FormatFloat(p[0]*100, 'f', 1, 64) + "%", strconv.FormatFloat(p[1]*100, 'f', 1, 64) + "%", chips[i]})
	}
	return out
}

// Roles pairs each theme role with its tones for the scheme being previewed.
func (v *view) Roles() []gen.Role { return v.B.Result.Theme.Roles }

// CodeTokens lists the --code-* token suffixes.
func (v *view) CodeTokens() []string { return gen.CodeTokens }
