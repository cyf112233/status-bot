package render

import (
	"image/color"
	"math"
	"strings"

	"github.com/fogleman/gg"
	"golang.org/x/image/font"

	"statusbot/internal/config"
	"statusbot/internal/sysinfo"
)

type cv struct {
	dc    *gg.Context
	snap  *sysinfo.Snapshot
	cfg   config.Render
	fonts *fontSet
}

func (c *cv) face(size float64, bold bool) font.Face {
	return c.fonts.face(size, bold)
}

func (c *cv) measure(s string, size float64, bold bool) float64 {
	return float64(font.MeasureString(c.face(size, bold), s)) / 64
}

// fit 按宽度截断并加省略号，保证任何文本都不会溢出。
func (c *cv) fit(s string, maxW, size float64, bold bool) string {
	if maxW <= 0 {
		return ""
	}
	if c.measure(s, size, bold) <= maxW {
		return s
	}
	const ell = "…"
	ew := c.measure(ell, size, bold)
	runes := []rune(s)
	lo, hi := 0, len(runes)
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if c.measure(string(runes[:mid]), size, bold)+ew <= maxW {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	if lo <= 0 {
		return ell
	}
	return string(runes[:lo]) + ell
}

func (c *cv) baseline(centerY, size float64, bold bool) float64 {
	m := c.face(size, bold).Metrics()
	return centerY + float64(m.Ascent-m.Descent)/2/64
}

// str 以 baseline 定位绘制文本；anchor: 0 左对齐，1 居中，2 右对齐。
func (c *cv) str(s string, x, y, size float64, bold bool, col color.NRGBA, anchor int) {
	if s == "" {
		return
	}
	c.dc.SetFontFace(c.face(size, bold))
	c.dc.SetColor(col)
	if anchor != 0 {
		w := c.measure(s, size, bold)
		if anchor == 1 {
			x -= w / 2
		} else {
			x -= w
		}
	}
	c.dc.DrawString(s, x, y)
}

// strMid 以垂直中心定位。
func (c *cv) strMid(s string, x, centerY, size float64, bold bool, col color.NRGBA, anchor int) {
	c.str(s, x, c.baseline(centerY, size, bold), size, bold, col, anchor)
}

// spaced 绘制带字距的小标题。
func (c *cv) spaced(s string, x, centerY, size, tracking float64, col color.NRGBA) {
	y := c.baseline(centerY, size, true)
	c.dc.SetFontFace(c.face(size, true))
	c.dc.SetColor(col)
	for _, r := range s {
		rs := string(r)
		c.dc.DrawString(rs, x, y)
		x += c.measure(rs, size, true) + tracking
	}
}

func (c *cv) roundedFill(x, y, w, h, r float64, fill color.NRGBA) {
	c.dc.SetColor(fill)
	c.dc.DrawRoundedRectangle(x, y, w, h, r)
	c.dc.Fill()
}

func (c *cv) hairline(x, y, w float64) {
	c.dc.SetColor(colDivider)
	c.dc.SetLineWidth(1)
	c.dc.DrawLine(x, y, x+w, y)
	c.dc.Stroke()
}

// card 绘制卡片背景并返回内容区顶部。
func (c *cv) card(y, h float64, title, meta string) (float64, float64, float64) {
	x := float64(marginX)
	w := float64(contentW)
	radius := 22.0

	c.roundedFill(x, y, w, h, radius, colCard)

	innerX := x + cardPad
	innerW := w - 2*cardPad
	titleY := y + cardPad + 15

	c.strMid(title, innerX, titleY, 18.5, true, colStrong, 0)
	if meta != "" {
		meta = c.fit(meta, innerW-160, 12, false)
		c.strMid(meta, x+w-cardPad, titleY, 12, false, colMuted, 2)
	}
	return innerX, y + cardPad + 30, innerW
}

// bar 绘制 Miuix 风格的纯色圆角进度条。
func (c *cv) bar(x, y, w, h, pct float64, fill color.NRGBA) {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	r := h / 2
	c.roundedFill(x, y, w, h, r, colTrack)
	fw := w * pct / 100
	if fw <= 0 {
		return
	}
	if fw < h {
		fw = h
	}
	if fw > w {
		fw = w
	}
	c.roundedFill(x, y, fw, h, r, fill)
}

func (c *cv) ring(cx, cy, radius, stroke, pct float64, col color.NRGBA) {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	c.dc.SetLineWidth(stroke)
	c.dc.SetColor(colTrack)
	c.dc.DrawCircle(cx, cy, radius)
	c.dc.Stroke()

	if pct <= 0 {
		return
	}
	start := -math.Pi / 2
	end := start + 2*math.Pi*pct/100

	c.dc.SetLineCapRound()
	c.dc.SetLineWidth(stroke)
	c.dc.SetColor(col)
	c.dc.DrawArc(cx, cy, radius, start, end)
	c.dc.Stroke()
	c.dc.SetLineCapButt()
}

// tile 绘制一个指标块。
func (c *cv) tile(x, y, w, h float64, label, value, sub string, col color.NRGBA) {
	c.roundedFill(x, y, w, h, 16, colCardHigh)

	c.strMid(label, x+14, y+16, 11, false, colMuted, 0)
	c.strMid(value, x+14, y+39, 19, true, col, 0)
	c.strMid(sub, x+14, y+58, 10.5, false, colMuted, 0)
}

func (c *cv) tag(x, y float64, text string, col color.NRGBA) float64 {
	size := 9.5
	w := c.measure(text, size, true) + 12
	h := 15.0
	c.roundedFill(x, y-h/2, w, h, 7, alpha(col, 30))
	c.dc.SetColor(alpha(col, 90))
	c.dc.SetLineWidth(0.8)
	c.dc.DrawRoundedRectangle(x+0.4, y-h/2+0.4, w-0.8, h-0.8, 7)
	c.dc.Stroke()
	c.strMid(text, x+6, y+0.5, size, true, col, 0)
	return w
}

func joinNonEmpty(parts ...string) string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, " · ")
}
