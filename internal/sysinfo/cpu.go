package sysinfo

import (
	"strconv"
	"strings"
)

type cpuTimes struct {
	total uint64
	idle  uint64
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
		var t cpuTimes
		for i, f := range fields[1:] {
			v := parseUint(f)
			if i >= 8 {
				break
			}
			t.total += v
			if i == 3 || i == 4 { // idle + iowait
				t.idle += v
			}
		}
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

func cpuModel() string {
	for _, line := range strings.Split(string(readFile("/proc/cpuinfo")), "\n") {
		key := strings.ToLower(strings.TrimSpace(strings.SplitN(line, ":", 2)[0]))
		val := ""
		if p := strings.SplitN(line, ":", 2); len(p) == 2 {
			val = strings.TrimSpace(p[1])
		}
		switch key {
		case "model name", "hardware", "cpu model", "processor":
			if val != "" && !isNumeric(val) {
				return val
			}
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
	// 回退到 /proc/cpuinfo 的 cpu MHz
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
	// hwmon 兜底
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
