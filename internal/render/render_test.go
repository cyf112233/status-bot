package render

import (
	"bytes"
	"image/png"
	"os"
	"testing"
	"time"

	"statusbot/internal/config"
	"statusbot/internal/sysinfo"
)

func fakeSnapshot() *sysinfo.Snapshot {
	per := make([]float64, 64)
	for i := range per {
		per[i] = float64((i*7)%95) + 3
	}
	cpuProcs := make([]sysinfo.Proc, 8)
	memProcs := make([]sysinfo.Proc, 8)
	for i := 0; i < 8; i++ {
		cpuProcs[i] = sysinfo.Proc{PID: 1000 + i, Name: []string{"ffmpeg", "postgres: writer", "node /srv/app", "mysqld", "redis-server", "containerd", "java", "nginx: worker"}[i], CPU: float64(80 - i*9), RSS: uint64(120-i*10) << 20}
		memProcs[i] = sysinfo.Proc{PID: 2000 + i, Name: []string{"mysql", "chrome", "postgres", "node", "java", "containerd", "redis-server", "systemd-journal"}[i], CPU: float64(3), RSS: uint64(900-i*50) << 20}
	}
	var disks []sysinfo.Disk
	for i := 0; i < 8; i++ {
		disks = append(disks, sysinfo.Disk{
			Device: "/dev/nvme0n1p" + string(rune('1'+i)),
			Mount:  []string{"/", "/boot", "/data", "/var/lib/mysql", "/home", "/srv", "/backup", "/mnt/very-long-name"}[i],
			FS:     "ext4",
			Total:  uint64(512<<30) + uint64(i)<<30,
			Used:   uint64(256+i*20) << 30,
			Free:   uint64(256-i*10) << 30,
			Inodes: 64 << 20, InodesUsed: uint64(2+i) << 20,
		})
	}
	return &sysinfo.Snapshot{
		At: time.Date(2026, 11, 2, 22, 31, 5, 0, time.Local),
		Host: sysinfo.Host{
			Hostname: "edge-node-01",
			OS:       "Debian GNU/Linux 12 (bookworm)",
			Kernel:   "6.1.0-18-amd64",
			Arch:     "x86_64",
			Virt:     "KVM",
			Uptime:   51*24*time.Hour + 14*time.Hour + 23*time.Minute,
			BootTime: time.Date(2026, 9, 12, 8, 8, 0, 0, time.Local),
			Load1:    3.42, Load5: 2.98, Load15: 2.71,
		},
		CPU: sysinfo.CPU{
			Model: "AMD EPYC 7B13 64-Core Processor", Cores: 64,
			Usage: 37.4, PerCore: per, FreqMHz: 3200, TempC: 64, FanRPM: 2100,
			UserPct: 12.3, SysPct: 4.1, IOWaitPct: 0.2, StealPct: 0, IdlePct: 83.4,
			Governor: "performance", CacheKB: 32768,
		},
		Memory: sysinfo.Memory{
			Total: 64 << 30, Available: 27 << 30, Used: 37 << 30, Cached: 12 << 30,
			Buffers: 1 << 30, CachedOnly: 9 << 30, SReclaimable: 2 << 30,
			SwapTotal: 8 << 30, SwapUsed: 1 << 30,
			Dirty: 64 << 20, Writeback: 0, Slab: 3 << 30, PageTables: 256 << 20, Committed: 48 << 30, Shmem: 512 << 20,
		},
		GPUs: []sysinfo.GPU{
			{Name: "NVIDIA GeForce RTX 4090", Util: 78, MemUsed: 18 << 30, MemTotal: 24 << 30, TempC: 71, PowerW: 320, PowerLimit: 450, Fan: 62, MemUtil: 44, ClockSM: 2520, ClockMem: 10501},
			{Name: "NVIDIA GeForce RTX 3090", Util: 12, MemUsed: 2 << 30, MemTotal: 24 << 30, TempC: 45, PowerW: 55, PowerLimit: 350, Fan: 34, MemUtil: 8, ClockSM: 1395, ClockMem: 9751},
		},
		Disks: disks,
		Nets: []sysinfo.Net{
			{Name: "eth0", Addrs: []string{"10.0.0.12", "fd00::12"}, RxRate: 12.4 * (1 << 20), TxRate: 2.1 * (1 << 20), RxTotal: 900 << 30, TxTotal: 120 << 30},
			{Name: "wlan0", Addrs: []string{"192.168.1.20"}, RxRate: 640 << 10, TxRate: 40 << 10, Wireless: true, SSID: "Home-WiFi-5G", Quality: 62, Signal: -51, Bitrate: "866.7 MBit/s"},
			{Name: "eth1", Addrs: []string{"172.16.0.5"}, RxRate: 1.2 * (1 << 20), TxRate: 300 << 10},
		},
		NetStats: sysinfo.NetStats{
			TCPEstablished: 128, TCPListen: 14, TCPTimeWait: 37, TCPTotal: 245,
			Gateway: "192.168.1.1", DNS: []string{"223.5.5.5", "119.29.29.29"},
			RxTotal: 900 << 30, TxTotal: 120 << 30,
		},
		IO: sysinfo.IO{
			ReadBps: 120 << 20, WriteBps: 34 << 20, ReadIOPS: 1200, WriteIOPS: 480, Util: 63.2,
			Disks: []sysinfo.DiskIO{
				{Name: "nvme0n1", ReadBps: 80 << 20, WriteBps: 20 << 20, Util: 72},
				{Name: "nvme1n1", ReadBps: 30 << 20, WriteBps: 12 << 20, Util: 41},
				{Name: "sda", ReadBps: 10 << 20, WriteBps: 2 << 20, Util: 12},
			},
		},
		Kernel: sysinfo.Kernel{
			CTXPerSec: 23456, IntrPerSec: 18234, ForksPerSec: 12.4,
			CTXTotal: 2345678901, IntrTotal: 1823456789, ForksTotal: 45678,
			Running: 3, Blocked: 0, Threads: 1289,
			FileHandles: 4096, FileHandlesMax: 9223372036854775807,
		},
		Sensors: []sysinfo.Sensor{
			{Name: "Package id 0 · coretemp", Value: 64, Unit: "°C"},
			{Name: "Composite · nvme", Value: 41, Unit: "°C"},
			{Name: "edge · amdgpu", Value: 58, Unit: "°C"},
			{Name: "CPU Fan · it8728", Value: 1850, Unit: "RPM"},
			{Name: "Sensor 2 · it8728", Value: 38, Unit: "°C"},
			{Name: "Sensor 3 · it8728", Value: 42, Unit: "°C"},
		},
		ProcsCPU: cpuProcs, ProcsMem: memProcs, ProcCount: 312,
	}
}

func TestRenderSynthetic(t *testing.T) {
	pngBytes, err := Render(fakeSnapshot(), config.Render{Width: 720, TopProcesses: 8})
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		t.Fatal(err)
	}
	b := img.Bounds()
	if b.Dx() != 720 {
		t.Fatalf("width = %d", b.Dx())
	}
	if b.Dy() < 1400 {
		t.Fatalf("height = %d, 内容可能被裁切", b.Dy())
	}
	if len(pngBytes) < 40*1024 {
		t.Fatalf("图片过小: %d", len(pngBytes))
	}
	if os.Getenv("DUMP_IMAGE") != "" {
		_ = os.WriteFile("/tmp/synthetic.png", pngBytes, 0o644)
	}
}

func TestRenderScaledWidth(t *testing.T) {
	pngBytes, err := Render(fakeSnapshot(), config.Render{Width: 1080, TopProcesses: 8})
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 1080 {
		t.Fatalf("width = %d, want 1080", img.Bounds().Dx())
	}
}

func TestFitNeverExceedsWidth(t *testing.T) {
	c := &cv{fonts: newFontSet()}
	for _, s := range []string{"短", "一个非常非常长的进程名称-with-english-and-1234567890", ""} {
		for _, size := range []float64{9.5, 12, 18.5, 36} {
			got := c.fit(s, 120, size, false)
			if w := c.measure(got, size, false); w > 120.01 {
				t.Fatalf("fit(%q) 宽度 %.2f 超过 120", s, w)
			}
		}
	}
}
