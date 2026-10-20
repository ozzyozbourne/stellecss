// Package config defines stellar.config.json: the single source of truth from
// which every Stellar token is generated.
//
// Struct tags drive both JSON encoding and the generated JSON Schema (see
// [Schema]): `desc` is the schema description, `min`/`max` bound numbers and
// array lengths, and `enum` lists the allowed strings separated by "|".
package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

// SchemaURL is written into new config files so editors can validate them.
const SchemaURL = "./stellar.config.schema.json"

// Version is the config format version; Load rejects any other value.
const Version = 1

type Config struct {
	Schema  string  `json:"$schema,omitempty"`
	Version int     `json:"version" desc:"Config format version." min:"1" max:"1"`
	General General `json:"general"`
	Colors  Colors  `json:"colors"`
	Fonts   Fonts   `json:"fonts"`
	Surface Surface `json:"surface"`
	Motion  Motion  `json:"motion"`
	Bundle  Bundle  `json:"bundle"`
}

type General struct {
	Viewport    Viewport    `json:"viewport"`
	Size        Scale       `json:"size"`
	ZIndexes    ZIndexes    `json:"zindexes"`
	AspectRatio AspectRatio `json:"aspectRatio"`
	Normalize   Normalize   `json:"normalize"`
}

type Viewport struct {
	Disabled     bool    `json:"disabled"`
	Min          float64 `json:"min" desc:"Smallest supported layout width in px." min:"120" max:"4000"`
	Max          float64 `json:"max" desc:"Largest supported layout width in px." min:"120" max:"8000"`
	BaseFontSize float64 `json:"baseFontSize" desc:"Pixels per rem." min:"4" max:"64"`
}

// Scale is a fluid modular scale: step n interpolates between
// BaseMin·ScaleMin^n at Viewport.Min and BaseMax·ScaleMax^n at Viewport.Max.
type Scale struct {
	Disabled      bool        `json:"disabled"`
	BaseMin       float64     `json:"baseMin" desc:"Step 0 at the minimum viewport, in rem." min:"0" max:"100"`
	BaseMax       float64     `json:"baseMax" desc:"Step 0 at the maximum viewport, in rem." min:"0" max:"100"`
	ScaleMin      string      `json:"scaleMin" desc:"Musical ratio applied at the minimum viewport." enum:"Unison|Minor Second|Major Second|Minor Third|Major Third|Perfect Fourth|Augmented Fourth|Perfect Fifth|Golden Ratio|Major Sixth|Minor Seventh|Major Seventh|Octave"`
	ScaleMax      string      `json:"scaleMax" desc:"Musical ratio applied at the maximum viewport." enum:"Unison|Minor Second|Major Second|Minor Third|Major Third|Perfect Fourth|Augmented Fourth|Perfect Fifth|Golden Ratio|Major Sixth|Minor Seventh|Major Seventh|Octave"`
	NegativeSteps int         `json:"negativeSteps" desc:"Steps emitted below 0." min:"0" max:"10"`
	PositiveSteps int         `json:"positiveSteps" desc:"Steps emitted above 0." min:"0" max:"30"`
	Pairs         []Pair      `json:"pairs" desc:"Tokens that span two steps, from one step at the minimum viewport to another at the maximum."`
	Named         []NamedStep `json:"named" desc:"Tokens with explicit rem bounds."`
}

type Pair struct {
	From int `json:"from" min:"-10" max:"30"`
	To   int `json:"to" min:"-10" max:"30"`
}

type NamedStep struct {
	Name string  `json:"name" desc:"Token suffix, lower-case letters, digits and hyphens."`
	Min  float64 `json:"min" desc:"Value at the minimum viewport, in rem." min:"0" max:"100"`
	Max  float64 `json:"max" desc:"Value at the maximum viewport, in rem." min:"0" max:"100"`
}

type ZIndexes struct {
	Disabled bool     `json:"disabled"`
	Levels   []ZLevel `json:"levels"`
}

type ZLevel struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Value       int    `json:"value" min:"-1" max:"2147483647"`
}

type AspectRatio struct {
	Disabled bool         `json:"disabled"`
	Ratios   []NamedRatio `json:"ratios"`
}

type NamedRatio struct {
	Name   string  `json:"name"`
	Width  float64 `json:"width" min:"0.01" max:"1000"`
	Height float64 `json:"height" min:"0.01" max:"1000"`
}

type Normalize struct {
	Disabled bool `json:"disabled"`
	// Each flag adds one group of opinionated base rules.
	BoxSizing    bool `json:"boxSizing" desc:"border-box on every element."`
	ResetMargins bool `json:"resetMargins" desc:"Remove default margins."`
	MediaBlock   bool `json:"mediaBlock" desc:"img, svg, video and canvas render as blocks capped at their container width."`
	InheritFonts bool `json:"inheritFonts" desc:"Form controls inherit font."`
	TextWrap     bool `json:"textWrap" desc:"Balance headings and prettify paragraphs."`
	SmoothScroll bool `json:"smoothScroll" desc:"Smooth scrolling unless reduced motion is requested."`
}

type Colors struct {
	Theme     Theme     `json:"theme"`
	Named     Named     `json:"named"`
	Charts    Charts    `json:"charts"`
	Code      Code      `json:"code"`
	Gradients Gradients `json:"gradients"`
}

// Theme turns seed colours (picked from an image or typed in) into tone ramps.
type Theme struct {
	Disabled bool `json:"disabled"`
	// Image is a path, relative to the config file, from which Seeds are extracted.
	Image       string   `json:"image" desc:"Image, relative to the config file, that seeds are extracted from. Empty keeps Seeds."`
	Seeds       []string `json:"seeds" desc:"Extracted or hand-picked chips as #rrggbb." max:"48"`
	HarmonyMode string   `json:"harmonyMode" desc:"How secondary and tertiary hues relate to primary." enum:"best|complement|analogous|triad|split|square|tetradic"`
	// Palette is the index into the ranked mini palettes; 0 is the best.
	Palette     int     `json:"palette" min:"0" max:"40"`
	Steps       int     `json:"steps" desc:"Tones per ramp." min:"3" max:"20"`
	ToneBias    float64 `json:"toneBias" desc:"Percent trimmed from both tone extremes." min:"0" max:"40"`
	Easing      string  `json:"easing" desc:"Tone distribution across the ramp." enum:"linear|in-quad|out-quad|in-out-quad|in-out-cubic|in-out-sine"`
	OnTargetLc  float64 `json:"onTargetLc" desc:"APCA Lc the -on text colour must reach against its tone." min:"0" max:"108"`
	DimTargetLc float64 `json:"dimTargetLc" desc:"APCA Lc the -dim colour aims for against its tone." min:"0" max:"108"`
	// Scheme picks how dark mode is emitted.
	Scheme string `json:"scheme" desc:"media: a prefers-color-scheme block plus [data-theme] overrides. light-dark: one light-dark() pair per token, switched by color-scheme." enum:"media|light-dark"`
	// Invert swaps which ramp end is light in dark mode; when false dark mode reuses light tones.
	Invert bool `json:"invert" desc:"Dark mode mirrors each ramp so step 1 stays the background end."`
}

type Named struct {
	Disabled      bool         `json:"disabled"`
	OnTargetLc    float64      `json:"onTargetLc" min:"0" max:"108"`
	DimTargetLc   float64      `json:"dimTargetLc" min:"0" max:"108"`
	NegativeSteps int          `json:"negativeSteps" min:"0" max:"6"`
	PositiveSteps int          `json:"positiveSteps" min:"0" max:"6"`
	HuePerStep    float64      `json:"huePerStep" desc:"Degrees of hue rotation per step." min:"-60" max:"60"`
	ChromaPerStep float64      `json:"chromaPerStep" desc:"Percent chroma change per step." min:"-50" max:"50"`
	TonePerStep   float64      `json:"tonePerStep" desc:"OKLCH lightness points per step." min:"-30" max:"30"`
	Tokens        []NamedColor `json:"tokens"`
}

type NamedColor struct {
	Name  string `json:"name"`
	Color string `json:"color" desc:"#rrggbb"`
}

type Charts struct {
	Disabled         bool    `json:"disabled"`
	OnTargetLc       float64 `json:"onTargetLc" min:"0" max:"108"`
	DimTargetLc      float64 `json:"dimTargetLc" min:"0" max:"108"`
	QualitativeCount int     `json:"qualitativeCount" min:"1" max:"25"`
	Qualitative      bool    `json:"qualitative" desc:"Export the qualitative palette."`
	DivergingCount   int     `json:"divergingCount" min:"0" max:"12"`
	ToneSteps        int     `json:"toneSteps" min:"3" max:"20"`
	Diverging        []bool  `json:"diverging" desc:"Export flag per diverging palette." max:"12"`
}

type Code struct {
	Disabled        bool    `json:"disabled"`
	OnTargetLc      float64 `json:"onTargetLc" min:"0" max:"108"`
	DimTargetLc     float64 `json:"dimTargetLc" min:"0" max:"108"`
	SurfaceTargetLc float64 `json:"surfaceTargetLc" min:"0" max:"108"`
	InvertLightDark bool    `json:"invertLightDark" desc:"Swap which palette is used in light and dark mode."`
	Variant         int     `json:"variant" desc:"Index of the chosen syntax palette mix." min:"0" max:"40"`
}

type Gradients struct {
	Disabled bool       `json:"disabled"`
	Stops    int        `json:"stops" desc:"color-mix stops per gradient." min:"2" max:"16"`
	Items    []Gradient `json:"items"`
}

type Gradient struct {
	Name  string `json:"name"`
	Angle int    `json:"angle" min:"0" max:"360"`
	From  string `json:"from" desc:"Token name without --, e.g. primary-6."`
	To    string `json:"to"`
}

type Fonts struct {
	Families      FontFamilies  `json:"families"`
	Import        FontImport    `json:"import"`
	Weights       FontWeights   `json:"weights"`
	Sizes         Scale         `json:"sizes"`
	LetterSpacing LetterSpacing `json:"letterSpacing"`
	LineHeights   LineHeights   `json:"lineHeights"`
}

type FontFamilies struct {
	Disabled bool         `json:"disabled"`
	Preset   string       `json:"preset" desc:"Last applied UI-library preset, informational."`
	Families []FontFamily `json:"families"`
}

type FontFamily struct {
	Name  string   `json:"name"`
	Stack []string `json:"stack" desc:"Font faces from highest priority to the generic fallback."`
}

type FontImport struct {
	Disabled bool     `json:"disabled"`
	URLs     []string `json:"urls" desc:"Stylesheet URLs emitted as @import at the top of the bundle."`
}

type FontWeights struct {
	Disabled bool          `json:"disabled"`
	Weights  []NamedNumber `json:"weights"`
}

type LetterSpacing struct {
	Disabled bool          `json:"disabled"`
	Values   []NamedNumber `json:"values" desc:"Values in em."`
}

type LineHeights struct {
	Disabled bool          `json:"disabled"`
	Values   []NamedNumber `json:"values" desc:"Unitless line-height multipliers."`
}

type NamedNumber struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
}

type Surface struct {
	Radius  Radius  `json:"radius"`
	Borders Borders `json:"borders"`
	Shadows Shadows `json:"shadows"`
}

type Radius struct {
	Disabled bool          `json:"disabled"`
	Values   []NamedNumber `json:"values" desc:"Radii in rem; 9999 emits a pill radius."`
}

type Borders struct {
	Disabled bool          `json:"disabled"`
	Values   []NamedNumber `json:"values" desc:"Border widths in px."`
}

type Shadows struct {
	Disabled bool    `json:"disabled"`
	Steps    int     `json:"steps" min:"1" max:"10"`
	Hue      float64 `json:"hue" desc:"Shadow tint hue in degrees." min:"0" max:"360"`
	Strength float64 `json:"strength" desc:"Opacity of the darkest layer in light mode, percent." min:"0" max:"100"`
}

type Motion struct {
	Disabled  bool          `json:"disabled"`
	Durations []NamedNumber `json:"durations" desc:"Durations in ms."`
	Easings   []NamedEasing `json:"easings"`
}

type NamedEasing struct {
	Name  string `json:"name"`
	Value string `json:"value" desc:"Any CSS <easing-function>."`
}

type Bundle struct {
	Output            string `json:"output" desc:"Bundle path relative to the config file."`
	Namespace         string `json:"namespace" desc:"Selector that receives the custom properties."`
	Overwrite         bool   `json:"overwrite"`
	ColorFallbacks    bool   `json:"colorFallbacks" desc:"Emit a #rrggbb declaration before each oklch() one."`
	FontImports       bool   `json:"fontImports"`
	ReducedMotion     bool   `json:"reducedMotion" desc:"Zero durations under prefers-reduced-motion: reduce."`
	MinifyWhitespace  bool   `json:"minifyWhitespace"`
	MinifySyntax      bool   `json:"minifySyntax"`
	MinifyIdentifiers bool   `json:"minifyIdentifiers"`
	Sourcemap         string `json:"sourcemap" enum:"none|inline|external"`
	SourcesContent    bool   `json:"sourcesContent"`
	SourceRoot        string `json:"sourceRoot"`
	LineLimit         int    `json:"lineLimit" min:"0" max:"100000"`
}

// Ratios maps each musical interval name to its frequency ratio.
var Ratios = map[string]float64{
	"Unison":           1,
	"Minor Second":     1.067,
	"Major Second":     1.125,
	"Minor Third":      1.2,
	"Major Third":      1.25,
	"Perfect Fourth":   1.333,
	"Augmented Fourth": 1.414,
	"Perfect Fifth":    1.5,
	"Golden Ratio":     1.618,
	"Major Sixth":      1.667,
	"Minor Seventh":    1.778,
	"Major Seventh":    1.875,
	"Octave":           2,
}

// RatioNames lists [Ratios] in ascending order for selects.
var RatioNames = []string{"Unison", "Minor Second", "Major Second", "Minor Third", "Major Third", "Perfect Fourth", "Augmented Fourth", "Perfect Fifth", "Golden Ratio", "Major Sixth", "Minor Seventh", "Major Seventh", "Octave"}

// Load reads and validates a config. Fields missing from the file keep their
// [Default] values, so old files gain new sections automatically.
func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(b)
}

// Parse decodes JSON over [Default] and validates the result.
func Parse(b []byte) (*Config, error) {
	c := Default()
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(c); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return c, nil
}

// Marshal renders the config as indented JSON with a trailing newline.
func (c *Config) Marshal() []byte {
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		panic(err) // the type has no unmarshalable fields
	}
	return append(b, '\n')
}

// Save writes the config atomically, so a reader never sees half a file.
func (c *Config) Save(path string) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".stellar-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(c.Marshal()); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// Clone returns a deep copy.
func (c *Config) Clone() *Config {
	n := Default()
	if err := json.Unmarshal(c.Marshal(), n); err != nil {
		panic(err)
	}
	return n
}

// Validate checks the cross-field rules the schema cannot express.
func (c *Config) Validate() error {
	var errs []error
	if c.Version != Version {
		errs = append(errs, fmt.Errorf("version %d is not supported, want %d", c.Version, Version))
	}
	v := c.General.Viewport
	if v.Min >= v.Max {
		errs = append(errs, fmt.Errorf("general.viewport.min (%g) must be below max (%g)", v.Min, v.Max))
	}
	if v.BaseFontSize <= 0 {
		errs = append(errs, errors.New("general.viewport.baseFontSize must be positive"))
	}
	for name, s := range map[string]Scale{"general.size": c.General.Size, "fonts.sizes": c.Fonts.Sizes} {
		if _, ok := Ratios[s.ScaleMin]; !ok {
			errs = append(errs, fmt.Errorf("%s.scaleMin %q is not a known ratio", name, s.ScaleMin))
		}
		if _, ok := Ratios[s.ScaleMax]; !ok {
			errs = append(errs, fmt.Errorf("%s.scaleMax %q is not a known ratio", name, s.ScaleMax))
		}
		if s.NegativeSteps < 0 || s.PositiveSteps < 0 {
			errs = append(errs, fmt.Errorf("%s steps must not be negative", name))
		}
	}
	if !slices.Contains([]string{"media", "light-dark"}, c.Colors.Theme.Scheme) {
		errs = append(errs, fmt.Errorf("colors.theme.scheme %q must be media or light-dark", c.Colors.Theme.Scheme))
	}
	if c.Colors.Theme.Steps < 3 {
		errs = append(errs, errors.New("colors.theme.steps must be at least 3"))
	}
	return errors.Join(errs...)
}
