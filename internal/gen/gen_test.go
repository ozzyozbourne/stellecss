package gen

import (
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/ozzyozbourne/stellecss/internal/color"
	"github.com/ozzyozbourne/stellecss/internal/config"
)

// Stellar's 9-stop gradients put stops at 12% and 38%, so 12.5 and 37.5
// round half to even rather than up.
func TestGradientStopsRoundHalfToEven(t *testing.T) {
	want := []int{0, 12, 25, 38, 50, 62, 75, 88, 100}
	if got := GradientStops(9); !slices.Equal(got, want) {
		t.Errorf("GradientStops(9) = %v, want %v", got, want)
	}
}

func TestAPCAContrastPolarity(t *testing.T) {
	black, white := color.MustHex("#000000"), color.MustHex("#ffffff")
	if lc := color.Contrast(black, white); math.Abs(lc-106.04) > 0.1 {
		t.Errorf("black on white Lc = %.2f, want ≈106.04", lc)
	}
	if lc := color.Contrast(white, black); math.Abs(lc+107.88) > 0.1 {
		t.Errorf("white on black Lc = %.2f, want ≈-107.88", lc)
	}
}

// An -on colour must reach the APCA target when any text colour can; for mid
// tones where nothing reaches it, it must be the best of black and white.
func TestOnColoursMeetTheirAPCATargetWhenReachable(t *testing.T) {
	c := config.Default()
	target := c.Colors.Theme.OnTargetLc
	for _, r := range Theme(c.Colors.Theme).Roles {
		for i, tn := range append(slices.Clone(r.Light), r.Dark...) {
			bg := tn.Bg.RGB()
			best := math.Max(math.Abs(color.Contrast(color.MustHex("#000000"), bg)), math.Abs(color.Contrast(color.MustHex("#ffffff"), bg)))
			lc := math.Abs(color.Contrast(tn.On.RGB(), bg))
			if want := math.Min(target, best); lc < want-0.5 {
				t.Errorf("%s tone %d: |Lc| %.1f, want at least %.1f", r.Name, i, lc, want)
			}
		}
	}
}

func TestOKLCHFormatMatchesStellar(t *testing.T) {
	got := color.OKLCH{L: 0.95642, C: 0.02512, H: 232.9651}.CSS()
	if got != "oklch(95.642% 0.0251 232.965)" && got != "oklch(95.642% 0.025 232.965)" {
		t.Errorf("CSS() = %q", got)
	}
}

func TestDisablingASectionRemovesItsTokens(t *testing.T) {
	full := Build(config.Default())
	c := config.Default()
	c.Colors.Charts.Disabled = true
	c.Colors.Code.Disabled = true
	less := Build(c)
	if strings.Contains(less.CSS, "--chart-") || strings.Contains(less.CSS, "--code-") {
		t.Error("disabled charts/code still emitted")
	}
	if CountProperties(less.CSS) >= CountProperties(full.CSS) {
		t.Error("disabling sections did not reduce the property count")
	}
}

// Without the viewport there is nothing to interpolate across, so the fluid
// scales must disappear instead of emitting clamp() with a zero range.
func TestDisabledViewportDropsFluidScales(t *testing.T) {
	c := config.Default()
	c.General.Viewport.Disabled = true
	css := Build(c).CSS
	if strings.Contains(css, "--size-0:") || strings.Contains(css, "--font-size-0:") {
		t.Error("fluid tokens emitted without a viewport")
	}
}

func TestLightDarkSchemeEmitsOnePairPerToken(t *testing.T) {
	c := config.Default()
	c.Colors.Theme.Scheme = "light-dark"
	css := Build(c).CSS
	if strings.Contains(css, "prefers-color-scheme: dark") {
		t.Error("light-dark scheme still emitted a media block")
	}
	if !strings.Contains(css, "--primary-1: light-dark(oklch(") {
		t.Error("no light-dark() pair for --primary-1")
	}
}
