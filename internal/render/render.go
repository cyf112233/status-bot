package render

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/png"
	"math"

	"github.com/fogleman/gg"
	xdraw "golang.org/x/image/draw"

	"statusbot/internal/config"
	"statusbot/internal/sysinfo"
)

const (
	designW   = 900
	marginX   = 30
	contentW  = designW - 2*marginX
	cardPad   = 22
	cardGap   = 16
	headerH   = 160
	footerH   = 50
	bottomPad = 26
)

// Preload 预加载随程序打包的字体，避免首个请求承担解析开销。
func Preload() error { return loadFonts() }

// Render 绘制状态图。以手机竖屏为优先，按 900 宽排版后按需缩放。
func Render(snap *sysinfo.Snapshot, r config.Render) ([]byte, error) {
	if err := loadFonts(); err != nil {
		return nil, err
	}
	if snap == nil {
		return nil, errors.New("暂无采样数据")
	}
	c := &cv{snap: snap, cfg: r, fonts: newFontSet()}
	h := int(math.Ceil(c.height()))
	c.dc = gg.NewContext(designW, h)
	c.background(h)
	c.drawHeader()

	y := float64(headerH)
	y = c.sectionCPU(y)
	y = c.sectionMemory(y)
	if len(snap.GPUs) > 0 {
		y = c.sectionGPU(y)
	}
	if len(snap.Disks) > 0 {
		y = c.sectionDisks(y)
	}
	if len(snap.Nets) > 0 {
		y = c.sectionNets(y)
	}
	y = c.sectionNetStats(y)
	y = c.sectionIO(y)
	y = c.sectionKernel(y)
	if len(snap.Sensors) > 0 {
		y = c.sectionSensors(y)
	}
	if len(snap.ProcsCPU) > 0 || len(snap.ProcsMem) > 0 {
		y = c.sectionProcs(y)
	}
	c.drawFooter(y)

	img := c.dc.Image()
	if r.Width > 0 && r.Width != designW {
		nw := r.Width
		nh := int(math.Round(float64(h) * float64(nw) / designW))
		dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
		xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, img.Bounds(), xdraw.Over, nil)
		img = dst
	}

	var buf bytes.Buffer
	buf.Grow(384 * 1024)
	enc := png.Encoder{CompressionLevel: png.BestSpeed}
	if err := enc.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (c *cv) height() float64 {
	y := float64(headerH)
	y += c.cpuHeight() + cardGap
	y += c.memHeight() + cardGap
	if len(c.snap.GPUs) > 0 {
		y += c.gpuHeight() + cardGap
	}
	if len(c.snap.Disks) > 0 {
		y += c.diskHeight() + cardGap
	}
	if len(c.snap.Nets) > 0 {
		y += c.netHeight() + cardGap
	}
	y += c.netStatsHeight() + cardGap
	y += c.ioHeight() + cardGap
	y += c.kernelHeight() + cardGap
	if len(c.snap.Sensors) > 0 {
		y += c.sensorsHeight() + cardGap
	}
	if len(c.snap.ProcsCPU) > 0 || len(c.snap.ProcsMem) > 0 {
		y += c.procHeight() + cardGap
	}
	return y + footerH + bottomPad
}

func (c *cv) background(h int) {
	c.dc.SetColor(colBg)
	c.dc.DrawRectangle(0, 0, designW, float64(h))
	c.dc.Fill()
}

func (c *cv) drawHeader() {
	x := float64(marginX)
	right := float64(designW - marginX)

	c.dc.SetColor(colMuted)
	c.dc.DrawCircle(x+3, 42, 3)
	c.dc.Fill()
	c.spaced("SYSTEM STATUS", x+15, 42, 11, 2.4, colMuted)

	hostname := c.fit(c.snap.Host.Hostname, contentW-300, 34, true)
	c.strMid(hostname, x, 82, 34, true, colStrong, 0)

	meta := joinNonEmpty(c.snap.Host.OS, c.snap.Host.Kernel, c.snap.Host.Arch, c.snap.Host.Virt)
	meta = c.fit(meta, contentW-300, 13.5, false)
	c.strMid(meta, x, 112, 13.5, false, colSub, 0)

	c.strMid("运行时长", right, 30, 11, false, colMuted, 2)
	c.strMid(fmtUptime(c.snap.Host.Uptime), right, 62, 24, true, colStrong, 2)
	c.strMid("启动于 "+fmtBoot(c.snap.Host.BootTime), right, 94, 11.5, false, colMuted, 2)

	c.hairline(x, 138, contentW)
}

func (c *cv) drawFooter(y float64) {
	c.hairline(marginX, y+4, contentW)
	left := "statusbot · " + c.snap.Host.Timezone + " · 采样于 " + fmtClock(c.snap.At)
	c.strMid(left, marginX, y+26, 11, false, colMuted, 0)
	right := fmt.Sprintf("%d 核 · %s 内存 · %d 个进程", c.snap.CPU.Cores, fmtBytes(c.snap.Memory.Total), c.snap.ProcCount)
	c.strMid(right, designW-marginX, y+26, 11, false, colMuted, 2)
}
