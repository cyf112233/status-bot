package sysinfo

import (
	"strconv"
	"strings"
)

type kernelRaw struct {
	ctxt  uint64
	intr  uint64
	forks uint64
}

func collectKernel(prev kernelRaw, has bool, elapsed float64) (Kernel, kernelRaw) {
	var raw kernelRaw
	running, blocked, threads := 0, 0, 0

	for _, line := range strings.Split(string(readFile("/proc/stat")), "\n") {
		f := strings.Fields(line)
		if len(f) < 2 {
			continue
		}
		switch f[0] {
		case "ctxt":
			raw.ctxt = parseUint(f[1])
		case "processes":
			raw.forks = parseUint(f[1])
		case "procs_running":
			running = int(parseUint(f[1]))
		case "procs_blocked":
			blocked = int(parseUint(f[1]))
		case "intr":
			raw.intr = parseUint(f[1])
		}
	}
	if f := strings.Fields(readTrim("/proc/loadavg")); len(f) >= 4 {
		if p := strings.SplitN(f[3], "/", 2); len(p) == 2 {
			threads, _ = strconv.Atoi(p[1])
		}
	}

	k := Kernel{
		CTXTotal:   raw.ctxt,
		IntrTotal:  raw.intr,
		ForksTotal: raw.forks,
		Running:    running,
		Blocked:    blocked,
		Threads:    threads,
	}
	if has && elapsed > 0 {
		rate := func(a, b uint64) float64 {
			if b < a {
				return 0
			}
			return float64(b-a) / elapsed
		}
		k.CTXPerSec = rate(prev.ctxt, raw.ctxt)
		k.IntrPerSec = rate(prev.intr, raw.intr)
		k.ForksPerSec = rate(prev.forks, raw.forks)
	}
	if f := strings.Fields(readTrim("/proc/sys/fs/file-nr")); len(f) >= 3 {
		allocated := parseUint(f[0])
		unused := parseUint(f[1])
		k.FileHandlesMax = parseUint(f[2])
		if allocated >= unused {
			k.FileHandles = allocated - unused
		} else {
			k.FileHandles = allocated
		}
	}
	return k, raw
}
