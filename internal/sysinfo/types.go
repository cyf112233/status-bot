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
	NetStats  NetStats
	IO        IO
	Kernel    Kernel
	Sensors   []Sensor
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
	Timezone string
}

type CPU struct {
	Model     string
	Cores     int
	Usage     float64
	PerCore   []float64
	FreqMHz   float64
	TempC     float64
	FanRPM    int
	UserPct   float64
	SysPct    float64
	IOWaitPct float64
	StealPct  float64
	IdlePct   float64
	Governor  string
	CacheKB   uint64
}

type Memory struct {
	Total        uint64
	Available    uint64
	Used         uint64
	Cached       uint64
	Buffers      uint64
	CachedOnly   uint64
	SReclaimable uint64
	SwapTotal    uint64
	SwapUsed     uint64
	Dirty        uint64
	Writeback    uint64
	Slab         uint64
	PageTables   uint64
	Committed    uint64
	Shmem        uint64
}

type GPU struct {
	Name       string
	Util       float64
	MemUsed    uint64
	MemTotal   uint64
	TempC      float64
	PowerW     float64
	PowerLimit float64
	Fan        float64
	FanRPM     int
	MemUtil    float64
	ClockSM    int
	ClockMem   int
}

type Disk struct {
	Device     string
	Mount      string
	FS         string
	Total      uint64
	Used       uint64
	Free       uint64
	Inodes     uint64
	InodesUsed uint64
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

type NetStats struct {
	TCPEstablished int
	TCPListen      int
	TCPTimeWait    int
	TCPTotal       int
	Gateway        string
	DNS            []string
	RxTotal        uint64
	TxTotal        uint64
}

type DiskIO struct {
	Name      string
	ReadBps   float64
	WriteBps  float64
	ReadIOPS  float64
	WriteIOPS float64
	Util      float64
}

type IO struct {
	ReadBps   float64
	WriteBps  float64
	ReadIOPS  float64
	WriteIOPS float64
	Util      float64
	Disks     []DiskIO
}

type Kernel struct {
	CTXPerSec      float64
	IntrPerSec     float64
	ForksPerSec    float64
	CTXTotal       uint64
	IntrTotal      uint64
	ForksTotal     uint64
	Running        int
	Blocked        int
	Threads        int
	FileHandles    uint64
	FileHandlesMax uint64
}

type Sensor struct {
	Name  string
	Value float64
	Unit  string
}

type Proc struct {
	PID  int
	Name string
	CPU  float64
	RSS  uint64
}
