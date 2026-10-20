// Package color converts between sRGB and OKLCH, maps colours into the sRGB
// gamut, and measures APCA lightness contrast.
package color

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// RGB is gamma-encoded sRGB with channels in [0, 1].
type RGB struct{ R, G, B float64 }

// OKLCH is perceptual lightness L in [0, 1], chroma C ≥ 0, and hue H in degrees.
type OKLCH struct{ L, C, H float64 }

// OKLab is the Cartesian form of [OKLCH].
type OKLab struct{ L, A, B float64 }

// ParseHex accepts #rgb or #rrggbb.
func ParseHex(s string) (RGB, error) {
	h := strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(h) == 3 {
		h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
	}
	if len(h) != 6 {
		return RGB{}, fmt.Errorf("color %q is not #rgb or #rrggbb", s)
	}
	v, err := strconv.ParseUint(h, 16, 32)
	if err != nil {
		return RGB{}, fmt.Errorf("color %q: %w", s, err)
	}
	return RGB{float64(v>>16&0xff) / 255, float64(v>>8&0xff) / 255, float64(v&0xff) / 255}, nil
}

// MustHex is [ParseHex] for literals known to be valid.
func MustHex(s string) RGB {
	c, err := ParseHex(s)
	if err != nil {
		panic(err)
	}
	return c
}

// Hex formats the colour as #rrggbb after clamping to the gamut.
func (c RGB) Hex() string {
	b := func(v float64) int { return int(math.Round(clamp01(v) * 255)) }
	return fmt.Sprintf("#%02x%02x%02x", b(c.R), b(c.G), b(c.B))
}

func clamp01(v float64) float64 { return math.Min(1, math.Max(0, v)) }

func toLinear(c float64) float64 {
	if c <= 0.04045 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}

func fromLinear(c float64) float64 {
	if c <= 0.0031308 {
		return 12.92 * c
	}
	return 1.055*math.Pow(c, 1/2.4) - 0.055
}

// Lab converts to OKLab.
func (c RGB) Lab() OKLab {
	r, g, b := toLinear(c.R), toLinear(c.G), toLinear(c.B)
	l := math.Cbrt(0.4122214708*r + 0.5363325363*g + 0.0514459929*b)
	m := math.Cbrt(0.2119034982*r + 0.6806995451*g + 0.1073969566*b)
	s := math.Cbrt(0.0883024619*r + 0.2817188376*g + 0.6299787005*b)
	return OKLab{
		0.2104542553*l + 0.7936177850*m - 0.0040720468*s,
		1.9779984951*l - 2.4285922050*m + 0.4505937099*s,
		0.0259040371*l + 0.7827717662*m - 0.8086757660*s,
	}
}

// RGB converts OKLab to sRGB without clamping; out-of-gamut channels fall outside [0, 1].
func (c OKLab) RGB() RGB {
	l := c.L + 0.3963377774*c.A + 0.2158037573*c.B
	m := c.L - 0.1055613458*c.A - 0.0638541728*c.B
	s := c.L - 0.0894841775*c.A - 1.2914855480*c.B
	l, m, s = l*l*l, m*m*m, s*s*s
	return RGB{
		fromLinear(4.0767416621*l - 3.3077115913*m + 0.2309699292*s),
		fromLinear(-1.2684380046*l + 2.6097574011*m - 0.3413193965*s),
		fromLinear(-0.0041960863*l - 0.7034186147*m + 1.7076147010*s),
	}
}

// LCH converts OKLab to polar form.
func (c OKLab) LCH() OKLCH {
	h := math.Atan2(c.B, c.A) * 180 / math.Pi
	if h < 0 {
		h += 360
	}
	return OKLCH{c.L, math.Hypot(c.A, c.B), h}
}

// Lab converts OKLCH to Cartesian form.
func (c OKLCH) Lab() OKLab {
	r := c.H * math.Pi / 180
	return OKLab{c.L, c.C * math.Cos(r), c.C * math.Sin(r)}
}

// LCH converts sRGB to OKLCH.
func (c RGB) LCH() OKLCH { return c.Lab().LCH() }

// RGB converts to sRGB, reducing chroma first if needed so the result is in gamut.
func (c OKLCH) RGB() RGB { return c.InGamut().Lab().RGB() }

func inGamut(c RGB) bool {
	const eps = 1e-6
	return c.R >= -eps && c.R <= 1+eps && c.G >= -eps && c.G <= 1+eps && c.B >= -eps && c.B <= 1+eps
}

// InGamut returns the colour with the highest chroma at the same lightness and
// hue that sRGB can display. Lightness is clamped to [0, 1].
func (c OKLCH) InGamut() OKLCH {
	c.L = clamp01(c.L)
	if c.C < 0 {
		c.C = 0
	}
	if inGamut(c.Lab().RGB()) {
		return c
	}
	lo, hi := 0.0, c.C
	for range 30 {
		mid := (lo + hi) / 2
		if inGamut(OKLCH{c.L, mid, c.H}.Lab().RGB()) {
			lo = mid
		} else {
			hi = mid
		}
	}
	c.C = lo
	return c
}

// CSS formats the colour the way Stellar writes it: oklch(95.642% 0.025 232.965).
func (c OKLCH) CSS() string {
	c = c.InGamut()
	return "oklch(" + trim(c.L*100, 3) + "% " + trim(c.C, 4) + " " + trim(c.H, 3) + ")"
}

// Hex is the sRGB fallback for [OKLCH.CSS].
func (c OKLCH) Hex() string { return c.RGB().Hex() }

func trim(v float64, prec int) string {
	s := strconv.FormatFloat(v, 'f', prec, 64)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	}
	if s == "-0" {
		s = "0"
	}
	return s
}

// Distance is the Euclidean distance in OKLab, a perceptual colour difference.
func Distance(a, b OKLab) float64 {
	return math.Sqrt((a.L-b.L)*(a.L-b.L) + (a.A-b.A)*(a.A-b.A) + (a.B-b.B)*(a.B-b.B))
}

// HueDelta is the unsigned shortest angle between two hues, in [0, 180].
func HueDelta(a, b float64) float64 {
	d := math.Mod(math.Abs(a-b), 360)
	if d > 180 {
		d = 360 - d
	}
	return d
}

// Mix interpolates in OKLab, t=0 giving a and t=1 giving b.
func Mix(a, b OKLab, t float64) OKLab {
	return OKLab{a.L + (b.L-a.L)*t, a.A + (b.A-a.A)*t, a.B + (b.B-a.B)*t}
}
