package sysinfo

import (
	"context"
	"testing"
	"time"
)

func TestSampleOnce(t *testing.T) {
	s := New(500*time.Millisecond, 5, 3)
	snap := s.SampleOnce(context.Background())
	if snap == nil {
		t.Fatal("快照为空")
	}
	if snap.Memory.Total == 0 || snap.Memory.Used > snap.Memory.Total {
		t.Fatalf("内存数据异常: %+v", snap.Memory)
	}
	if snap.CPU.Cores < 1 {
		t.Fatalf("核心数异常: %d", snap.CPU.Cores)
	}
	if snap.CPU.Usage < 0 || snap.CPU.Usage > 100 {
		t.Fatalf("CPU 使用率越界: %f", snap.CPU.Usage)
	}
	for _, p := range snap.CPU.PerCore {
		if p < 0 || p > 100 {
			t.Fatalf("核心使用率越界: %f", p)
		}
	}
	if snap.ProcCount < 1 {
		t.Fatalf("进程数异常: %d", snap.ProcCount)
	}
	if snap.Host.Hostname == "" || snap.Host.Arch == "" {
		t.Fatalf("主机信息不完整: %+v", snap.Host)
	}
	if s.Snapshot() == nil {
		t.Fatal("Snapshot() 未发布")
	}
}
