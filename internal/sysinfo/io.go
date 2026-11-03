package sysinfo

import (
	"regexp"
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

var (
	physicalDisk = regexp.MustCompile(`^(sd[a-z]+|hd[a-z]+|vd[a-z]+|xvd[a-z]+|nvme[0-9]+n[0-9]+|mmcblk[0-9]+|md[0-9]+)$`)
)

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

func diffIO(prev, cur map[string]diskIORaw, elapsed float64) ioRate {
	if elapsed <= 0 {
		return ioRate{}
	}
	var r ioRate
	var n float64
	for name, c := range cur {
		p, ok := prev[name]
		if !ok {
			continue
		}
		n++
		if c.readSectors >= p.readSectors {
			r.readBps += float64(c.readSectors-p.readSectors) * 512 / elapsed
		}
		if c.writeSectors >= p.writeSectors {
			r.writeBps += float64(c.writeSectors-p.writeSectors) * 512 / elapsed
		}
		if c.readIOs >= p.readIOs {
			r.readIOPS += float64(c.readIOs-p.readIOs) / elapsed
		}
		if c.writeIOs >= p.writeIOs {
			r.writeIOPS += float64(c.writeIOs-p.writeIOs) / elapsed
		}
		if c.ioMillis >= p.ioMillis && elapsed > 0 {
			r.util += float64(c.ioMillis-p.ioMillis) / (elapsed * 1000) * 100
		}
	}
	if n > 1 {
		r.util /= n
	}
	if r.util > 100 {
		r.util = 100
	}
	return r
}
