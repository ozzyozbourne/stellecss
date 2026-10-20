package color

import "math"

// APCA-W3 0.0.98G-4g constants.
const (
	normBG     = 0.56
	normTXT    = 0.57
	revTXT     = 0.62
	revBG      = 0.65
	blkThrs    = 0.022
	blkClmp    = 1.414
	scaleBoW   = 1.14
	scaleWoB   = 1.14
	loBoWoff   = 0.027
	loWoBoff   = 0.027
	deltaYmin  = 0.0005
	loClip     = 0.1
	mainTRC    = 2.4
	sRco, sGco = 0.2126729, 0.7151522
	sBco       = 0.0721750
)

func screenY(c RGB) float64 {
	y := sRco*math.Pow(clamp01(c.R), mainTRC) + sGco*math.Pow(clamp01(c.G), mainTRC) + sBco*math.Pow(clamp01(c.B), mainTRC)
	if y < blkThrs {
		y += math.Pow(blkThrs-y, blkClmp)
	}
	return y
}

// Contrast returns the APCA lightness contrast Lc of text on bg. It is
// positive for dark text on a light background and negative for light text
// on a dark one; black on white is about 106 and white on black about −108.
func Contrast(text, bg RGB) float64 {
	yt, yb := screenY(text), screenY(bg)
	if math.Abs(yb-yt) < deltaYmin {
		return 0
	}
	var out float64
	if yb > yt {
		sapc := (math.Pow(yb, normBG) - math.Pow(yt, normTXT)) * scaleBoW
		if sapc < loClip {
			return 0
		}
		out = sapc - loBoWoff
	} else {
		sapc := (math.Pow(yb, revBG) - math.Pow(yt, revTXT)) * scaleWoB
		if sapc > -loClip {
			return 0
		}
		out = sapc + loWoBoff
	}
	return out * 100
}

// OnColor returns a text colour of bg's hue whose |Lc| against bg reaches
// target with the smallest lightness change. If neither darker nor lighter
// text reaches target, it returns whichever of black or white contrasts more.
func OnColor(bg OKLCH, target float64) OKLCH {
	bgRGB := bg.RGB()
	tint := OKLCH{H: bg.H, C: math.Min(bg.C, 0.06) * 0.55}
	best, bestDelta, found := OKLCH{}, math.Inf(1), false
	for _, dir := range []float64{-1, 1} {
		// Walk away from bg until the contrast reaches the target.
		for i := 1; i <= 200; i++ {
			l := bg.L + dir*float64(i)*0.005
			if l < 0 || l > 1 {
				break
			}
			c := OKLCH{l, tint.C, tint.H}.InGamut()
			if math.Abs(Contrast(c.RGB(), bgRGB)) >= target {
				if d := math.Abs(l - bg.L); d < bestDelta {
					best, bestDelta, found = c, d, true
				}
				break
			}
		}
	}
	if found {
		return best
	}
	black, white := OKLCH{0, 0, bg.H}, OKLCH{1, 0, bg.H}
	if math.Abs(Contrast(black.RGB(), bgRGB)) >= math.Abs(Contrast(white.RGB(), bgRGB)) {
		return black
	}
	return white
}
