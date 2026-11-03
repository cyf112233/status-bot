// Package sysinfo 从 /proc 与 /sys 采集主机状态，后台定频采样，
// 渲染侧只读取不可变快照，避免高并发时反复扫描系统。
package sysinfo

import "time"

type Snapshot struct {
	At        time.Time
	Host      Host
	CPU       CPU
	Memory    Memory
	GPUs      []GPU
	Disks     []Disk
	Nets      []Net
	IO        IO
	ProcsCPU  []Proc
	ProcsMem  []Proc
	ProcCount int
}

type Host struct {
	Hostname string
	OS       string
	Kernel   string
	Arch     string
	Virt     string
	Uptime   time.Duration
	BootTime time.Time
	Load1    float64
	Load5    float64
	Load15   float64
}

type CPU struct {
	Model   string
	Cores   int
	Usage   float64
	PerCore []float64
	FreqMHz float64
	TempC   float64
	FanRPM  int
}

type Memory struct {
	Total     uint64
	Available uint64
	Used      uint64
	Cached    uint64
	SwapTotal uint64
	SwapUsed  uint64
}

type GPU struct {
	Name       string
	Util       float64
	MemUsed    uint64
	MemTotal   uint64
	TempC      float64
	PowerW     float64
	PowerLimit float64
}

type Disk struct {
	Device string
	Mount  string
	FS     string
	Total  uint64
	Used   uint64
	Free   uint64
}

type Net struct {
	Name     string
	Addrs    []string
	RxRate   float64
	TxRate   float64
	RxTotal  uint64
	TxTotal  uint64
	Wireless bool
	SSID     string
	Quality  int
	Signal   int
	Bitrate  string
}

type IO struct {
	ReadBps   float64
	WriteBps  float64
	ReadIOPS  float64
	WriteIOPS float64
	Util      float64
}

type Proc struct {
	PID  int
	Name string
	CPU  float64
	RSS  uint64
}
