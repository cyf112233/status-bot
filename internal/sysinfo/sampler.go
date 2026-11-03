package sysinfo

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// Sampler 以固定频率采样并发布不可变快照。
type Sampler struct {
	interval time.Duration
	topN     int
	maxDisks int

	snap atomic.Pointer[Snapshot]

	mu       sync.Mutex
	lastAt   time.Time
	prevCPU  []cpuTimes
	prevNet  map[string]netCounters
	prevIO   map[string]diskIORaw
	prevProc map[int]uint64

	wifiCache map[string]wifiExtra
	gpus      []GPU
	gpuAt     time.Time
	gpuTTL    time.Duration

	cpuModel string
}

func New(interval time.Duration, topN, maxDisks int) *Sampler {
	if interval < 300*time.Millisecond {
		interval = time.Second
	}
	return &Sampler{
		interval:  interval,
		topN:      topN,
		maxDisks:  maxDisks,
		prevNet:   map[string]netCounters{},
		prevIO:    map[string]diskIORaw{},
		prevProc:  map[int]uint64{},
		wifiCache: map[string]wifiExtra{},
		cpuModel:  cpuModel(),
	}
}

// Start 先做两次采样（获得真实 CPU 差值），随后后台定时采样。
func (s *Sampler) Start(ctx context.Context) {
	s.SampleOnce(ctx)
	go func() {
		t := time.NewTicker(s.interval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				s.mu.Lock()
				s.sample(ctx)
				s.mu.Unlock()
			}
		}
	}()
}

// SampleOnce 立即采两次，返回第二次的结果。
func (s *Sampler) SampleOnce(ctx context.Context) *Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lastAt.IsZero() {
		s.sample(ctx)
	}
	select {
	case <-time.After(400 * time.Millisecond):
	case <-ctx.Done():
	}
	return s.sample(ctx)
}

func (s *Sampler) Snapshot() *Snapshot { return s.snap.Load() }

func (s *Sampler) sample(ctx context.Context) *Snapshot {
	now := time.Now()
	elapsed := s.interval.Seconds()
	if !s.lastAt.IsZero() {
		elapsed = now.Sub(s.lastAt).Seconds()
		if elapsed <= 0.01 {
			elapsed = 0.01
		}
		if elapsed > 10 {
			elapsed = 10
		}
	}

	curCPU := readCPUTimes()
	usage := 0.0
	perCore := make([]float64, 0, len(curCPU))
	if len(s.prevCPU) == len(curCPU) && len(curCPU) > 0 {
		usage = cpuUsage(s.prevCPU[0], curCPU[0])
		for i := 1; i < len(curCPU); i++ {
			perCore = append(perCore, cpuUsage(s.prevCPU[i], curCPU[i]))
		}
	} else {
		for i := 1; i < len(curCPU); i++ {
			perCore = append(perCore, 0)
		}
	}
	cores := len(curCPU) - 1
	if cores <= 0 {
		cores = runtime.NumCPU()
	}
	s.prevCPU = curCPU

	mem := collectMemory()
	disks := collectDisks(s.maxDisks)

	rawIO := readDiskIO()
	ioRate := diffIO(s.prevIO, rawIO, elapsed)
	s.prevIO = rawIO

	rawNet := readNetDev()
	nets := collectNets(ctx, rawNet, s.prevNet, elapsed, s.wifiCache, now)
	for k, v := range rawNet {
		s.prevNet[k] = v
	}

	if len(s.gpus) > 0 {
		s.gpuTTL = 2 * time.Second
	} else {
		s.gpuTTL = 8 * time.Second
	}
	if s.gpuAt.IsZero() || now.Sub(s.gpuAt) > s.gpuTTL {
		s.gpus = collectGPUs(ctx)
		s.gpuAt = now
	}

	procsCPU, procsMem, procCount := collectProcs(s.prevProc, elapsed, s.topN)

	snap := &Snapshot{
		At:   now,
		Host: collectHost(now),
		CPU: CPU{
			Model:   s.cpuModel,
			Cores:   cores,
			Usage:   usage,
			PerCore: perCore,
			FreqMHz: cpuFreqMHz(),
			TempC:   cpuTemp(),
			FanRPM:  cpuFanRPM(),
		},
		Memory:    mem,
		GPUs:      append([]GPU(nil), s.gpus...),
		Disks:     disks,
		Nets:      nets,
		IO:        IO{ReadBps: ioRate.readBps, WriteBps: ioRate.writeBps, ReadIOPS: ioRate.readIOPS, WriteIOPS: ioRate.writeIOPS, Util: ioRate.util},
		ProcsCPU:  procsCPU,
		ProcsMem:  procsMem,
		ProcCount: procCount,
	}
	s.snap.Store(snap)
	s.lastAt = now
	return snap
}
