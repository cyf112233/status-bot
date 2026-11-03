package sysinfo

import (
	"os"
	"sort"
	"strconv"
	"strings"
)

func collectProcs(prev map[int]uint64, elapsed float64, topN int) ([]Proc, []Proc, int) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, nil, 0
	}
	pageSize := uint64(os.Getpagesize())
	list := make([]Proc, 0, len(entries))
	ticks := make(map[int]uint64, len(entries))

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(e.Name())
		if err != nil || pid <= 0 {
			continue
		}
		b := readFile("/proc/" + e.Name() + "/stat")
		if len(b) == 0 {
			continue
		}
		s := string(b)
		open := strings.IndexByte(s, '(')
		closeIdx := strings.LastIndexByte(s, ')')
		if open < 0 || closeIdx <= open {
			continue
		}
		name := s[open+1 : closeIdx]
		fields := strings.Fields(s[closeIdx+1:])
		if len(fields) < 13 {
			continue
		}
		utime := parseUint(fields[11])
		stime := parseUint(fields[12])
		total := utime + stime
		ticks[pid] = total
		cpu := 0.0
		if p, ok := prev[pid]; ok && total >= p && elapsed > 0 {
			cpu = float64(total-p) / elapsed // CLK_TCK = 100
			if cpu > 999 {
				cpu = 999
			}
		}
		var rss uint64
		if sm := strings.Fields(string(readFile("/proc/" + e.Name() + "/statm"))); len(sm) >= 2 {
			rss = parseUint(sm[1]) * pageSize
		}
		if len(name) > 40 {
			name = name[:40]
		}
		list = append(list, Proc{PID: pid, Name: name, CPU: cpu, RSS: rss})
	}

	for pid, t := range ticks {
		prev[pid] = t
	}
	for pid := range prev {
		if _, ok := ticks[pid]; !ok {
			delete(prev, pid)
		}
	}

	byCPU := append([]Proc(nil), list...)
	sort.Slice(byCPU, func(i, j int) bool {
		if byCPU[i].CPU == byCPU[j].CPU {
			return byCPU[i].RSS > byCPU[j].RSS
		}
		return byCPU[i].CPU > byCPU[j].CPU
	})
	byMem := append([]Proc(nil), list...)
	sort.Slice(byMem, func(i, j int) bool {
		if byMem[i].RSS == byMem[j].RSS {
			return byMem[i].CPU > byMem[j].CPU
		}
		return byMem[i].RSS > byMem[j].RSS
	})
	if topN > 0 && len(byCPU) > topN {
		byCPU = byCPU[:topN]
	}
	if topN > 0 && len(byMem) > topN {
		byMem = byMem[:topN]
	}
	return byCPU, byMem, len(list)
}
