package render

import (
	"fmt"
	"math"
	"strconv"
	"time"
)

func fmtBytes(v uint64) string {
	f := float64(v)
	units := []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB"}
	i := 0
	for f >= 1024 && i < len(units)-1 {
		f /= 1024
		i++
	}
	if i == 0 {
		return strconv.FormatUint(v, 10) + " B"
	}
	if f < 10 {
		return strconv.FormatFloat(f, 'f', 2, 64) + " " + units[i]
	}
	if f < 100 {
		return strconv.FormatFloat(f, 'f', 1, 64) + " " + units[i]
	}
	return strconv.FormatFloat(f, 'f', 0, 64) + " " + units[i]
}

func fmtRate(v float64) string {
	if v < 0 || math.IsNaN(v) || math.IsInf(v, 0) {
		v = 0
	}
	units := []string{"B/s", "KiB/s", "MiB/s", "GiB/s", "TiB/s"}
	i := 0
	for v >= 1024 && i < len(units)-1 {
		v /= 1024
		i++
	}
	if i == 0 {
		return strconv.FormatFloat(v, 'f', 0, 64) + " " + units[i]
	}
	if v < 10 {
		return strconv.FormatFloat(v, 'f', 2, 64) + " " + units[i]
	}
	if v < 100 {
		return strconv.FormatFloat(v, 'f', 1, 64) + " " + units[i]
	}
	return strconv.FormatFloat(v, 'f', 0, 64) + " " + units[i]
}

func fmtPct(v float64) string {
	if v < 0 {
		v = 0
	}
	if v > 100 {
		v = 100
	}
	if v >= 100 {
		return "100%"
	}
	return strconv.FormatFloat(v, 'f', 1, 64) + "%"
}

func fmtPct0(v float64) string {
	if v < 0 {
		v = 0
	}
	if v > 100 {
		v = 100
	}
	return strconv.FormatFloat(v, 'f', 0, 64) + "%"
}

// fmtProcCPU 允许超过 100%（多线程进程）。
func fmtProcCPU(v float64) string {
	if v < 0 {
		v = 0
	}
	if v >= 100 {
		return strconv.FormatFloat(v, 'f', 0, 64) + "%"
	}
	if v >= 10 {
		return strconv.FormatFloat(v, 'f', 0, 64) + "%"
	}
	return strconv.FormatFloat(v, 'f', 1, 64) + "%"
}

func fmtLoad(v float64) string {
	return strconv.FormatFloat(v, 'f', 2, 64)
}

func fmtFreq(mhz float64) string {
	if mhz <= 0 {
		return "未知"
	}
	return strconv.FormatFloat(mhz/1000, 'f', 2, 64) + " GHz"
}

func fmtTemp(v float64) string {
	if v <= 0 {
		return "—"
	}
	return strconv.FormatFloat(v, 'f', 0, 64) + "°C"
}

func fmtUptime(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60
	if days > 0 {
		return fmt.Sprintf("%d 天 %02d:%02d", days, hours, mins)
	}
	return fmt.Sprintf("%02d:%02d:%02d", hours, mins, int(d.Seconds())%60)
}

func fmtClock(t time.Time) string {
	return t.Local().Format("2006-01-02 15:04:05")
}

func fmtBoot(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return t.Local().Format("01-02 15:04")
}

func fmtCount(v int) string {
	return strconv.Itoa(v)
}
