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

func (c *cv) coreCols() int { return 2 }

func (c *cv) coreRows() int {
	n := minInt(len(c.snap.CPU.PerCore), 8)
	if n <= 0 {
		return 1
	}
	rows := (n + 1) / 2
	if rows < 1 {
		rows = 1
	}
	return rows
}

func (c *cv) cpuHeight() float64 { return 500 + float64(c.coreRows())*64 }

func (c *cv) sectionCPU(y float64) float64 {
	h := c.cpuHeight()
	usage := c.snap.CPU.Usage
	meta := fmt.Sprintf("%d 核 · %s", c.snap.CPU.Cores, fmtFreq(c.snap.CPU.FreqMHz))
	if cache := fmtCacheKB(c.snap.CPU.CacheKB); cache != "—" {
		meta += " · " + cache
	}
	innerX, contentY, innerW := c.card(y, h, "处理器", meta)

	cx, cy := innerX+80, contentY+84
	col := healthColor(usage)
	c.ring(cx, cy, 66, 14, usage, col)
	c.strMid(fmtPct0(usage), cx, cy-6, 34, true, colStrong, 1)
	c.strMid("占用", cx, cy+30, 18, false, colMuted, 1)

	model := c.fit(c.snap.CPU.Model, innerW-170, 20, true)
	c.strMid(model, innerX+170, contentY+20, 20, true, colSub, 0)

	values := []string{
		fmtTemp(c.snap.CPU.TempC),
		fmtFreq(c.snap.CPU.FreqMHz),
		fmtLoad(c.snap.Host.Load1),
		fmtCount(c.snap.ProcCount),
	}
	labels := []string{"温度", "主频", "平均负载", "进程"}
	colW := (innerW - 170) / 2
	for i := range labels {
		row := i / 2
		colIdx := i % 2
		x := innerX + 170 + float64(colIdx)*colW
		c.strMid(labels[i], x, contentY+62+float64(row)*74, 20, false, colMuted, 0)
		c.strMid(values[i], x, contentY+96+float64(row)*74, 34, true, colStrong, 0)
	}

	c.hairline(innerX, contentY+212, innerW)

	coreLabel := "各核心负载"
	if c.snap.CPU.Cores > 8 {
		coreLabel = fmt.Sprintf("各核心负载 · 显示前 8 / 共 %d 核", c.snap.CPU.Cores)
	}
	c.strMid(coreLabel, innerX, contentY+224, 20, false, colMuted, 0)

	gridTop := contentY + 256
	const gap = 16.0
	cellW := (innerW - gap) / 2
	per := c.snap.CPU.PerCore
	if len(per) > 8 {
		per = per[:8]
	}
	for i, p := range per {
		r := i / 2
		colIdx := i % 2
		x := innerX + float64(colIdx)*(cellW+gap)
		yy := gridTop + float64(r)*64
		c.strMid(fmt.Sprintf("C%d", i), x, yy+14, 20, false, colMuted, 0)
		c.strMid(fmtPct0(p), x+72, yy+14, 20, false, colSub, 0)
		c.bar(x, yy+32, cellW, 12, p, healthColor(p))
	}

	timeTop := gridTop + float64(c.coreRows())*64 + 24
	c.hairline(innerX, timeTop-16, innerW)
	items := []struct {
		label string
		value float64
	}{
		{"用户", c.snap.CPU.UserPct},
		{"系统", c.snap.CPU.SysPct},
		{"IO 等待", c.snap.CPU.IOWaitPct},
		{"空闲", c.snap.CPU.IdlePct},
	}
	halfW := innerW / 2
	for i, item := range items {
		row := i / 2
		colIdx := i % 2
		x := innerX + float64(colIdx)*halfW
		cy := timeTop + 14 + float64(row)*48
		c.strMid(item.label, x, cy, 20, false, colMuted, 0)
		rx := x + halfW - 16
		c.strMid(fmtPct(item.value), rx, cy, 24, true, colStrong, 2)
	}
	return y + h + cardGap
}

// ---------- 内存 ----------

func (c *cv) memHeight() float64 { return 456 }

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
	c.memRow(innerX, contentY+92, innerW, "交换空间", m.SwapUsed, m.SwapTotal, swapPct, colAccent2)

	items := []struct {
		label string
		value string
	}{
		{"可用", fmtBytes(m.Available)},
		{"缓存", fmtBytes(m.Cached)},
		{"Buffers", fmtBytes(m.Buffers)},
		{"Slab", fmtBytes(m.Slab)},
		{"脏页", fmtBytes(m.Dirty)},
		{"页表", fmtBytes(m.PageTables)},
		{"共享", fmtBytes(m.Shmem)},
		{"已提交", fmtBytes(m.Committed)},
	}
	c.hairline(innerX, contentY+180, innerW)
	gridTop := contentY + 192
	const cols = 2
	cellW := innerW / cols
	for i, item := range items {
		row := i / cols
		colIdx := i % cols
		x := innerX + float64(colIdx)*cellW
		cy := gridTop + float64(row)*40 + 20
		c.strMid(item.label, x, cy, 18, false, colMuted, 0)
		c.strMid(item.value, x+cellW-24, cy, 22, false, colSub, 2)
	}
	return y + h + cardGap
}

func (c *cv) memRow(x, y, w float64, label string, used, total uint64, pct float64, fill color.NRGBA) {
	c.strMid(label, x, y+14, 24, true, colText, 0)
	right := "未启用"
	if total > 0 {
		right = fmt.Sprintf("%s / %s · %s", fmtBytes(used), fmtBytes(total), fmtPct(pct))
	}
	c.strMid(right, x+w, y+14, 22, true, colStrong, 2)
	if total > 0 {
		c.bar(x, y+40, w, 14, pct, fill)
	} else {
		c.roundedFill(x, y+40, w, 14, 7, colTrack)
	}
}

// ---------- 显卡 ----------

func (c *cv) gpuHeight() float64 {
	n := minInt(len(c.snap.GPUs), 2)
	if n <= 0 {
		return 0
	}
	return 76 + float64(n)*170 + float64(n-1)*16 + 28
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
		by := contentY + float64(i)*186
		name := c.fit(g.Name, innerW, 24, true)
		c.strMid(name, innerX, by+18, 24, true, colStrong, 0)

		chips := joinNonEmpty(fmtTemp(g.TempC), powerText(g), fanText(g))
		if g.MemUtil > 0 {
			chips = joinNonEmpty(chips, "显存带宽 "+fmtPct0(g.MemUtil))
		}
		chips = c.fit(chips, innerW, 18, false)
		c.strMid(chips, innerX, by+50, 18, false, colSub, 0)

		c.strMid("使用率", innerX, by+92, 20, false, colMuted, 0)
		c.bar(innerX+100, by+84, innerW-100-80, 12, g.Util, colAccent)
		c.strMid(fmtPct0(g.Util), innerX+innerW, by+92, 24, true, colStrong, 2)

		memLabel := "显存"
		if g.MemTotal > 0 {
			memLabel = fmt.Sprintf("显存 %s / %s", fmtBytes(g.MemUsed), fmtBytes(g.MemTotal))
		}
		memLabel = c.fit(memLabel, 220, 20, false)
		c.strMid(memLabel, innerX, by+146, 20, false, colMuted, 0)
		memPct := 0.0
		if g.MemTotal > 0 {
			memPct = float64(g.MemUsed) / float64(g.MemTotal) * 100
		}
		c.bar(innerX+220, by+138, innerW-220-80, 12, memPct, colAccent2)
		c.strMid(fmtPct0(memPct), innerX+innerW, by+146, 24, true, colStrong, 2)
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

func fanText(g sysinfo.GPU) string {
	if g.Fan > 0 {
		return fmt.Sprintf("风扇 %.0f%%", g.Fan)
	}
	if g.FanRPM > 0 {
		return fmt.Sprintf("风扇 %d RPM", g.FanRPM)
	}
	return ""
}

// ---------- 存储 ----------

func (c *cv) diskHeight() float64 {
	return 76 + float64(len(c.snap.Disks))*104 + 28
}

func (c *cv) sectionDisks(y float64) float64 {
	h := c.diskHeight()
	disks := c.snap.Disks
	innerX, contentY, innerW := c.card(y, h, "存储", fmt.Sprintf("%d 个分区", len(disks)))

	for i, d := range disks {
		ry := contentY + float64(i)*110
		pct := 0.0
		if d.Total > 0 {
			pct = float64(d.Used) / float64(d.Total) * 100
		}

		c.strMid(c.fit(d.Mount, innerW-150, 24, true), innerX, ry+16, 24, true, colText, 0)
		c.strMid(fmtPct(pct), innerX+innerW, ry+16, 26, true, colStrong, 2)

		dev := strings.TrimPrefix(d.Device, "/dev/")
		if dev == "" || dev == "overlay" {
			dev = d.FS
		} else {
			dev = dev + " · " + d.FS
		}
		if d.Inodes > 0 {
			inodePct := float64(d.InodesUsed) / float64(d.Inodes) * 100
			dev += " · inode " + fmtPct0(inodePct)
		}
		c.strMid(c.fit(dev, innerW-200, 18, false), innerX, ry+46, 18, false, colMuted, 0)
		c.strMid(fmt.Sprintf("%s / %s", fmtBytes(d.Used), fmtBytes(d.Total)), innerX+innerW, ry+46, 18, false, colMuted, 2)

		c.bar(innerX, ry+72, innerW, 14, pct, healthColor(pct))

		if i < len(disks)-1 {
			c.hairline(innerX, ry+104, innerW)
		}
	}
	return y + h + cardGap
}

// ---------- 网络 ----------

func (c *cv) netHeight() float64 {
	return 76 + float64(len(c.snap.Nets))*100 + 28
}

func (c *cv) sectionNets(y float64) float64 {
	h := c.netHeight()
	nets := c.snap.Nets
	innerX, contentY, innerW := c.card(y, h, "网络", fmt.Sprintf("%d 个接口", len(nets)))

	for i, n := range nets {
		ry := contentY + float64(i)*110

		name := c.fit(n.Name, 240, 26, true)
		c.strMid(name, innerX, ry+16, 26, true, colStrong, 0)
		nameW := c.measure(name, 26, true)
		if n.Wireless {
			c.tag(innerX+nameW+10, ry+16, "WiFi", colSub)
		}

		addr := strings.Join(n.Addrs, "  ")
		if addr == "" {
			addr = "无地址"
		}
		if n.Wireless {
			q := normQuality(n.Quality, n.Signal)
			addr = joinNonEmpty(addr, n.SSID, "信号 "+fmtPct0(float64(q)))
		}
		addr = c.fit(addr, innerW-280, 20, false)
		c.strMid(addr, innerX, ry+50, 20, false, colMuted, 0)

		c.strMid(speedText(n.RxRate, true), innerX+innerW, ry+16, 24, true, colStrong, 2)
		c.strMid(speedText(n.TxRate, false), innerX+innerW, ry+50, 20, true, colSub, 2)

		if i < len(nets)-1 {
			c.hairline(innerX, ry+100, innerW)
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

// ---------- 网络连接 ----------

func (c *cv) netStatsHeight() float64 { return 386 }

func (c *cv) sectionNetStats(y float64) float64 {
	h := c.netStatsHeight()
	st := c.snap.NetStats
	innerX, contentY, innerW := c.card(y, h, "网络连接", "TCP")

	tiles := []struct {
		label string
		value string
		sub   string
	}{
		{"已建立", fmtCount(st.TCPEstablished), "TCP"},
		{"监听", fmtCount(st.TCPListen), "TCP"},
		{"TIME_WAIT", fmtCount(st.TCPTimeWait), "TCP"},
		{"总连接", fmtCount(st.TCPTotal), "TCP"},
	}
	const gap = 16.0
	tw := (innerW - gap) / 2
	for i, t := range tiles {
		row := i / 2
		colIdx := i % 2
		c.tile(innerX+float64(colIdx)*(tw+gap), contentY+float64(row)*126, tw, 110, t.label, t.value, t.sub, colStrong)
	}

	gateway := st.Gateway
	if gateway == "" {
		gateway = "—"
	}
	dns := "—"
	if len(st.DNS) > 0 {
		dns = strings.Join(st.DNS, ", ")
	}
	line1 := c.fit("默认网关 "+gateway+" · DNS "+dns, innerW, 20, false)
	c.strMid(line1, innerX, contentY+240, 20, false, colSub, 0)

	line2 := fmt.Sprintf("累计流量 ↓ %s · ↑ %s", fmtBytes(st.RxTotal), fmtBytes(st.TxTotal))
	c.strMid(line2, innerX, contentY+272, 20, false, colMuted, 0)
	return y + h + cardGap
}

// ---------- 磁盘 IO ----------

func (c *cv) ioHeight() float64 {
	n := minInt(len(c.snap.IO.Disks), 4)
	if n == 0 {
		return 320
	}
	return 364 + float64(n)*76
}

func (c *cv) sectionIO(y float64) float64 {
	h := c.ioHeight()
	io := c.snap.IO
	innerX, contentY, innerW := c.card(y, h, "磁盘 I/O", fmt.Sprintf("%d 块磁盘", len(io.Disks)))

	const gap = 16.0
	tw := (innerW - gap) / 2
	tiles := []struct {
		label string
		value string
		sub   string
	}{
		{"读取速率", fmtRate(io.ReadBps), fmt.Sprintf("IOPS %.0f", io.ReadIOPS)},
		{"写入速率", fmtRate(io.WriteBps), fmt.Sprintf("IOPS %.0f", io.WriteIOPS)},
		{"磁盘繁忙", fmtPct(io.Util), "繁忙时间占比"},
		{"总 IOPS", fmtCompact(io.ReadIOPS + io.WriteIOPS), "读写合计"},
	}
	for i, t := range tiles {
		row := i / 2
		colIdx := i % 2
		c.tile(innerX+float64(colIdx)*(tw+gap), contentY+float64(row)*116, tw, 100, t.label, t.value, t.sub, colStrong)
	}

	disks := io.Disks
	if len(disks) > 4 {
		disks = disks[:4]
	}
	if len(disks) > 0 {
		c.strMid("设备", innerX, contentY+240, 20, false, colMuted, 0)
		c.strMid("繁忙", innerX+innerW, contentY+240, 20, false, colMuted, 2)

		for i, d := range disks {
			ry := contentY + 260 + float64(i)*76
			c.strMid(c.fit(d.Name, innerW-140, 22, true), innerX, ry+20, 22, true, colText, 0)
			c.strMid(fmtPct0(d.Util), innerX+innerW, ry+20, 20, false, colSub, 2)
			line := fmt.Sprintf("读 %s · 写 %s", fmtRate(d.ReadBps), fmtRate(d.WriteBps))
			c.strMid(line, innerX, ry+52, 20, false, colSub, 0)
		}
	}
	return y + h + cardGap
}

// ---------- 内核 ----------

func (c *cv) kernelHeight() float64 { return 474 }

func (c *cv) sectionKernel(y float64) float64 {
	h := c.kernelHeight()
	k := c.snap.Kernel
	innerX, contentY, innerW := c.card(y, h, "内核活动", "实时")

	const gap = 16.0
	tw := (innerW - gap) / 2
	tiles := []struct {
		label string
		value string
		sub   string
	}{
		{"上下文切换", fmtCompact(k.CTXPerSec) + "/s", "累计 " + fmtCompactU(k.CTXTotal)},
		{"中断", fmtCompact(k.IntrPerSec) + "/s", "累计 " + fmtCompactU(k.IntrTotal)},
		{"新建进程", fmtCompact(k.ForksPerSec) + "/s", "累计 " + fmtCompactU(k.ForksTotal)},
		{"运行中", fmtCount(k.Running), "进程"},
		{"阻塞", fmtCount(k.Blocked), "进程"},
		{"线程", fmtCount(k.Threads), "总计"},
	}
	for i, t := range tiles {
		row := i / 2
		colIdx := i % 2
		c.tile(innerX+float64(colIdx)*(tw+gap), contentY+float64(row)*126, tw, 110, t.label, t.value, t.sub, colStrong)
	}

	fhMax := fmtCompactU(k.FileHandlesMax)
	if k.FileHandlesMax >= 1<<50 {
		fhMax = "不限"
	}
	fh := fmt.Sprintf("文件句柄 %s / %s", fmtCompactU(k.FileHandles), fhMax)
	c.strMid(fh, innerX, contentY+360, 20, false, colMuted, 0)
	return y + h + cardGap
}

// ---------- 传感器 ----------

func (c *cv) sensorsHeight() float64 {
	n := minInt(len(c.snap.Sensors), 8)
	return 76 + float64(n)*48 + 28
}

func (c *cv) sectionSensors(y float64) float64 {
	h := c.sensorsHeight()
	sensors := c.snap.Sensors
	if len(sensors) > 8 {
		sensors = sensors[:8]
	}
	innerX, contentY, innerW := c.card(y, h, "温度与风扇", fmt.Sprintf("%d 项", len(c.snap.Sensors)))

	for i, s := range sensors {
		cy := contentY + float64(i)*48 + 24
		c.strMid(c.fit(s.Name, innerW-120, 20, false), innerX, cy, 20, false, colSub, 0)
		value := fmt.Sprintf("%.0f°C", s.Value)
		if s.Unit == "RPM" {
			value = fmt.Sprintf("%.0f RPM", s.Value)
		}
		c.strMid(value, innerX+innerW, cy, 22, true, colStrong, 2)
	}
	return y + h + cardGap
}

// ---------- 进程 ----------

func (c *cv) procHeight() float64 {
	n := len(c.snap.ProcsCPU)
	if len(c.snap.ProcsMem) > n {
		n = len(c.snap.ProcsMem)
	}
	return 190 + float64(n)*88
}

func (c *cv) sectionProcs(y float64) float64 {
	h := c.procHeight()
	meta := fmt.Sprintf("共 %d 个进程", c.snap.ProcCount)
	if c.snap.Kernel.Threads > 0 {
		meta += fmt.Sprintf(" · %d 线程", c.snap.Kernel.Threads)
	}
	innerX, contentY, innerW := c.card(y, h, "进程", meta)

	c.strMid("CPU 占用 TOP", innerX, contentY+18, 22, true, colSub, 0)
	for i, p := range c.snap.ProcsCPU {
		c.procRow(innerX, contentY+40+float64(i)*44, innerW, p, false)
	}

	n := len(c.snap.ProcsCPU)
	if len(c.snap.ProcsMem) > n {
		n = len(c.snap.ProcsMem)
	}
	header2 := contentY + 75 + float64(n)*44
	c.hairline(innerX, header2-14, innerW)
	c.strMid("内存占用 TOP", innerX, header2, 22, true, colSub, 0)
	for i, p := range c.snap.ProcsMem {
		c.procRow(innerX, header2+11+float64(i)*44, innerW, p, true)
	}
	return y + h + cardGap
}

func (c *cv) procRow(x, y, w float64, p sysinfo.Proc, mem bool) {
	value := fmtProcCPU(p.CPU)
	vcCol := colStrong
	if mem {
		value = fmtBytes(p.RSS)
		vcCol = colSub
	}

	name := c.fit(p.Name, w-190, 24, false)
	c.strMid(name, x, y+12, 24, false, colText, 0)
	c.strMid(fmt.Sprintf("%d", p.PID), x+w-120, y+12, 18, false, colMuted, 2)
	c.strMid(value, x+w, y+12, 24, true, vcCol, 2)
}
