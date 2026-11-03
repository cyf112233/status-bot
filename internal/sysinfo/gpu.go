package sysinfo

import (
	"context"
	"os/exec"
	"strings"
	"sync"
	"time"
)

var (
	smiOnce sync.Once
	smiPath string
)

func findSMI() string {
	smiOnce.Do(func() {
		if p, err := exec.LookPath("nvidia-smi"); err == nil {
			smiPath = p
		}
	})
	return smiPath
}

func collectGPUs(ctx context.Context) []GPU {
	if smi := findSMI(); smi != "" {
		if gpus := nvidiaGPUs(ctx, smi); len(gpus) > 0 {
			return gpus
		}
	}
	return amdGPUs()
}

func nvidiaGPUs(ctx context.Context, smi string) []GPU {
	cctx, cancel := context.WithTimeout(ctx, 1800*time.Millisecond)
	defer cancel()
	query := "name,utilization.gpu,memory.used,memory.total,temperature.gpu,power.draw,power.limit,fan.speed,memory.utilization,clocks.current.sm,clocks.current.memory"
	out, err := exec.CommandContext(cctx, smi, "--query-gpu="+query, "--format=csv,noheader,nounits").Output()
	if err != nil {
		out, err = exec.CommandContext(cctx, smi,
			"--query-gpu=name,utilization.gpu,memory.used,memory.total,temperature.gpu,power.draw,power.limit",
			"--format=csv,noheader,nounits").Output()
		if err != nil {
			return nil
		}
	}
	var gpus []GPU
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		f := strings.Split(line, ",")
		if len(f) < 7 {
			continue
		}
		g := GPU{
			Name:       strings.TrimSpace(f[0]),
			Util:       parseFloat(f[1]),
			MemUsed:    uint64(parseFloat(f[2])) * 1024 * 1024,
			MemTotal:   uint64(parseFloat(f[3])) * 1024 * 1024,
			TempC:      parseFloat(f[4]),
			PowerW:     parseFloat(f[5]),
			PowerLimit: parseFloat(f[6]),
		}
		if len(f) >= 11 {
			g.Fan = parseFloat(f[7])
			g.MemUtil = parseFloat(f[8])
			g.ClockSM = int(parseFloat(f[9]))
			g.ClockMem = int(parseFloat(f[10]))
		}
		gpus = append(gpus, g)
	}
	return gpus
}

func amdGPUs() []GPU {
	cards, _ := filepathGlob("/sys/class/drm/card[0-9]*")
	var gpus []GPU
	for _, card := range cards {
		dev := card + "/device"
		busy := readInt(dev + "/gpu_busy_percent")
		if busy <= 0 {
			continue
		}
		g := GPU{Util: float64(busy), Name: "AMD GPU"}
		if name := readTrim(dev + "/product_name"); name != "" {
			g.Name = name
		}
		g.MemUsed = readUint(dev + "/mem_info_vram_used")
		g.MemTotal = readUint(dev + "/mem_info_vram_total")
		hwmons, _ := filepathGlob(dev + "/hwmon/hwmon*")
		for _, h := range hwmons {
			if t := readInt(h + "/temp1_input"); t > 0 {
				g.TempC = float64(t) / 1000
			}
			if fan := readInt(h + "/fan1_input"); fan > 0 {
				g.FanRPM = int(fan)
			}
			if p := readInt(h + "/power1_average"); p > 0 {
				g.PowerW = float64(p) / 1e6
			}
			if p := readInt(h + "/power1_input"); p > 0 {
				g.PowerW = float64(p) / 1e6
			}
		}
		gpus = append(gpus, g)
	}
	return gpus
}
