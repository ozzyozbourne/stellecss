// Package gen turns a [config.Config] into Stellar's CSS sections.
package gen

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/ozzyozbourne/stellecss/internal/color"
)

// Decl is one declaration. Fallback, when set, is written first as a plain
// sRGB value for browsers without oklch() support (bundle.colorFallbacks).
type Decl struct {
	Name, Value, Fallback string
}

// Block is one rule. Media wraps it in an at-rule such as
// "@media (prefers-color-scheme: dark)". Selector ":root" is replaced by the
// bundle namespace when rendered.
type Block struct {
	Media    string
	Selector string
	Decls    []Decl
	// Raw is literal CSS emitted inside the block after Decls.
	Raw string
}

// Section is one /* stellar:<ID>.css */ part of the bundle.
type Section struct {
	ID     string
	Blocks []Block
}

// RenderOptions controls how sections become text.
type RenderOptions struct {
	Namespace      string
	ColorFallbacks bool
	// Hex rewrites every oklch() in a value as #rrggbb; fallbacks are then redundant.
	Hex bool
}

var oklchRe = regexp.MustCompile(`oklch\(([\d.]+)% ([\d.]+) ([\d.]+)\)`)

// toHex rewrites the oklch() colours in v, as rendered by [color.OKLCH.CSS], to #rrggbb.
func toHex(v string) string {
	return oklchRe.ReplaceAllStringFunc(v, func(m string) string {
		p := oklchRe.FindStringSubmatch(m)
		l, _ := strconv.ParseFloat(p[1], 64)
		c, _ := strconv.ParseFloat(p[2], 64)
		h, _ := strconv.ParseFloat(p[3], 64)
		return color.OKLCH{L: l / 100, C: c, H: h}.Hex()
	})
}

// Render writes sections as readable CSS. Minification is esbuild's job.
func Render(sections []Section, o RenderOptions) string {
	ns := o.Namespace
	if ns == "" {
		ns = ":root"
	}
	var b strings.Builder
	for i, s := range sections {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString("/* stellar:" + s.ID + ".css */\n")
		for _, bl := range s.Blocks {
			sel := strings.ReplaceAll(bl.Selector, ":root", ns)
			indent := ""
			if bl.Media != "" {
				b.WriteString(bl.Media + " {\n")
				indent = "  "
			}
			if sel == "" {
				b.WriteString(indentLines(bl.Raw, indent))
			} else {
				b.WriteString(indent + sel + " {\n")
				for _, d := range bl.Decls {
					if o.Hex {
						d.Value, d.Fallback = toHex(d.Value), ""
					}
					if o.ColorFallbacks && d.Fallback != "" {
						b.WriteString(indent + "  " + d.Name + ": " + d.Fallback + ";\n")
					}
					// Multi-line values (font stacks, gradients) continue one level deeper.
					v := strings.ReplaceAll(d.Value, "\n", "\n"+indent+"  ")
					b.WriteString(indent + "  " + d.Name + ": " + v + ";\n")
				}
				if bl.Raw != "" {
					b.WriteString(indentLines(bl.Raw, indent+"  "))
				}
				b.WriteString(indent + "}\n")
			}
			if bl.Media != "" {
				b.WriteString("}\n")
			}
		}
	}
	return b.String()
}

func indentLines(s, indent string) string {
	if s == "" {
		return ""
	}
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	for i, l := range lines {
		if l != "" {
			lines[i] = indent + l
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

// num formats v with at most prec decimals and no trailing zeros.
func num(v float64, prec int) string {
	s := strconv.FormatFloat(v, 'f', prec, 64)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	}
	if s == "-0" {
		return "0"
	}
	return s
}

// CountProperties counts custom property declarations in CSS text.
func CountProperties(css string) int {
	n := 0
	for i := 0; i+1 < len(css); i++ {
		if css[i] == '-' && css[i+1] == '-' && (i == 0 || strings.ContainsRune("{; \n\t", rune(css[i-1]))) {
			j := i + 2
			for j < len(css) && css[j] != ':' && css[j] != ';' && css[j] != '}' && css[j] != ')' && css[j] != ',' {
				j++
			}
			if j < len(css) && css[j] == ':' {
				n++
			}
			i = j
		}
	}
	return n
}
