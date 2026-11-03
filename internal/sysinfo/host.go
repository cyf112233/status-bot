package sysinfo

import (
	"os"
	"runtime"
	"strings"
	"time"
)

func collectHost(now time.Time) Host {
	h := Host{
		Hostname: hostname(),
		OS:       osPrettyName(),
		Kernel:   readTrim("/proc/sys/kernel/osrelease"),
		Arch:     archName(runtime.GOARCH),
		Virt:     detectVirt(),
		Timezone: hostTimezone(now),
	}
	if up := readTrim("/proc/uptime"); up != "" {
		if f := strings.Fields(up); len(f) > 0 {
			h.Uptime = time.Duration(parseFloat(f[0]) * float64(time.Second))
		}
	}
	if bt := readBtime(); !bt.IsZero() {
		h.BootTime = bt
	} else if h.Uptime > 0 {
		h.BootTime = now.Add(-h.Uptime)
	}
	if la := readTrim("/proc/loadavg"); la != "" {
		f := strings.Fields(la)
		if len(f) >= 3 {
			h.Load1 = parseFloat(f[0])
			h.Load5 = parseFloat(f[1])
			h.Load15 = parseFloat(f[2])
		}
	}
	return h
}

func hostTimezone(now time.Time) string {
	if tz := readTrim("/etc/timezone"); tz != "" {
		return tz
	}
	return now.Location().String()
}

func hostname() string {
	if h, err := os.Hostname(); err == nil && h != "" {
		return h
	}
	return "unknown"
}

func osPrettyName() string {
	for _, line := range strings.Split(string(readFile("/etc/os-release")), "\n") {
		if strings.HasPrefix(line, "PRETTY_NAME=") {
			v := strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), string([]rune{rune(34), rune(39)}))
			if v != "" {
				return v
			}
		}
	}
	if s := readTrim("/etc/issue"); s != "" {
		return strings.SplitN(s, "\n", 2)[0]
	}
	return "Linux"
}

func archName(a string) string {
	switch a {
	case "amd64":
		return "x86_64"
	case "386":
		return "i386"
	case "arm64":
		return "aarch64"
	case "arm":
		return "arm"
	default:
		return a
	}
}

func readBtime() time.Time {
	for _, line := range strings.Split(string(readFile("/proc/stat")), "\n") {
		if strings.HasPrefix(line, "btime ") {
			if sec := parseUint(strings.TrimSpace(strings.TrimPrefix(line, "btime "))); sec > 0 {
				return time.Unix(int64(sec), 0)
			}
		}
	}
	return time.Time{}
}

func detectVirt() string {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return "Docker"
	}
	cg := string(readFile("/proc/1/cgroup"))
	if strings.Contains(cg, "docker") || strings.Contains(cg, "kubepods") {
		return "Docker"
	}
	if strings.Contains(cg, "lxc") {
		return "LXC"
	}
	if strings.Contains(string(readFile("/proc/1/environ")), "container=") {
		return "Container"
	}
	product := readTrim("/sys/class/dmi/id/product_name")
	lower := strings.ToLower(product)
	switch {
	case strings.Contains(lower, "kvm"):
		return "KVM"
	case strings.Contains(lower, "vmware"):
		return "VMware"
	case strings.Contains(lower, "virtualbox"):
		return "VirtualBox"
	case strings.Contains(lower, "qemu"):
		return "QEMU"
	case strings.Contains(lower, "hyper-v") || strings.Contains(lower, "virtual machine"):
		return "Hyper-V"
	}
	if product != "" {
		return product
	}
	return ""
}
