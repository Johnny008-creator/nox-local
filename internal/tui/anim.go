package tui

import (
	"image/color"
	"math"
	"strings"

	"charm.land/lipgloss/v2"
)

// Animation helpers. Everything is driven by m.frame (one tick ≈ 110 ms), so
// there is no extra timer: when nothing animates, nothing redraws more often.

// lerp mixes a and b; t=0 is a, t=1 is b.
func lerp(a, b color.Color, t float64) color.Color {
	t = math.Max(0, math.Min(1, t))
	ar, ag, ab, _ := a.RGBA()
	br, bg, bb, _ := b.RGBA()
	mix := func(x, y uint32) uint8 { return uint8((float64(x>>8)*(1-t) + float64(y>>8)*t) + 0.5) }
	return color.RGBA{mix(ar, br), mix(ag, bg), mix(ab, bb), 255}
}

// shimmer colours s along stops and sweeps a bright band across it, the way
// light runs along a polished edge. The band travels one rune per frame.
func shimmer(s string, frame int, stops []color.Color) string {
	runes := []rune(s)
	n := len(runes)
	if n == 0 {
		return ""
	}
	pal := lipgloss.Blend1D(max(n, 2), stops...)
	period := n + 10
	center := float64(frame%period) - 5
	var b strings.Builder
	for i, r := range runes {
		if r == ' ' {
			b.WriteRune(r)
			continue
		}
		d := math.Abs(float64(i) - center)
		glow := math.Max(0, 1-d/4) // 4 runes wide on each side
		c := lerp(pal[min(i, len(pal)-1)], cWhite, glow*0.75)
		b.WriteString(lipgloss.NewStyle().Foreground(c).Render(string(r)))
	}
	return b.String()
}

// pulse makes a glyph breathe between the accent and the background tone.
func pulse(glyph string, frame int) string {
	t := (math.Sin(float64(frame)*0.55) + 1) / 2
	return lipgloss.NewStyle().Foreground(lerp(cCard, cLime, 0.35+0.65*t)).Render(glyph)
}

const scrambleRunes = "0123456789abcdefABCDEF#$%&*+=<>?/\\|"

// scramble is the working indicator: a few characters that keep re-rolling
// while an aurora gradient flows through them (inspired by Charm's Crush).
// Each position has its own "birth" offset so the block fades in staggered.
func scramble(width, frame int, stops []color.Color) string {
	loop := append(append([]color.Color{}, stops...), stops[0])
	pal := lipgloss.Blend1D(24, loop...)
	var b strings.Builder
	for i := 0; i < width; i++ {
		birth := (i * 7919) % 5
		if frame < birth {
			b.WriteRune(' ')
			continue
		}
		h := uint32(frame*2654435761) ^ uint32(i*40503)
		h ^= h >> 13
		h *= 0x5bd1e995
		r := scrambleRunes[int(h%uint32(len(scrambleRunes)))]
		c := pal[((i*3-frame)%len(pal)+len(pal))%len(pal)]
		b.WriteString(lipgloss.NewStyle().Foreground(c).Bold(true).Render(string(r)))
	}
	return b.String()
}
