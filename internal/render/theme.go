package render

import "image/color"

// 取色自 Miuix darkColorScheme 的深色语义 token。
// 画面遵守 Miuix 规则：黑 / 白 / 灰 + 唯一强调色蓝，只有状态（告警/异常）才出现琥珀与红。
var (
	colBg       = color.NRGBA{0x00, 0x00, 0x00, 0xFF} // surface
	colCard     = color.NRGBA{0x24, 0x24, 0x24, 0xFF} // surfaceContainer
	colCardHigh = color.NRGBA{0x2D, 0x2D, 0x2D, 0xFF} // surfaceContainerHighest
	colTrack    = color.NRGBA{0x39, 0x39, 0x39, 0xFF} // dividerLine
	colDivider  = color.NRGBA{0x39, 0x39, 0x39, 0xFF}

	colText  = color.NRGBA{0xF2, 0xF2, 0xF2, 0xFF} // onSurface
	colSub   = color.NRGBA{0xCC, 0xCC, 0xCC, 0xFF} // onSurfaceSecondary
	colMuted = color.NRGBA{0x8A, 0x8A, 0x8A, 0xFF} // 50% 白

	colAccent  = color.NRGBA{0x27, 0x7A, 0xF7, 0xFF} // primary
	colAccent2 = color.NRGBA{0x33, 0x8F, 0xE4, 0xFF} // primaryContainer
	colAmber   = color.NRGBA{0xF5, 0x9E, 0x0B, 0xFF} // 注意
	colRed     = color.NRGBA{0xF1, 0x25, 0x22, 0xFF} // error
)

func alpha(c color.NRGBA, a uint8) color.NRGBA {
	c.A = a
	return c
}

func healthColor(p float64) color.NRGBA {
	switch {
	case p >= 90:
		return colRed
	case p >= 80:
		return colAmber
	default:
		return colAccent
	}
}
