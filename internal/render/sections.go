package render

import (
	"fmt"
	"image/color"
	"strings"

	"statusbot/internal/sysinfo"
)

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ---------- CPU ----------

func (c *cv) coreCols() int {
	n := len(c.snap.CPU.PerCore)
	if n <= 0 {
		return 1
	}
	if n > 8 {
		return 8
	}
	return n
}

func (c *cv) coreRows() int {
	n := minInt(len(c.snap.CPU.PerCore), 16)
	if n <= 0 {
		return 1
	}
	rows := (n + c.coreCols() - 1) / c.coreCols()
	if rows < 1 {
		rows = 1
	}
	return rows
}

func (c *cv) cpuHeight() float64 { return 194 + float64(c.coreRows())*30 }

func (c *cv) sectionCPU(y float64) float64 {
	h := c.cpuHeight()
	usage := c.snap.CPU.Usage
	meta := fmt.Sprintf("%d 核 · %s", c.snap.CPU.Cores, fmtFreq(c.snap.CPU.FreqMHz))
	innerX, contentY, innerW := c.card(y, h, "处理器", meta)

	cx, cy := innerX+52, contentY+50
	col := healthColor(usage)
	c.ring(cx, cy, 43, 9, usage, col)
	c.strMid(fmtPct0(usage), cx, cy-5, 18, true, colText, 1)
	c.strMid("占用", cx, cy+18, 9, false, colMuted, 1)

	model := c.fit(c.snap.CPU.Model, innerW-118, 15, true)
	c.strMid(model, innerX+118, contentY+10, 15, true, colSub, 0)

	values := []string{
		fmtTemp(c.snap.CPU.TempC),
		fmtFreq(c.snap.CPU.FreqMHz),
		fmtLoad(c.snap.Host.Load1),
		fmtCount(c.snap.ProcCount),
	}
	labels := []string{"温度", "主频", "平均负载", "进程"}
	colW := (innerW - 118) / 4
	for i := range labels {
		x := innerX + 118 + float64(i)*colW
		c.strMid(labels[i], x, contentY+42, 10.5, false, colMuted, 0)
		vc := colText
		if i == 0 && c.snap.CPU.TempC >= 80 {
			vc = colAmber
		}
		c.strMid(values[i], x, contentY+69, 18, true, vc, 0)
	}

	coreLabel := "各核心负载"
	if c.snap.CPU.Cores > 16 {
		coreLabel = fmt.Sprintf("各核心负载 · 显示前 16 / 共 %d 核", c.snap.CPU.Cores)
	}
	c.strMid(coreLabel, innerX, contentY+112, 10.5, false, colMuted, 0)

	gridTop := contentY + 126
	cols := c.coreCols()
	const gap = 10.0
	cellW := (innerW - float64(cols-1)*gap) / float64(cols)
	per := c.snap.CPU.PerCore
	if len(per) > 16 {
		per = per[:16]
	}
	for i, p := range per {
		r := i / cols
		colIdx := i % cols
		x := innerX + float64(colIdx)*(cellW+gap)
		yy := gridTop + float64(r)*30
		c.strMid(fmt.Sprintf("C%d", i), x, yy+6, 9.5, false, colMuted, 0)
		c.strMid(fmtPct0(p), x+30, yy+6, 9.5, false, colSub, 0)
		pc := healthColor(p)
		c.bar(x, yy+13, cellW, 6, p, pc)
	}
	return y + h + cardGap
}

// ---------- 内存 ----------

func (c *cv) memHeight() float64 { return 196 }

func (c *cv) sectionMemory(y float64) float64 {
	h := c.memHeight()
	m := c.snap.Memory
	innerX, contentY, innerW := c.card(y, h, "内存", fmt.Sprintf("总计 %s", fmtBytes(m.Total)))

	ramPct := 0.0
	if m.Total > 0 {
		ramPct = float64(m.Used) / float64(m.Total) * 100
	}
	c.memRow(innerX, contentY, innerW, "物理内存", m.Used, m.Total, ramPct, colAccent)

	swapPct := 0.0
	if m.SwapTotal > 0 {
		swapPct = float64(m.SwapUsed) / float64(m.SwapTotal) * 100
	}
	c.memRow(innerX, contentY+58, innerW, "交换空间", m.SwapUsed, m.SwapTotal, swapPct, colAccent2)

	detail := fmt.Sprintf("缓存 %s · 可用 %s", fmtBytes(m.Cached), fmtBytes(m.Available))
	c.strMid(detail, innerX, contentY+58+48+10, 10.5, false, colMuted, 0)
	return y + h + cardGap
}

func (c *cv) memRow(x, y, w float64, label string, used, total uint64, pct float64, fill color.NRGBA) {
	c.strMid(label, x, y+10, 13.5, true, colText, 0)
	right := "未启用"
	if total > 0 {
		right = fmt.Sprintf("%s / %s · %s", fmtBytes(used), fmtBytes(total), fmtPct(pct))
	}
	c.strMid(right, x+w, y+10, 12, false, colSub, 2)
	if total > 0 {
		c.bar(x, y+28, w, 9, pct, fill)
	} else {
		c.roundedFill(x, y+28, w, 9, 4.5, colTrack)
	}
}

// ---------- 显卡 ----------

func (c *cv) gpuHeight() float64 {
	n := minInt(len(c.snap.GPUs), 2)
	if n <= 0 {
		return 0
	}
	return 54 + float64(n)*78 + float64(n-1)*10 + 20
}

func (c *cv) sectionGPU(y float64) float64 {
	h := c.gpuHeight()
	gpus := c.snap.GPUs
	if len(gpus) > 2 {
		gpus = gpus[:2]
	}
	meta := fmt.Sprintf("%d 张显卡", len(c.snap.GPUs))
	innerX, contentY, innerW := c.card(y, h, "显卡", meta)

	for i, g := range gpus {
		by := contentY + float64(i)*88
		name := c.fit(g.Name, innerW-190, 14, true)
		c.strMid(name, innerX, by+10, 14, true, colText, 0)

		chips := joinNonEmpty(fmtTemp(g.TempC), powerText(g))
		c.strMid(chips, innerX+innerW, by+10, 10.5, false, colSub, 2)

		c.strMid("使用率", innerX, by+36, 10.5, false, colMuted, 0)
		c.bar(innerX+62, by+32, innerW-62-58, 8, g.Util, colAccent)
		c.strMid(fmtPct0(g.Util), innerX+innerW, by+36, 11.5, true, colText, 2)

		memLabel := "显存"
		if g.MemTotal > 0 {
			memLabel = fmt.Sprintf("显存 %s / %s", fmtBytes(g.MemUsed), fmtBytes(g.MemTotal))
		}
		memLabel = c.fit(memLabel, 190, 10.5, false)
		c.strMid(memLabel, innerX, by+64, 10.5, false, colMuted, 0)
		memPct := 0.0
		if g.MemTotal > 0 {
			memPct = float64(g.MemUsed) / float64(g.MemTotal) * 100
		}
		c.bar(innerX+190, by+60, innerW-190-58, 8, memPct, colAccent2)
		c.strMid(fmtPct0(memPct), innerX+innerW, by+64, 11.5, true, colText, 2)
	}
	return y + h + cardGap
}

func powerText(g sysinfo.GPU) string {
	if g.PowerW <= 0 {
		return ""
	}
	if g.PowerLimit > 0 {
		return fmt.Sprintf("%.0f / %.0f W", g.PowerW, g.PowerLimit)
	}
	return fmt.Sprintf("%.0f W", g.PowerW)
}

// ---------- 存储 ----------

func (c *cv) diskHeight() float64 {
	return 74 + float64(len(c.snap.Disks))*42
}

func (c *cv) sectionDisks(y float64) float64 {
	h := c.diskHeight()
	disks := c.snap.Disks
	innerX, contentY, innerW := c.card(y, h, "存储", fmt.Sprintf("%d 个分区", len(disks)))

	for i, d := range disks {
		ry := contentY + float64(i)*42
		pct := 0.0
		if d.Total > 0 {
			pct = float64(d.Used) / float64(d.Total) * 100
		}

		mount := c.fit(d.Mount, 238, 13.5, true)
		c.strMid(mount, innerX, ry+13, 13.5, true, colText, 0)

		dev := strings.TrimPrefix(d.Device, "/dev/")
		if dev == "" || dev == "overlay" {
			dev = d.FS
		} else {
			dev = dev + " · " + d.FS
		}
		dev = c.fit(dev, 238, 10, false)
		c.strMid(dev, innerX, ry+30, 10, false, colMuted, 0)

		dcColor := healthColor(pct)
		c.bar(innerX+250, ry+17, innerW-250-166, 8, pct, dcColor)

		vc := colText
		if pct >= 90 {
			vc = colRed
		} else if pct >= 80 {
			vc = colAmber
		}
		c.strMid(fmtPct(pct), innerX+innerW, ry+12, 12.5, true, vc, 2)
		c.strMid(fmt.Sprintf("%s / %s", fmtBytes(d.Used), fmtBytes(d.Total)), innerX+innerW, ry+30, 10, false, colMuted, 2)

		if i < len(disks)-1 {
			c.hairline(innerX, ry+42, innerW)
		}
	}
	return y + h + cardGap
}

// ---------- 网络 ----------

func (c *cv) netHeight() float64 {
	return 74 + float64(len(c.snap.Nets))*50
}

func (c *cv) sectionNets(y float64) float64 {
	h := c.netHeight()
	nets := c.snap.Nets
	innerX, contentY, innerW := c.card(y, h, "网络", fmt.Sprintf("%d 个接口", len(nets)))

	for i, n := range nets {
		ry := contentY + float64(i)*50

		name := c.fit(n.Name, 200, 14, true)
		c.strMid(name, innerX, ry+13, 14, true, colText, 0)
		nameW := c.measure(name, 14, true)
		if n.Wireless {
			c.tag(innerX+nameW+8, ry+13, "WiFi", colAccent)
		}

		addr := strings.Join(n.Addrs, "  ")
		if addr == "" {
			addr = "无地址"
		}
		if n.Wireless {
			q := normQuality(n.Quality, n.Signal)
			addr = joinNonEmpty(addr, n.SSID, "信号 "+fmtPct0(float64(q)))
		}
		addr = c.fit(addr, innerW-240, 10.5, false)
		c.strMid(addr, innerX, ry+32, 10.5, false, colMuted, 0)

		c.strMid(speedText(n.RxRate, true), innerX+innerW, ry+12, 12.5, true, colAccent, 2)
		c.strMid(speedText(n.TxRate, false), innerX+innerW, ry+32, 11.5, false, colSub, 2)

		if i < len(nets)-1 {
			c.hairline(innerX, ry+50, innerW)
		}
	}
	return y + h + cardGap
}

func normQuality(q, signal int) int {
	if q <= 0 && signal < 0 {
		q = (signal + 100) * 2
	}
	if q <= 0 {
		return 0
	}
	if q <= 70 {
		return q * 100 / 70
	}
	if q > 100 {
		return 100
	}
	return q
}

func speedText(v float64, down bool) string {
	if down {
		return "↓ " + fmtRate(v)
	}
	return "↑ " + fmtRate(v)
}

// ---------- 磁盘 IO ----------

func (c *cv) ioHeight() float64 { return 150 }

func (c *cv) sectionIO(y float64) float64 {
	h := c.ioHeight()
	io := c.snap.IO
	innerX, contentY, innerW := c.card(y, h, "磁盘 I/O", "")

	const gap = 12.0
	tw := (innerW - 2*gap) / 3
	c.tile(innerX, contentY, tw, 76, "读取速率", fmtRate(io.ReadBps), fmt.Sprintf("IOPS %.0f", io.ReadIOPS), colAccent)
	c.tile(innerX+tw+gap, contentY, tw, 76, "写入速率", fmtRate(io.WriteBps), fmt.Sprintf("IOPS %.0f", io.WriteIOPS), colSub)
	c.tile(innerX+2*(tw+gap), contentY, tw, 76, "磁盘繁忙", fmtPct(io.Util), "繁忙时间占比", healthColor(io.Util))
	return y + h + cardGap
}

// ---------- 进程 ----------

func (c *cv) procHeight() float64 {
	n := len(c.snap.ProcsCPU)
	if len(c.snap.ProcsMem) > n {
		n = len(c.snap.ProcsMem)
	}
	return 96 + float64(n)*30
}

func (c *cv) sectionProcs(y float64) float64 {
	h := c.procHeight()
	innerX, contentY, innerW := c.card(y, h, "进程", fmt.Sprintf("共 %d 个进程", c.snap.ProcCount))

	gap := 28.0
	colW := (innerW - gap) / 2

	c.strMid("CPU 占用 TOP", innerX, contentY+8, 10.5, true, colAccent, 0)
	c.strMid("内存占用 TOP", innerX+colW+gap, contentY+8, 10.5, true, colSub, 0)

	rows := len(c.snap.ProcsCPU)
	if len(c.snap.ProcsMem) > rows {
		rows = len(c.snap.ProcsMem)
	}
	for i := 0; i < rows; i++ {
		ry := contentY + 24 + float64(i)*30
		if i < len(c.snap.ProcsCPU) {
			c.procRow(innerX, ry, colW, c.snap.ProcsCPU[i], false)
		}
		if i < len(c.snap.ProcsMem) {
			c.procRow(innerX+colW+gap, ry, colW, c.snap.ProcsMem[i], true)
		}
	}
	return y + h + cardGap
}

func (c *cv) procRow(x, y, w float64, p sysinfo.Proc, mem bool) {
	value := fmtProcCPU(p.CPU)
	vcCol := colText
	if p.CPU >= 200 {
		vcCol = colRed
	} else if p.CPU >= 100 {
		vcCol = colAmber
	}
	if mem {
		value = fmtBytes(p.RSS)
		vcCol = colSub
	}

	name := c.fit(p.Name, w-118, 12.5, false)
	c.strMid(name, x, y+8, 12.5, false, colText, 0)
	c.strMid(fmt.Sprintf("%d", p.PID), x+w-72, y+8, 9.5, false, colMuted, 2)
	c.strMid(value, x+w, y+8, 12.5, true, vcCol, 2)
}
