package config

import (
	"slices"
	"strings"
)

// Default returns the kitchen-sink baseline: every section enabled, so a
// project starts with everything and turns off what it does not need.
func Default() *Config {
	return &Config{
		Schema:  SchemaURL,
		Version: Version,
		General: General{
			Viewport: Viewport{Min: 320, Max: 1440, BaseFontSize: 16},
			Size: Scale{
				BaseMin: 0.5, BaseMax: 1.1,
				ScaleMin: "Minor Second", ScaleMax: "Minor Third",
				NegativeSteps: 2, PositiveSteps: 12,
				Pairs: []Pair{},
				Named: []NamedStep{{Name: "custom-001", Min: 0.5, Max: 1.2}},
			},
			ZIndexes: ZIndexes{Levels: []ZLevel{
				{"hide", "Pushed behind the page.", -1},
				{"base", "Default stacking.", 0},
				{"raised", "Cards and controls lifted off their surface.", 1},
				{"dropdown", "Menus anchored to a control.", 1000},
				{"sticky", "Sticky headers and toolbars.", 1100},
				{"drawer", "Navigation drawers and shell surfaces.", 1200},
				{"overlay", "Scrims behind modal content.", 1300},
				{"modal", "Dialogs.", 1400},
				{"popover", "Popovers above dialogs.", 1500},
				{"toast", "Transient notifications.", 1600},
				{"tooltip", "Tooltips, always on top.", 1700},
			}},
			AspectRatio: AspectRatio{Ratios: []NamedRatio{
				{"square", 1, 1},
				{"portrait", 3, 4},
				{"landscape", 4, 3},
				{"photo", 3, 2},
				{"widescreen", 16, 9},
				{"ultrawide", 21, 9},
				{"golden", 1.618, 1},
			}},
			Normalize: Normalize{BoxSizing: true, ResetMargins: true, MediaBlock: true, InheritFonts: true, TextWrap: true, SmoothScroll: true},
		},
		Colors: Colors{
			Theme: Theme{
				Seeds:                defaultSeeds(),
				HarmonyMode:          "best",
				Steps:                12,
				ToneBias:             5,
				Lightest:             98.5,
				Darkest:              13,
				NeutralChroma:        0.012,
				NeutralVariantChroma: 0.03,
				Easing:               "in-out-quad",
				OnTargetLc:           90,
				DimTargetLc:          30,
				Scheme:               "media",
				Invert:               true,
			},
			Named: Named{
				OnTargetLc: 90, DimTargetLc: 30,
				NegativeSteps: 2, PositiveSteps: 2,
				HuePerStep: 6, ChromaPerStep: 1.5, TonePerStep: 4,
				Tokens: defaultNamed(),
			},
			Charts: Charts{
				OnTargetLc: 90, DimTargetLc: 30,
				QualitativeCount: 12, Qualitative: true,
				DivergingCount: 6, ToneSteps: 12,
				Diverging: []bool{true, true, true, true, true, true, true, true, true, true, true, true},
			},
			Code: Code{OnTargetLc: 90, DimTargetLc: 30, SurfaceTargetLc: 30, InvertLightDark: true},
			Gradients: Gradients{Stops: 9, Items: []Gradient{
				{"brand-surface", 135, "primary-6", "secondary-6"},
				{"brand-accent", 90, "secondary-6", "tertiary-6"},
				{"neutral-fade", 180, "neutral-1", "neutral-3"},
				{"diverging", 120, "secondary-6", "error-6"},
			}},
		},
		Fonts: Fonts{
			Families: FontFamilies{Families: defaultFamilies()},
			Import: FontImport{URLs: []string{
				"https://fonts.bunny.net/css?family=inter:300,400,500,600,700,800,900|inconsolata:400,700&display=swap",
			}},
			Weights: FontWeights{Weights: []NamedNumber{
				{"thin", 100}, {"extra-light", 200}, {"light", 300}, {"normal", 400}, {"medium", 500},
				{"semi-bold", 600}, {"bold", 700}, {"extra-bold", 800}, {"black", 900},
			}},
			Sizes: Scale{
				BaseMin: 1, BaseMax: 1.1,
				ScaleMin: "Major Second", ScaleMax: "Minor Third",
				NegativeSteps: 2, PositiveSteps: 12,
				Pairs: []Pair{}, Named: []NamedStep{},
			},
			LetterSpacing: LetterSpacing{Values: []NamedNumber{
				{"tighter", -0.05}, {"tight", -0.025}, {"normal", 0}, {"wide", 0.025}, {"wider", 0.05}, {"widest", 0.1},
			}},
			LineHeights: LineHeights{Values: []NamedNumber{
				{"none", 1}, {"tight", 1.15}, {"snug", 1.3}, {"normal", 1.5}, {"relaxed", 1.65}, {"loose", 2},
			}},
		},
		Surface: Surface{
			Radius: Radius{Values: []NamedNumber{
				{"none", 0}, {"xs", 0.125}, {"sm", 0.25}, {"md", 0.5}, {"lg", 0.75}, {"xl", 1}, {"2xl", 1.5}, {"pill", 9999},
			}},
			Borders: Borders{Values: []NamedNumber{{"hairline", 0.5}, {"thin", 1}, {"medium", 2}, {"thick", 4}}},
			Shadows: Shadows{Steps: 6, Hue: 230, Strength: 14},
		},
		Motion: Motion{
			Durations: []NamedNumber{{"instant", 0}, {"fast", 120}, {"base", 200}, {"slow", 320}, {"slower", 500}, {"slowest", 800}},
			Easings: []NamedEasing{
				{"standard", "cubic-bezier(0.2, 0, 0, 1)"},
				{"in", "cubic-bezier(0.55, 0.055, 0.675, 0.19)"},
				{"out", "cubic-bezier(0.215, 0.61, 0.355, 1)"},
				{"in-out", "cubic-bezier(0.645, 0.045, 0.355, 1)"},
				{"spring", "linear(0, 0.009, 0.035 2.1%, 0.141, 0.281 6.7%, 0.723 12.9%, 0.938 16.7%, 1.017, 1.077, 1.121, 1.149 24.3%, 1.159, 1.163, 1.161, 1.154 29.9%, 1.129 32.8%, 1.051 39.6%, 1.017 43.1%, 0.991, 0.977 51%, 0.974 53.8%, 0.975 57.1%, 0.997 69.8%, 1.003 76.9%, 1)"},
			},
		},
		Bundle: Bundle{
			Output: "css/stellar.css", Namespace: ":root", Overwrite: true, ColorFormat: "oklch",
			FontImports: true, ReducedMotion: true,
			Sourcemap: "none", SourcesContent: true,
		},
	}
}

// defaultSeeds are the 24 chips color.Extract finds in the editor's bundled
// nebula image (internal/web/static/img/nebula.jpg), so a fresh config has a
// theme before any image is uploaded.
func defaultSeeds() []string {
	return []string{
		"#4a331e", "#3c2316", "#12202a", "#203136", "#294249", "#613d21", "#0a101b", "#375253",
		"#2a140e", "#764c28", "#396166", "#8c5f32", "#656244", "#457476", "#767751", "#a27844",
		"#558984", "#b68d4d", "#809473", "#a1ac80", "#d5a456", "#e6cb77", "#f5eab8", "#bac8ac",
	}
}

// BrandColors and LanguageColors are the lists the Named panel's
// "Upsert brands" and "Upsert languages" buttons merge in.
var (
	BrandColors = []NamedColor{
		{"adobe", "#fa0f00"}, {"android", "#3ddc84"}, {"aws", "#ff9900"}, {"bluesky", "#0085ff"},
		{"docker", "#2496ed"}, {"figma", "#f24e1e"}, {"microsoft", "#00a4ef"}, {"safari", "#006cff"},
		{"slack", "#4a154b"}, {"stripe", "#635bff"}, {"ubuntu", "#e95420"},
	}
	LanguageColors = []NamedColor{
		{"c", "#a8b9cc"}, {"clojure", "#63b132"}, {"cplusplus", "#00599c"}, {"csharp", "#512bd4"},
		{"css", "#1572b6"}, {"dart", "#0175c2"}, {"elixir", "#4b275f"}, {"go", "#00add8"},
		{"html", "#e34f26"}, {"java", "#5382a1"}, {"javascript", "#f7df1e"}, {"kotlin", "#7f52ff"},
		{"odin", "#3882d2"}, {"php", "#777bb4"}, {"python", "#3776ab"}, {"ruby", "#cc342d"},
		{"rust", "#dea584"}, {"swift", "#f05138"}, {"typescript", "#3178c6"}, {"zig", "#f7a41d"},
	}
)

func defaultNamed() []NamedColor {
	out := append(slices.Clone(BrandColors), LanguageColors...)
	slices.SortFunc(out, func(a, b NamedColor) int { return strings.Compare(a.Name, b.Name) })
	return out
}

// defaultFamilies follow the Modern Font Stacks classification, plus sans,
// serif and mono aliases that start with the imported web fonts.
func defaultFamilies() []FontFamily {
	mono := []string{`"Dank Mono"`, `"Operator Mono"`, "Inconsolata", `"Fira Mono"`, "ui-monospace", `"SF Mono"`, "Monaco", `"Droid Sans Mono"`, `"Source Code Pro"`, `"Cascadia Code"`, "Menlo", "Consolas", `"DejaVu Sans Mono"`, "monospace"}
	return []FontFamily{
		{"sans", []string{"Inter", "system-ui", "-apple-system", `"Segoe UI"`, "Roboto", `"Helvetica Neue"`, "Arial", "sans-serif"}},
		{"serif", []string{"Charter", `"Bitstream Charter"`, `"Sitka Text"`, "Cambria", "Georgia", "serif"}},
		{"mono", mono},
		{"system-ui", []string{"system-ui", "sans-serif"}},
		{"transitional", []string{"Charter", `"Bitstream Charter"`, `"Sitka Text"`, "Cambria", "serif"}},
		{"old-style", []string{`"Iowan Old Style"`, `"Palatino Linotype"`, `"URW Palladio L"`, "P052", "serif"}},
		{"humanist", []string{"Seravek", `"Gill Sans Nova"`, "Ubuntu", "Calibri", `"DejaVu Sans"`, "source-sans-pro", "sans-serif"}},
		{"geometric-humanist", []string{"Avenir", "Montserrat", "Corbel", `"URW Gothic"`, "source-sans-pro", "sans-serif"}},
		{"classical-humanist", []string{"Optima", "Candara", `"Noto Sans"`, "source-sans-pro", "sans-serif"}},
		{"neo-grotesque", []string{"Inter", "Roboto", `"Helvetica Neue"`, `"Arial Nova"`, `"Nimbus Sans"`, "Arial", "sans-serif"}},
		{"monospace-slab-serif", []string{`"Nimbus Mono PS"`, `"Courier New"`, "monospace"}},
		{"monospace-code", mono},
		{"industrial", []string{"Bahnschrift", `"DIN Alternate"`, `"Franklin Gothic Medium"`, `"Nimbus Sans Narrow"`, "sans-serif-condensed", "sans-serif"}},
		{"rounded-sans", []string{"ui-rounded", `"Hiragino Maru Gothic ProN"`, "Quicksand", "Comfortaa", "Manjari", `"Arial Rounded MT"`, `"Arial Rounded MT Bold"`, "Calibri", "source-sans-pro", "sans-serif"}},
		{"slab-serif", []string{"Rockwell", `"Rockwell Nova"`, `"Roboto Slab"`, `"DejaVu Serif"`, `"Sitka Small"`, "serif"}},
		{"antique", []string{"Superclarendon", `"Bookman Old Style"`, `"URW Bookman"`, `"URW Bookman L"`, `"Georgia Pro"`, "Georgia", "serif"}},
		{"didone", []string{"Didot", `"Bodoni MT"`, `"Noto Serif Display"`, `"URW Palladio L"`, "P052", "Sylfaen", "serif"}},
		{"handwritten", []string{`"Segoe Print"`, `"Bradley Hand"`, "Chilanka", "TSCu_Comic", "casual", "cursive"}},
	}
}

// FontPresets swap the sans, serif and mono stacks to match a UI library.
var FontPresets = map[string]map[string][]string{
	"Ant Design":   {"sans": {"-apple-system", "BlinkMacSystemFont", `"Segoe UI"`, "Roboto", `"Helvetica Neue"`, "Arial", `"Noto Sans"`, "sans-serif"}, "mono": {`"SFMono-Regular"`, "Consolas", `"Liberation Mono"`, "Menlo", "Courier", "monospace"}},
	"Bootstrap":    {"sans": {"system-ui", "-apple-system", `"Segoe UI"`, "Roboto", `"Helvetica Neue"`, `"Noto Sans"`, `"Liberation Sans"`, "Arial", "sans-serif"}, "mono": {"SFMono-Regular", "Menlo", "Monaco", "Consolas", `"Liberation Mono"`, `"Courier New"`, "monospace"}},
	"Bulma":        {"sans": {"BlinkMacSystemFont", "-apple-system", `"Segoe UI"`, "Roboto", "Oxygen", "Ubuntu", "Cantarell", `"Fira Sans"`, `"Droid Sans"`, `"Helvetica Neue"`, "Helvetica", "Arial", "sans-serif"}, "mono": {"monospace"}},
	"Carbon (IBM)": {"sans": {`"IBM Plex Sans"`, `"Helvetica Neue"`, "Arial", "sans-serif"}, "serif": {`"IBM Plex Serif"`, "Georgia", "Times", "serif"}, "mono": {`"IBM Plex Mono"`, `"Menlo"`, `"DejaVu Sans Mono"`, `"Bitstream Vera Sans Mono"`, "Courier", "monospace"}},
	"Chakra UI":    {"sans": {"-apple-system", "BlinkMacSystemFont", `"Segoe UI"`, "Helvetica", "Arial", "sans-serif"}, "mono": {"SFMono-Regular", "Menlo", "Monaco", "Consolas", `"Liberation Mono"`, `"Courier New"`, "monospace"}},
	"Foundation":   {"sans": {`"Helvetica Neue"`, "Helvetica", "Roboto", "Arial", "sans-serif"}, "serif": {"Georgia", "Cambria", `"Times New Roman"`, "Times", "serif"}, "mono": {"Consolas", `"Liberation Mono"`, "Courier", "monospace"}},
	"Material UI":  {"sans": {"Roboto", "Helvetica", "Arial", "sans-serif"}, "mono": {`"Roboto Mono"`, "monospace"}},
	"Open Props":   {"sans": {"system-ui", "sans-serif"}, "serif": {"ui-serif", "serif"}, "mono": {`"Dank Mono"`, `"Operator Mono"`, "Inconsolata", `"Fira Mono"`, "ui-monospace", `"SF Mono"`, "Monaco", `"Droid Sans Mono"`, `"Source Code Pro"`, `"Cascadia Code"`, "Menlo", "Consolas", `"DejaVu Sans Mono"`, "monospace"}},
	"Primer":       {"sans": {"-apple-system", "BlinkMacSystemFont", `"Segoe UI"`, `"Noto Sans"`, "Helvetica", "Arial", "sans-serif"}, "mono": {"ui-monospace", "SFMono-Regular", `"SF Mono"`, "Menlo", "Consolas", `"Liberation Mono"`, "monospace"}},
	"Semantic UI":  {"sans": {"Lato", `"Helvetica Neue"`, "Arial", "Helvetica", "sans-serif"}, "mono": {"Monaco", "Menlo", "Consolas", "monospace"}},
	"System":       {"sans": {"system-ui", "sans-serif"}, "serif": {"ui-serif", "serif"}, "mono": {"ui-monospace", "monospace"}},
	"Tailwind":     {"sans": {"ui-sans-serif", "system-ui", "sans-serif", `"Apple Color Emoji"`, `"Segoe UI Emoji"`, `"Segoe UI Symbol"`, `"Noto Color Emoji"`}, "serif": {"ui-serif", "Georgia", "Cambria", `"Times New Roman"`, "Times", "serif"}, "mono": {"ui-monospace", "SFMono-Regular", "Menlo", "Monaco", "Consolas", `"Liberation Mono"`, `"Courier New"`, "monospace"}},
}

// FontPresetNames orders [FontPresets] for display.
var FontPresetNames = []string{"Ant Design", "Bootstrap", "Bulma", "Carbon (IBM)", "Chakra UI", "Foundation", "Material UI", "Open Props", "Primer", "Semantic UI", "System", "Tailwind"}
