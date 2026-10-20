package web

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/ozzyozbourne/stellecss/internal/config"
)

// section is one editor page. Path is the dotted JSON path of the config
// object it edits; reset and the "Disable" toggle both act on that object.
type section struct {
	Group, Slug, Title, Desc, Usage string
	Path                            string
	// CanDisable is false for sections without a disabled flag (Bundle).
	CanDisable bool
}

func (s section) URL() string { return "/" + s.Group + "/" + s.Slug }
func (s section) Key() string { return s.Group + "/" + s.Slug }

type group struct {
	Slug, Title, Icon string
	Sections          []section
}

var groups = []group{
	{"general", "General", "general", []section{
		{"general", "viewport", "Viewport", "Set the viewport range every fluid token interpolates across.", "Fluid sizes and font sizes reach their minimum at the min viewport and their maximum at the max viewport. The base font size sets how many pixels one rem is.", "general.viewport", true},
		{"general", "size", "Size", "Build a fluid sizing scale that responds to the viewport range.", "Use the size scale anywhere you would reach for Tailwind-style utilities like h-2, w-6, or p-12. The size tokens are general-purpose: spacing, gaps, widths, heights, borders, and any other size-related value can all reference the same scale.", "general.size", true},
		{"general", "zindex", "Z-index", "Name the stacking layers so components agree on what sits above what.", "Use z-index tokens instead of magic numbers: z-index: var(--z-modal). Gaps between layers leave room for local stacking.", "general.zindexes", true},
		{"general", "aspect-ratio", "Aspect ratio", "Common media proportions as unitless ratios.", "aspect-ratio: var(--aspect-ratio-widescreen) keeps embeds and thumbnails in shape at every width.", "general.aspectRatio", true},
		{"general", "normalize", "Normalize", "Opinionated base rules that make the tokens predictable.", "Normalize rules ship at the top of the bundle. Turn off any group your project already resets.", "general.normalize", true},
	}},
	{"colors", "Colors", "colors", []section{
		{"colors", "theme", "Theme", "Extract a palette from an image, choose a harmony and generate accessible tone ramps.", "Each role gets numbered tones with matching -on (text) and -dim (secondary text) colours that meet the APCA targets: background: var(--primary-3); color: var(--primary-3-on).", "colors.theme", true},
		{"colors", "named", "Named", "Brand and language colours with relative steps.", "Named colours keep brand marks exact while the steps give hover and pressed states: var(--named-go-0), var(--named-go-1).", "colors.named", true},
		{"colors", "charts", "Charts", "Qualitative series colours and diverging scales derived from the theme.", "Qualitative colours share one lightness so no series dominates. Diverging scales run between two well separated hues through a neutral midpoint.", "colors.charts", true},
		{"colors", "code", "Code", "Syntax highlighting colours that sit on your theme.", "Map highlighter classes to --code-* tokens; the palette follows light and dark mode with the theme.", "colors.code", true},
		{"colors", "gradients", "Gradients", "Gradients between theme tokens, mixed in sRGB at fixed stops.", "background: var(--gradient-brand-surface). Because the stops reference tokens, gradients follow the theme and dark mode.", "colors.gradients", true},
	}},
	{"fonts", "Fonts", "fonts", []section{
		{"fonts", "families", "Font families", "Font stacks by classification, plus presets matching popular UI libraries.", "font-family: var(--font-didone). Stacks fall back through installed system fonts, so nothing needs downloading unless you import it.", "fonts.families", true},
		{"fonts", "import", "Font import", "Stylesheets imported at the top of the bundle.", "Imports are emitted only when the Bundle panel's font imports option is on.", "fonts.import", true},
		{"fonts", "weights", "Font weights", "Named font weights.", "font-weight: var(--font-weight-semi-bold).", "fonts.weights", true},
		{"fonts", "sizes", "Font sizes", "A fluid type scale that grows with the viewport.", "font-size: var(--font-size-5). Step 0 is body text; negative steps are captions and fine print.", "fonts.sizes", true},
		{"fonts", "letter-spacing", "Letter spacing", "Tracking values in em.", "letter-spacing: var(--letter-spacing-wide) for small caps and labels.", "fonts.letterSpacing", true},
		{"fonts", "line-heights", "Line heights", "Unitless line-height multipliers.", "line-height: var(--line-height-snug) for headings, normal for body text.", "fonts.lineHeights", true},
	}},
	{"surface", "Surface", "surface", []section{
		{"surface", "radius", "Radius", "Corner radii.", "border-radius: var(--radius-md). --radius-pill and --radius-round cover chips and avatars.", "surface.radius", true},
		{"surface", "borders", "Borders", "Border widths.", "border: var(--border-width-thin) solid var(--neutral-4).", "surface.borders", true},
		{"surface", "shadows", "Shadows", "Layered elevation shadows tinted by one hue.", "box-shadow: var(--shadow-3). Dark mode deepens the tint automatically.", "surface.shadows", true},
	}},
	{"motion", "Motion", "motion", []section{
		{"motion", "durations", "Durations", "Animation and transition durations.", "transition: color var(--anim-duration-base) var(--ease-standard). With reduced-motion overrides on, every duration becomes 0ms for users who ask for less motion.", "motion", true},
		{"motion", "easings", "Easings", "Named timing functions.", "Pair a duration with an easing: var(--ease-out) for entering, var(--ease-in) for leaving.", "motion", true},
	}},
	{"export", "Export", "export", []section{
		{"export", "bundle", "Bundle", "Configure how Stellar packages your generated CSS through esbuild.", "Bundle settings only affect the final CSS bundle. They do not change token generation or previews, so you can tune output format without reauthoring your design system. Use the preview stats to see how output size shifts as you change bundle settings.", "bundle", false},
	}},
}

func findSection(g, s string) (section, bool) {
	for _, gr := range groups {
		if gr.Slug != g {
			continue
		}
		for _, sec := range gr.Sections {
			if sec.Slug == s {
				return sec, true
			}
		}
	}
	return section{}, false
}

// toTree converts a config into the generic JSON tree the editor's signals
// carry. $schema is dropped: it is not editable and "$" keys would be
// awkward in Datastar expressions.
func toTree(c *config.Config) map[string]any {
	var m map[string]any
	if err := json.Unmarshal(c.Marshal(), &m); err != nil {
		panic(err)
	}
	delete(m, "$schema")
	return m
}

func fromTree(m map[string]any) (*config.Config, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return config.Parse(b)
}

// lookup walks a dotted path through maps and arrays.
func lookup(root any, path string) (any, error) {
	cur := root
	for _, k := range strings.Split(path, ".") {
		switch v := cur.(type) {
		case map[string]any:
			n, ok := v[k]
			if !ok {
				return nil, fmt.Errorf("no key %q in %s", k, path)
			}
			cur = n
		case []any:
			i, err := strconv.Atoi(k)
			if err != nil || i < 0 || i >= len(v) {
				return nil, fmt.Errorf("bad index %q in %s", k, path)
			}
			cur = v[i]
		default:
			return nil, fmt.Errorf("%s does not lead to an object", path)
		}
	}
	return cur, nil
}

// assign sets the value at a dotted path whose parent is an object.
func assign(root map[string]any, path string, val any) error {
	parent, key, found := strings.CutLast(path, ".")
	if !found {
		root[path] = val
		return nil
	}
	p, err := lookup(root, parent)
	if err != nil {
		return err
	}
	m, ok := p.(map[string]any)
	if !ok {
		return fmt.Errorf("%s is not an object", parent)
	}
	m[key] = val
	return nil
}
