package sysinfo

import (
	"strconv"
	"strings"
)

func collectMemory() Memory {
	vals := map[string]uint64{}
	for _, line := range strings.Split(string(readFile("/proc/meminfo")), "\n") {
		p := strings.SplitN(line, ":", 2)
		if len(p) != 2 {
			continue
		}
		f := strings.Fields(p[1])
		if len(f) == 0 {
			continue
		}
		v, _ := strconv.ParseUint(f[0], 10, 64)
		// 单位统一成字节（原始为 kB）
		vals[p[0]] = v * 1024
	}
	m := Memory{
		Total:     vals["MemTotal"],
		Available: vals["MemAvailable"],
		Cached:    vals["Buffers"] + vals["Cached"] + vals["SReclaimable"],
		SwapTotal: vals["SwapTotal"],
	}
	if m.Available == 0 {
		m.Available = vals["MemFree"] + vals["Buffers"] + vals["Cached"]
	}
	if m.Available > m.Total {
		m.Available = m.Total
	}
	m.Used = m.Total - m.Available

	if m.SwapTotal > 0 {
		free := vals["SwapFree"]
		if free > m.SwapTotal {
			free = m.SwapTotal
		}
		m.SwapUsed = m.SwapTotal - free
	}
	return m
}
