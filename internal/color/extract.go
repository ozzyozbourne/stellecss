package color

import (
	"cmp"
	"image"
	"math"
	"slices"
)

// Extract clusters an image's pixels in OKLab with k-means and returns k
// chips ordered from most to least common. It samples at most 96×96 pixels
// and seeds deterministically, so the same image always yields the same chips.
func Extract(img image.Image, k int) []RGB {
	b := img.Bounds()
	stepX := max(1, b.Dx()/96)
	stepY := max(1, b.Dy()/96)
	var px []OKLab
	for y := b.Min.Y; y < b.Max.Y; y += stepY {
		for x := b.Min.X; x < b.Max.X; x += stepX {
			r, g, bb, a := img.At(x, y).RGBA()
			if a < 0x8000 {
				continue
			}
			px = append(px, RGB{float64(r) / 0xffff, float64(g) / 0xffff, float64(bb) / 0xffff}.Lab())
		}
	}
	if len(px) == 0 || k <= 0 {
		return nil
	}
	k = min(k, len(px))

	// k-means++ initialisation with a fixed choice rule (farthest point)
	// instead of random sampling, for reproducible chips.
	centers := []OKLab{px[len(px)/2]}
	dist := make([]float64, len(px))
	for i := range dist {
		dist[i] = math.Inf(1)
	}
	for len(centers) < k {
		far, farD := 0, -1.0
		last := centers[len(centers)-1]
		for i, p := range px {
			dist[i] = math.Min(dist[i], Distance(p, last))
			if dist[i] > farD {
				far, farD = i, dist[i]
			}
		}
		centers = append(centers, px[far])
	}

	assign := make([]int, len(px))
	for range 24 {
		sums := make([]OKLab, k)
		counts := make([]int, k)
		for i, p := range px {
			best, bestD := 0, math.Inf(1)
			for j, c := range centers {
				if d := Distance(p, c); d < bestD {
					best, bestD = j, d
				}
			}
			assign[i] = best
			sums[best].L += p.L
			sums[best].A += p.A
			sums[best].B += p.B
			counts[best]++
		}
		for j := range centers {
			if counts[j] > 0 {
				n := float64(counts[j])
				centers[j] = OKLab{sums[j].L / n, sums[j].A / n, sums[j].B / n}
			}
		}
	}
	counts := make([]int, k)
	for _, a := range assign {
		counts[a]++
	}
	idx := make([]int, k)
	for i := range idx {
		idx[i] = i
	}
	slices.SortStableFunc(idx, func(a, b int) int { return cmp.Compare(counts[b], counts[a]) })
	out := make([]RGB, 0, k)
	for _, i := range idx {
		c := centers[i].RGB()
		out = append(out, RGB{clamp01(c.R), clamp01(c.G), clamp01(c.B)})
	}
	return out
}

// Locate finds, for each chip, the sampled pixel closest to it in OKLab and
// returns its position as fractions of the image width and height, for
// placing chip markers over the image.
func Locate(img image.Image, chips []RGB) [][2]float64 {
	b := img.Bounds()
	stepX := max(1, b.Dx()/64)
	stepY := max(1, b.Dy()/64)
	out := make([][2]float64, len(chips))
	best := make([]float64, len(chips))
	labs := make([]OKLab, len(chips))
	for i, c := range chips {
		labs[i] = c.Lab()
		best[i] = math.Inf(1)
	}
	for y := b.Min.Y; y < b.Max.Y; y += stepY {
		for x := b.Min.X; x < b.Max.X; x += stepX {
			r, g, bb, _ := img.At(x, y).RGBA()
			p := RGB{float64(r) / 0xffff, float64(g) / 0xffff, float64(bb) / 0xffff}.Lab()
			for i, l := range labs {
				if d := Distance(p, l); d < best[i] {
					best[i] = d
					out[i] = [2]float64{float64(x-b.Min.X) / float64(b.Dx()), float64(y-b.Min.Y) / float64(b.Dy())}
				}
			}
		}
	}
	return out
}
