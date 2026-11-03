package sysinfo

import (
	"context"
	"net"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type netCounters struct {
	rx uint64
	tx uint64
}

type wirelessStat struct {
	quality int
	signal  int
}

type wifiExtra struct {
	ssid    string
	bitrate string
	at      time.Time
}

var virtualPrefixes = []string{
	"lo", "veth", "docker", "br-", "virbr", "vnet", "tap", "tun",
	"cni", "flannel", "kube", "cali", "dummy", "ifb", "vxlan", "wg",
}

func isVirtualIface(name string) bool {
	for _, p := range virtualPrefixes {
		if name == p || strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

func readNetDev() map[string]netCounters {
	out := map[string]netCounters{}
	for _, line := range strings.Split(string(readFile("/proc/net/dev")), "\n") {
		p := strings.SplitN(line, ":", 2)
		if len(p) != 2 {
			continue
		}
		name := strings.TrimSpace(p[0])
		f := strings.Fields(p[1])
		if len(f) < 16 {
			continue
		}
		out[name] = netCounters{rx: parseUint(f[0]), tx: parseUint(f[8])}
	}
	return out
}

func readWireless() map[string]wirelessStat {
	out := map[string]wirelessStat{}
	for _, line := range strings.Split(string(readFile("/proc/net/wireless")), "\n") {
		p := strings.SplitN(line, ":", 2)
		if len(p) != 2 || strings.Contains(line, "face") {
			continue
		}
		name := strings.TrimSpace(p[0])
		f := strings.Fields(p[1])
		if len(f) < 3 {
			continue
		}
		q := int(parseFloat(strings.TrimSuffix(f[1], ".")))
		sig := int(parseFloat(strings.TrimSuffix(f[2], ".")))
		out[name] = wirelessStat{quality: q, signal: sig}
	}
	return out
}

func interfaceAddrs() map[string][]string {
	out := map[string][]string{}
	ifaces, err := net.Interfaces()
	if err != nil {
		return out
	}
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		var v4, v6 []string
		for _, a := range addrs {
			ipnet, ok := a.(*net.IPNet)
			if !ok || ipnet.IP.IsLoopback() {
				continue
			}
			ip := ipnet.IP
			if ip.To4() != nil {
				v4 = append(v4, ip.String())
			} else if ip.IsGlobalUnicast() {
				v6 = append(v6, ip.String())
			}
		}
		list := append(v4, v6...)
		if len(list) > 2 {
			list = list[:2]
		}
		out[iface.Name] = list
	}
	return out
}

var (
	iwOnce sync.Once
	iwPath string
)

func findIW() string {
	iwOnce.Do(func() {
		for _, p := range []string{"/sbin/iw", "/usr/sbin/iw", "/usr/bin/iw"} {
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				iwPath = p
				return
			}
		}
		if p, err := exec.LookPath("iw"); err == nil {
			iwPath = p
		}
	})
	return iwPath
}

func lookupWifi(ctx context.Context, iface string, cache map[string]wifiExtra, now time.Time) (string, string) {
	if c, ok := cache[iface]; ok && now.Sub(c.at) < 15*time.Second {
		return c.ssid, c.bitrate
	}
	ssid, bitrate := "", ""
	iw := findIW()
	if iw != "" {
		cctx, cancel := context.WithTimeout(ctx, 900*time.Millisecond)
		defer cancel()
		out, err := exec.CommandContext(cctx, iw, "dev", iface, "link").Output()
		if err == nil {
			for _, line := range strings.Split(string(out), "\n") {
				line = strings.TrimSpace(line)
				switch {
				case strings.HasPrefix(line, "SSID:"):
					ssid = strings.TrimSpace(strings.TrimPrefix(line, "SSID:"))
				case strings.HasPrefix(line, "tx bitrate:"):
					bitrate = strings.TrimSpace(strings.TrimPrefix(line, "tx bitrate:"))
				}
			}
		}
	}
	if ssid == "" {
		cctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
		if out, err := exec.CommandContext(cctx, "iwgetid", "-r", iface).Output(); err == nil {
			ssid = strings.TrimSpace(string(out))
		}
		cancel()
	}
	if bitrate == "" {
		if v := readInt("/sys/class/net/" + iface + "/speed"); v > 0 {
			bitrate = strconv.FormatInt(v, 10) + " Mbps"
		}
	}
	cache[iface] = wifiExtra{ssid: ssid, bitrate: bitrate, at: now}
	return ssid, bitrate
}

func collectNets(ctx context.Context, raw, prev map[string]netCounters, elapsed float64, wifiCache map[string]wifiExtra, now time.Time) []Net {
	addrs := interfaceAddrs()
	wireless := readWireless()

	out := make([]Net, 0, len(raw))
	for name, c := range raw {
		if isVirtualIface(name) {
			continue
		}
		n := Net{Name: name, Addrs: addrs[name], RxTotal: c.rx, TxTotal: c.tx}
		if p, ok := prev[name]; ok && elapsed > 0 {
			if c.rx >= p.rx {
				n.RxRate = float64(c.rx-p.rx) / elapsed
			}
			if c.tx >= p.tx {
				n.TxRate = float64(c.tx-p.tx) / elapsed
			}
		}
		if st, err := os.Stat("/sys/class/net/" + name + "/wireless"); err == nil && st.IsDir() {
			n.Wireless = true
			if w, ok := wireless[name]; ok {
				n.Quality = w.quality
				n.Signal = w.signal
			}
			n.SSID, n.Bitrate = lookupWifi(ctx, name, wifiCache, now)
		}
		if len(n.Addrs) == 0 && n.RxTotal == 0 && n.TxTotal == 0 && !n.Wireless {
			continue
		}
		out = append(out, n)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Wireless != out[j].Wireless {
			return out[i].Wireless
		}
		return out[i].Name < out[j].Name
	})
	return out
}
