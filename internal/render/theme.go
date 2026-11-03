package render

import (
	"image/color"

	"github.com/fogleman/gg"
)

var (
	colBgTop    = color.NRGBA{0x08, 0x0D, 0x14, 0xFF}
	colBgMid    = color.NRGBA{0x0B, 0x13, 0x1D, 0xFF}
	colBgBottom = color.NRGBA{0x0A, 0x10, 0x18, 0xFF}

	colCard    = color.NRGBA{0x13, 0x1C, 0x28, 0xFF}
	colBorder  = color.NRGBA{0x23, 0x32, 0x46, 0xB0}
	colTrack   = color.NRGBA{0x1D, 0x29, 0x38, 0xFF}
	colDivider = color.NRGBA{0xFF, 0xFF, 0xFF, 0x0C}

	colText    = color.NRGBA{0xE9, 0xEF, 0xF7, 0xFF}
	colSub     = color.NRGBA{0x93, 0xA3, 0xB8, 0xFF}
	colMuted   = color.NRGBA{0x64, 0x76, 0x8C, 0xFF}
	colAccent  = color.NRGBA{0x2F, 0xD8, 0xC4, 0xFF}
	colAccent2 = color.NRGBA{0x4C, 0x9B, 0xFF, 0xFF}
	colViolet  = color.NRGBA{0xA7, 0x8B, 0xFA, 0xFF}
	colAmber   = color.NRGBA{0xFF, 0xBC, 0x5C, 0xFF}
	colRed     = color.NRGBA{0xFF, 0x6B, 0x7A, 0xFF}
)

func alpha(c color.NRGBA, a uint8) color.NRGBA {
	c.A = a
	return c
}

func lighten(c color.NRGBA, amount int) color.NRGBA {
	clamp := func(v int) uint8 {
		if v > 255 {
			return 255
		}
		if v < 0 {
			return 0
		}
		return uint8(v)
	}
	return color.NRGBA{clamp(int(c.R) + amount), clamp(int(c.G) + amount), clamp(int(c.B) + amount), c.A}
}

func healthColor(p float64) color.NRGBA {
	switch {
	case p >= 85:
		return colRed
	case p >= 65:
		return colAmber
	default:
		return colAccent
	}
}

func gradient(x0, y0, x1, y1 float64, from, to color.NRGBA) gg.Gradient {
	g := gg.NewLinearGradient(x0, y0, x1, y1)
	g.AddColorStop(0, from)
	g.AddColorStop(1, to)
	return g
}
