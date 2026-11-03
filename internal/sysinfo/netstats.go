package sysinfo

import (
	"strconv"
	"strings"
)

func collectNetStats(nets []Net) NetStats {
	var st NetStats
	for _, n := range nets {
		st.RxTotal += n.RxTotal
		st.TxTotal += n.TxTotal
	}
	st.Gateway = defaultGateway()
	st.DNS = dnsServers()
	for _, file := range []string{"/proc/net/tcp", "/proc/net/tcp6"} {
		lines := strings.Split(string(readFile(file)), "\n")
		for i, line := range lines {
			if i == 0 || strings.TrimSpace(line) == "" {
				continue
			}
			f := strings.Fields(line)
			if len(f) < 4 {
				continue
			}
			st.TCPTotal++
			switch f[3] {
			case "01":
				st.TCPEstablished++
			case "0A":
				st.TCPListen++
			case "06":
				st.TCPTimeWait++
			}
		}
	}
	return st
}

func defaultGateway() string {
	for i, line := range strings.Split(string(readFile("/proc/net/route")), "\n") {
		if i == 0 {
			continue
		}
		f := strings.Fields(line)
		if len(f) < 3 || f[1] != "00000000" || f[2] == "00000000" {
			continue
		}
		v, err := strconv.ParseUint(f[2], 16, 32)
		if err != nil {
			continue
		}
		b := []byte{byte(v), byte(v >> 8), byte(v >> 16), byte(v >> 24)}
		return strings.Join([]string{strconv.Itoa(int(b[0])), strconv.Itoa(int(b[1])), strconv.Itoa(int(b[2])), strconv.Itoa(int(b[3]))}, ".")
	}
	return ""
}

func dnsServers() []string {
	var out []string
	for _, line := range strings.Split(string(readFile("/etc/resolv.conf")), "\n") {
		f := strings.Fields(line)
		if len(f) >= 2 && f[0] == "nameserver" {
			out = append(out, f[1])
			if len(out) >= 2 {
				break
			}
		}
	}
	return out
}
