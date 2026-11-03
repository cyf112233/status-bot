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

func (c *cv) cpuHeight() float64 { return 220 + float64(c.coreRows())*30 }

func (c *cv) sectionCPU(y float64) float64 {
	h := c.cpuHeight()
	usage := c.snap.CPU.Usage
	meta := fmt.Sprintf("%d 核 · %s", c.snap.CPU.Cores, fmtFreq(c.snap.CPU.FreqMHz))
	if cache := fmtCacheKB(c.snap.CPU.CacheKB); cache != "—" {
		meta += " · " + cache
	}
	innerX, contentY, innerW := c.card(y, h, "处理器", meta)

	cx, cy := innerX+52, contentY+50
	col := healthColor(usage)
	c.ring(cx, cy, 43, 9, usage, col)
	c.strMid(fmtPct0(usage), cx, cy-5, 18, true, colStrong, 1)
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
		c.strMid(values[i], x, contentY+69, 18, true, colStrong, 0)
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
		c.bar(x, yy+13, cellW, 6, p, healthColor(p))
	}

	timeY := gridTop + float64(c.coreRows())*30 + 14
	c.strMid("CPU 时间", innerX, timeY, 10, false, colMuted, 0)
	breakdown := []struct {
		label string
		value float64
	}{
		{"用户", c.snap.CPU.UserPct},
		{"系统", c.snap.CPU.SysPct},
		{"IO 等待", c.snap.CPU.IOWaitPct},
		{"空闲", c.snap.CPU.IdlePct},
	}
	groupW := (innerW - 86) / 4
	for i, item := range breakdown {
		x := innerX + 86 + float64(i)*groupW
		c.strMid(item.label, x, timeY, 10, false, colMuted, 0)
		lw := c.measure(item.label, 10, false)
		c.strMid(fmtPct(item.value), x+lw+8, timeY, 11.5, true, colStrong, 0)
	}
	return y + h + cardGap
}

// ---------- 内存 ----------

func (c *cv) memHeight() float64 { return 236 }

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
	gridTop := contentY + 118
	const cols = 4
	cellW := innerW / cols
	for i, item := range items {
		row := i / cols
		colIdx := i % cols
		x := innerX + float64(colIdx)*cellW
		cy := gridTop + float64(row)*26 + 8
		c.strMid(item.label, x, cy, 9.5, false, colMuted, 0)
		lw := c.measure(item.label, 9.5, false)
		c.strMid(item.value, x+lw+7, cy, 11, false, colSub, 0)
	}
	return y + h + cardGap
}

func (c *cv) memRow(x, y, w float64, label string, used, total uint64, pct float64, fill color.NRGBA) {
	c.strMid(label, x, y+10, 13.5, true, colText, 0)
	right := "未启用"
	if total > 0 {
		right = fmt.Sprintf("%s / %s · %s", fmtBytes(used), fmtBytes(total), fmtPct(pct))
	}
	c.strMid(right, x+w, y+10, 12, true, colStrong, 2)
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
		name := c.fit(g.Name, innerW-260, 14, true)
		c.strMid(name, innerX, by+10, 14, true, colStrong, 0)

		chips := joinNonEmpty(fmtTemp(g.TempC), powerText(g), fanText(g))
		if g.MemUtil > 0 {
			chips = joinNonEmpty(chips, "显存带宽 "+fmtPct0(g.MemUtil))
		}
		chips = c.fit(chips, 250, 10.5, false)
		c.strMid(chips, innerX+innerW, by+10, 10.5, false, colSub, 2)

		c.strMid("使用率", innerX, by+36, 10.5, false, colMuted, 0)
		c.bar(innerX+62, by+32, innerW-62-58, 8, g.Util, colAccent)
		c.strMid(fmtPct0(g.Util), innerX+innerW, by+36, 11.5, true, colStrong, 2)

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
		c.strMid(fmtPct0(memPct), innerX+innerW, by+64, 11.5, true, colStrong, 2)
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
		if d.Inodes > 0 {
			inodePct := float64(d.InodesUsed) / float64(d.Inodes) * 100
			dev += " · inode " + fmtPct0(inodePct)
		}
		dev = c.fit(dev, 238, 10, false)
		c.strMid(dev, innerX, ry+30, 10, false, colMuted, 0)

		c.bar(innerX+250, ry+17, innerW-250-166, 8, pct, healthColor(pct))

		c.strMid(fmtPct(pct), innerX+innerW, ry+12, 12.5, true, colStrong, 2)
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
		c.strMid(name, innerX, ry+13, 14, true, colStrong, 0)
		nameW := c.measure(name, 14, true)
		if n.Wireless {
			c.tag(innerX+nameW+8, ry+13, "WiFi", colSub)
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

		c.strMid(speedText(n.RxRate, true), innerX+innerW, ry+12, 12.5, true, colStrong, 2)
		c.strMid(speedText(n.TxRate, false), innerX+innerW, ry+32, 11.5, true, colSub, 2)

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

// ---------- 网络连接 ----------

func (c *cv) netStatsHeight() float64 { return 192 }

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
	const gap = 10.0
	tw := (innerW - 3*gap) / 4
	for i, t := range tiles {
		c.tile(innerX+float64(i)*(tw+gap), contentY, tw, 76, t.label, t.value, t.sub, colStrong)
	}

	gateway := st.Gateway
	if gateway == "" {
		gateway = "—"
	}
	dns := "—"
	if len(st.DNS) > 0 {
		dns = strings.Join(st.DNS, ", ")
	}
	line1 := c.fit("默认网关 "+gateway+" · DNS "+dns, innerW, 10.5, false)
	c.strMid(line1, innerX, contentY+98, 10.5, false, colSub, 0)

	line2 := fmt.Sprintf("累计流量 ↓ %s · ↑ %s", fmtBytes(st.RxTotal), fmtBytes(st.TxTotal))
	c.strMid(line2, innerX, contentY+118, 10.5, false, colMuted, 0)
	return y + h + cardGap
}

// ---------- 磁盘 IO ----------

func (c *cv) ioHeight() float64 {
	n := minInt(len(c.snap.IO.Disks), 4)
	if n == 0 {
		return 150
	}
	return 180 + float64(n)*24
}

func (c *cv) sectionIO(y float64) float64 {
	h := c.ioHeight()
	io := c.snap.IO
	innerX, contentY, innerW := c.card(y, h, "磁盘 I/O", "")

	const gap = 12.0
	tw := (innerW - 2*gap) / 3
	c.tile(innerX, contentY, tw, 76, "读取速率", fmtRate(io.ReadBps), fmt.Sprintf("IOPS %.0f", io.ReadIOPS), colStrong)
	c.tile(innerX+tw+gap, contentY, tw, 76, "写入速率", fmtRate(io.WriteBps), fmt.Sprintf("IOPS %.0f", io.WriteIOPS), colStrong)
	c.tile(innerX+2*(tw+gap), contentY, tw, 76, "磁盘繁忙", fmtPct(io.Util), "繁忙时间占比", colStrong)

	disks := io.Disks
	if len(disks) > 4 {
		disks = disks[:4]
	}
	if len(disks) > 0 {
		headerY := contentY + 98
		c.strMid("设备", innerX, headerY, 9.5, false, colMuted, 0)
		c.strMid("读取", innerX+340, headerY, 9.5, false, colMuted, 0)
		c.strMid("写入", innerX+560, headerY, 9.5, false, colMuted, 0)
		c.strMid("繁忙", innerX+innerW, headerY, 9.5, false, colMuted, 2)

		for i, d := range disks {
			ry := contentY + 110 + float64(i)*24 + 12
			c.strMid(c.fit(d.Name, 250, 11, true), innerX, ry, 11, true, colText, 0)
			c.strMid(fmtRate(d.ReadBps), innerX+340, ry, 10.5, false, colSub, 0)
			c.strMid(fmtRate(d.WriteBps), innerX+560, ry, 10.5, false, colSub, 0)
			c.strMid(fmtPct0(d.Util), innerX+innerW, ry, 10.5, false, colSub, 2)
		}
	}
	return y + h + cardGap
}

// ---------- 内核 ----------

func (c *cv) kernelHeight() float64 { return 258 }

func (c *cv) sectionKernel(y float64) float64 {
	h := c.kernelHeight()
	k := c.snap.Kernel
	innerX, contentY, innerW := c.card(y, h, "内核活动", "实时")

	const gap = 12.0
	tw := (innerW - 2*gap) / 3
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
		row := i / 3
		colIdx := i % 3
		x := innerX + float64(colIdx)*(tw+gap)
		yy := contentY + float64(row)*84
		c.tile(x, yy, tw, 76, t.label, t.value, t.sub, colStrong)
	}

	fhMax := fmtCompactU(k.FileHandlesMax)
	if k.FileHandlesMax >= 1<<50 {
		fhMax = "不限"
	}
	fh := fmt.Sprintf("文件句柄 %s / %s", fmtCompactU(k.FileHandles), fhMax)
	c.strMid(fh, innerX, contentY+180, 10.5, false, colMuted, 0)
	return y + h + cardGap
}

// ---------- 传感器 ----------

func (c *cv) sensorsHeight() float64 {
	n := minInt(len(c.snap.Sensors), 8)
	rows := (n + 1) / 2
	return 54 + float64(rows)*22 + 20
}

func (c *cv) sectionSensors(y float64) float64 {
	h := c.sensorsHeight()
	sensors := c.snap.Sensors
	if len(sensors) > 8 {
		sensors = sensors[:8]
	}
	innerX, contentY, innerW := c.card(y, h, "温度与风扇", fmt.Sprintf("%d 项", len(c.snap.Sensors)))

	const gap = 30.0
	colW := (innerW - gap) / 2
	for i, s := range sensors {
		row := i / 2
		colIdx := i % 2
		x := innerX + float64(colIdx)*(colW+gap)
		cy := contentY + float64(row)*22 + 11
		name := c.fit(s.Name, colW-80, 10.5, false)
		c.strMid(name, x, cy, 10.5, false, colSub, 0)
		value := fmt.Sprintf("%.0f°C", s.Value)
		if s.Unit == "RPM" {
			value = fmt.Sprintf("%.0f RPM", s.Value)
		}
		c.strMid(value, x+colW, cy, 11, true, colStrong, 2)
	}
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
	meta := fmt.Sprintf("共 %d 个进程", c.snap.ProcCount)
	if c.snap.Kernel.Threads > 0 {
		meta += fmt.Sprintf(" · %d 线程", c.snap.Kernel.Threads)
	}
	innerX, contentY, innerW := c.card(y, h, "进程", meta)

	gap := 28.0
	colW := (innerW - gap) / 2

	c.strMid("CPU 占用 TOP", innerX, contentY+8, 10.5, true, colSub, 0)
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
	vcCol := colStrong
	if mem {
		value = fmtBytes(p.RSS)
		vcCol = colSub
	}

	name := c.fit(p.Name, w-118, 12.5, false)
	c.strMid(name, x, y+8, 12.5, false, colText, 0)
	c.strMid(fmt.Sprintf("%d", p.PID), x+w-72, y+8, 9.5, false, colMuted, 2)
	c.strMid(value, x+w, y+8, 12.5, true, vcCol, 2)
}
