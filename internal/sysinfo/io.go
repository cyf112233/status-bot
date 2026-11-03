package sysinfo

import (
	"regexp"
	"sort"
	"strings"
)

type diskIORaw struct {
	readSectors  uint64
	writeSectors uint64
	readIOs      uint64
	writeIOs     uint64
	ioMillis     uint64
}

type ioRate struct {
	readBps   float64
	writeBps  float64
	readIOPS  float64
	writeIOPS float64
	util      float64
}

var physicalDisk = regexp.MustCompile("^(sd[a-z]+|hd[a-z]+|vd[a-z]+|xvd[a-z]+|nvme[0-9]+n[0-9]+|mmcblk[0-9]+|md[0-9]+)$")

func readDiskIO() map[string]diskIORaw {
	out := map[string]diskIORaw{}
	for _, line := range strings.Split(string(readFile("/proc/diskstats")), "\n") {
		f := strings.Fields(line)
		if len(f) < 14 {
			continue
		}
		name := f[2]
		if !physicalDisk.MatchString(name) {
			continue
		}
		out[name] = diskIORaw{
			readSectors:  parseUint(f[5]),
			writeSectors: parseUint(f[9]),
			readIOs:      parseUint(f[3]),
			writeIOs:     parseUint(f[7]),
			ioMillis:     parseUint(f[12]),
		}
	}
	return out
}

// diffIO 返回汇总值与每块磁盘的明细，按吞吐量降序。
func diffIO(prev, cur map[string]diskIORaw, elapsed float64) (ioRate, []DiskIO) {
	if elapsed <= 0 {
		return ioRate{}, nil
	}
	var r ioRate
	var n float64
	var list []DiskIO
	for name, c := range cur {
		p, ok := prev[name]
		if !ok {
			continue
		}
		n++
		di := DiskIO{Name: name}
		if c.readSectors >= p.readSectors {
			di.ReadBps = float64(c.readSectors-p.readSectors) * 512 / elapsed
		}
		if c.writeSectors >= p.writeSectors {
			di.WriteBps = float64(c.writeSectors-p.writeSectors) * 512 / elapsed
		}
		if c.readIOs >= p.readIOs {
			di.ReadIOPS = float64(c.readIOs-p.readIOs) / elapsed
		}
		if c.writeIOs >= p.writeIOs {
			di.WriteIOPS = float64(c.writeIOs-p.writeIOs) / elapsed
		}
		if c.ioMillis >= p.ioMillis {
			di.Util = float64(c.ioMillis-p.ioMillis) / (elapsed * 1000) * 100
		}
		if di.Util > 100 {
			di.Util = 100
		}
		r.readBps += di.ReadBps
		r.writeBps += di.WriteBps
		r.readIOPS += di.ReadIOPS
		r.writeIOPS += di.WriteIOPS
		r.util += di.Util
		list = append(list, di)
	}
	if n > 1 {
		r.util /= n
	}
	if r.util > 100 {
		r.util = 100
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].ReadBps+list[i].WriteBps > list[j].ReadBps+list[j].WriteBps
	})
	return r, list
}
