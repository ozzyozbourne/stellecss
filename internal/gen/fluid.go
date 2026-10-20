package gen

import (
	"math"
	"strconv"

	"github.com/ozzyozbourne/stellecss/internal/config"
)

// Step is one token of a fluid scale. Min and Max are rem values at the
// minimum and maximum viewport; Min ≤ Max always holds, because a clamp()
// whose lower bound exceeds its upper bound would ignore the upper one.
type Step struct {
	Name     string
	Min, Max float64
}

// Clamp renders the step as clamp(min, calc(intercept + slope·vw), max) for
// the given viewport, interpolating linearly from Min at v.Min to Max at
// v.Max. A step whose bounds are equal renders as a plain rem value.
func (s Step) Clamp(v config.Viewport) string {
	lo, hi := s.Min, s.Max
	if num(lo, 6) == num(hi, 6) {
		return num(lo, 6) + "rem"
	}
	minRem := v.Min / v.BaseFontSize
	maxRem := v.Max / v.BaseFontSize
	slope := (hi - lo) / (maxRem - minRem)
	intercept := lo - minRem*slope
	return "clamp(" + num(lo, 6) + "rem, calc(" + num(intercept, 6) + "rem + " + num(slope*100, 6) + "vw), " + num(hi, 6) + "rem)"
}

// At returns the step's rem value at a viewport width in px.
func (s Step) At(v config.Viewport, width float64) float64 {
	t := (width - v.Min) / (v.Max - v.Min)
	return s.Min + (s.Max-s.Min)*math.Min(1, math.Max(0, t))
}

func stepValue(base, ratio float64, n int) float64 {
	return base * math.Pow(ratio, float64(n))
}

func sorted(a, b float64) (float64, float64) {
	if a > b {
		return b, a
	}
	return a, b
}

// Steps expands a scale into named tokens: -NegativeSteps … PositiveSteps,
// then pairs, then named steps. prefix is e.g. "size" or "font-size".
func Steps(prefix string, s config.Scale) []Step {
	rmin, rmax := config.Ratios[s.ScaleMin], config.Ratios[s.ScaleMax]
	var out []Step
	for n := -s.NegativeSteps; n <= s.PositiveSteps; n++ {
		lo, hi := sorted(stepValue(s.BaseMin, rmin, n), stepValue(s.BaseMax, rmax, n))
		out = append(out, Step{prefix + "-" + strconv.Itoa(n), lo, hi})
	}
	for _, p := range s.Pairs {
		lo, hi := sorted(stepValue(s.BaseMin, rmin, p.From), stepValue(s.BaseMax, rmax, p.To))
		out = append(out, Step{prefix + "-" + strconv.Itoa(p.From) + "-" + strconv.Itoa(p.To), lo, hi})
	}
	for _, nm := range s.Named {
		lo, hi := sorted(nm.Min, nm.Max)
		out = append(out, Step{prefix + "-" + nm.Name, lo, hi})
	}
	return out
}

func scaleSection(id, prefix string, s config.Scale, v config.Viewport) Section {
	var decls []Decl
	for _, st := range Steps(prefix, s) {
		decls = append(decls, Decl{Name: "--" + st.Name, Value: st.Clamp(v)})
	}
	return Section{ID: id, Blocks: []Block{{Selector: ":root", Decls: decls}}}
}
