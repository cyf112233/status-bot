package sysinfo

import (
	"strconv"
	"strings"
)

type cpuTimes struct {
	total  uint64
	idle   uint64
	user   uint64
	sys    uint64
	iowait uint64
	steal  uint64
}

// readCPUTimes 返回 [总, 核心0, 核心1...]。
func readCPUTimes() []cpuTimes {
	var out []cpuTimes
	for _, line := range strings.Split(string(readFile("/proc/stat")), "\n") {
		if !strings.HasPrefix(line, "cpu") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		var vals [8]uint64
		var t cpuTimes
		for i, f := range fields[1:] {
			if i >= 8 {
				break
			}
			vals[i] = parseUint(f)
			t.total += vals[i]
		}
		t.user = vals[0] + vals[1]
		t.sys = vals[2] + vals[5] + vals[6]
		t.idle = vals[3]
		t.iowait = vals[4]
		t.steal = vals[7]
		out = append(out, t)
	}
	return out
}

func cpuUsage(prev, cur cpuTimes) float64 {
	if cur.total <= prev.total {
		return 0
	}
	total := cur.total - prev.total
	idle := uint64(0)
	if cur.idle > prev.idle {
		idle = cur.idle - prev.idle
	}
	if idle > total {
		return 0
	}
	return float64(total-idle) / float64(total) * 100
}

// cpuBreakdown 返回用户 / 系统 / IO 等待 / 抢占 / 空闲占比。
func cpuBreakdown(prev, cur cpuTimes) (user, sys, iowait, steal, idle float64) {
	if cur.total <= prev.total {
		return
	}
	total := float64(cur.total - prev.total)
	pct := func(a, b uint64) float64 {
		if b <= a {
			return 0
		}
		return float64(b-a) / total * 100
	}
	return pct(prev.user, cur.user), pct(prev.sys, cur.sys),
		pct(prev.iowait, cur.iowait), pct(prev.steal, cur.steal), pct(prev.idle, cur.idle)
}

func cpuModel() string {
	for _, line := range strings.Split(string(readFile("/proc/cpuinfo")), "\n") {
		p := strings.SplitN(line, ":", 2)
		if len(p) != 2 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(p[0]))
		val := strings.TrimSpace(p[1])
		if val == "" || isNumeric(val) {
			continue
		}
		switch key {
		case "model name", "hardware", "cpu model", "processor":
			return val
		}
	}
	return "未知处理器"
}

func isNumeric(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}

func cpuFreqMHz() float64 {
	files, _ := filepathGlob("/sys/devices/system/cpu/cpu[0-9]*/cpufreq/scaling_cur_freq")
	if len(files) > 0 {
		var sum float64
		var n int
		for _, f := range files {
			if v := readInt(f); v > 0 {
				sum += float64(v) / 1000
				n++
			}
		}
		if n > 0 {
			return sum / float64(n)
		}
	}
	var sum float64
	var n int
	for _, line := range strings.Split(string(readFile("/proc/cpuinfo")), "\n") {
		if strings.HasPrefix(line, "cpu MHz") {
			if p := strings.SplitN(line, ":", 2); len(p) == 2 {
				sum += parseFloat(p[1])
				n++
			}
		}
	}
	if n > 0 {
		return sum / float64(n)
	}
	return 0
}

func cpuTemp() float64 {
	special := 0.0
	fallback := 0.0
	zones, _ := filepathGlob("/sys/class/thermal/thermal_zone*")
	for _, z := range zones {
		milli := readInt(z + "/temp")
		if milli <= 0 {
			continue
		}
		t := float64(milli) / 1000
		if t < 5 || t > 130 {
			continue
		}
		if t > fallback {
			fallback = t
		}
		typ := strings.ToLower(readTrim(z + "/type"))
		for _, key := range []string{"x86_pkg_temp", "coretemp", "cpu", "soc", "pkg", "k10temp"} {
			if strings.Contains(typ, key) {
				if t > special {
					special = t
				}
				break
			}
		}
	}
	if special > 0 {
		return special
	}
	if fallback > 0 {
		return fallback
	}
	hwmons, _ := filepathGlob("/sys/class/hwmon/hwmon*")
	for _, h := range hwmons {
		name := strings.ToLower(readTrim(h + "/name"))
		if !strings.Contains(name, "coretemp") && !strings.Contains(name, "k10temp") && !strings.Contains(name, "cpu") {
			continue
		}
		temps, _ := filepathGlob(h + "/temp*_input")
		for _, t := range temps {
			v := float64(readInt(t)) / 1000
			if v > 5 && v < 130 && v > special {
				special = v
			}
		}
	}
	return special
}

func cpuFanRPM() int {
	hwmons, _ := filepathGlob("/sys/class/hwmon/hwmon*")
	best := 0
	for _, h := range hwmons {
		fans, _ := filepathGlob(h + "/fan*_input")
		for _, f := range fans {
			if v := int(readInt(f)); v > best && v < 30000 {
				best = v
			}
		}
	}
	return best
}

func cpuGovernor() string {
	if s := readTrim("/sys/devices/system/cpu/cpu0/cpufreq/scaling_governor"); s != "" {
		return s
	}
	return ""
}

func cpuCacheKB() uint64 {
	for _, line := range strings.Split(string(readFile("/proc/cpuinfo")), "\n") {
		if !strings.HasPrefix(strings.ToLower(line), "cache size") {
			continue
		}
		if p := strings.SplitN(line, ":", 2); len(p) == 2 {
			f := strings.Fields(p[1])
			if len(f) > 0 {
				if v, err := strconv.ParseUint(f[0], 10, 64); err == nil {
					return v
				}
			}
		}
	}
	return 0
}
