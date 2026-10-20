package gen

import (
	"testing"

	"github.com/ozzyozbourne/stellecss/internal/config"
)

var demoViewport = config.Viewport{Min: 320, Max: 1440, BaseFontSize: 16}

// The expected lines are transcribed from the stellar.css Stellar v0.0.2
// produced on screen in the "Stellar CSS and internalising the Tao" demo
// (42:24 and 50:57), so they pin the real tool's clamp arithmetic.
func TestFluidScaleMatchesStellarDemoOutput(t *testing.T) {
	size := config.Scale{
		BaseMin: 0.5, BaseMax: 1.1,
		ScaleMin: "Minor Second", ScaleMax: "Perfect Fifth",
		NegativeSteps: 2, PositiveSteps: 12,
		Named: []config.NamedStep{{Name: "custom-001", Min: 0.5, Max: 1.2}},
	}
	fontSizes := config.Scale{
		BaseMin: 1, BaseMax: 1.1,
		ScaleMin: "Major Second", ScaleMax: "Minor Third",
		NegativeSteps: 2, PositiveSteps: 12,
	}
	// With equal bases, negative steps shrink faster under the larger max
	// ratio, so the value at the large viewport is the smaller bound.
	swapped := config.Scale{
		BaseMin: 1, BaseMax: 1,
		ScaleMin: "Major Second", ScaleMax: "Minor Third",
		NegativeSteps: 2, PositiveSteps: 1,
	}
	for _, tc := range []struct {
		scale  config.Scale
		prefix string
		want   map[string]string
	}{
		{size, "size", map[string]string{
			"size--2":         "clamp(0.439179rem, calc(0.424976rem + 0.071015vw), 0.488889rem)",
			"size--1":         "clamp(0.468604rem, calc(0.392966rem + 0.378185vw), 0.733333rem)",
			"size-0":          "clamp(0.5rem, calc(0.328571rem + 0.857143vw), 1.1rem)",
			"size-1":          "clamp(0.5335rem, calc(0.2145rem + 1.595vw), 1.65rem)",
			"size-3":          "clamp(0.607384rem, calc(-0.279792rem + 4.43588vw), 3.7125rem)",
			"size-8":          "clamp(0.840012rem, calc(-6.974784rem + 39.073979vw), 28.191797rem)",
			"size-12":         "clamp(1.088787rem, calc(-39.377551rem + 202.331692vw), 142.720972rem)",
			"size-custom-001": "clamp(0.5rem, calc(0.3rem + 1vw), 1.2rem)",
		}},
		{fontSizes, "font-size", map[string]string{
			"font-size--2": "clamp(0.763889rem, calc(0.756393rem + 0.037478vw), 0.790123rem)",
			"font-size-0":  "clamp(1rem, calc(0.971429rem + 0.142857vw), 1.1rem)",
			"font-size-5":  "clamp(1.802032rem, calc(1.534855rem + 1.335885vw), 2.737152rem)",
			"font-size-11": "clamp(3.653236rem, calc(2.361849rem + 6.456937vw), 8.173092rem)",
		}},
		{swapped, "font-size", map[string]string{
			"font-size--2": "clamp(0.694444rem, calc(0.667108rem + 0.136684vw), 0.790123rem)",
			"font-size-0":  "1rem",
			"font-size-1":  "clamp(1.125rem, calc(1.103571rem + 0.107143vw), 1.2rem)",
		}},
	} {
		got := map[string]string{}
		for _, s := range Steps(tc.prefix, tc.scale) {
			got[s.Name] = s.Clamp(demoViewport)
		}
		for name, want := range tc.want {
			if got[name] != want {
				t.Errorf("--%s:\n got %s\nwant %s", name, got[name], want)
			}
		}
	}
}

func TestStepAtClampsOutsideViewport(t *testing.T) {
	s := Step{"x", 1, 2}
	for _, tc := range []struct{ width, want float64 }{{100, 1}, {320, 1}, {880, 1.5}, {1440, 2}, {3000, 2}} {
		if got := s.At(demoViewport, tc.width); got != tc.want {
			t.Errorf("At(%v) = %v, want %v", tc.width, got, tc.want)
		}
	}
}
