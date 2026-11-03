package sysinfo

import (
	"os"
	"sort"
	"strings"
	"syscall"
)

var allowedFS = map[string]bool{
	"ext2": true, "ext3": true, "ext4": true, "xfs": true, "btrfs": true,
	"zfs": true, "f2fs": true, "vfat": true, "exfat": true, "ntfs": true,
	"ntfs3": true, "fuseblk": true, "overlay": true, "jfs": true,
	"reiserfs": true, "bcachefs": true, "nfs": true, "nfs4": true,
	"cifs": true, "smb3": true,
}

func collectDisks(limit int) []Disk {
	byDev := map[string]Disk{}
	var order []string

	for _, line := range strings.Split(string(readFile("/proc/mounts")), "\n") {
		f := strings.Fields(line)
		if len(f) < 3 {
			continue
		}
		device := decodeMount(f[0])
		mount := decodeMount(f[1])
		fs := f[2]
		if !allowedFS[fs] {
			continue
		}
		if mount == "" {
			continue
		}
		skip := false
		for _, p := range []string{"/proc", "/sys", "/dev/", "/run/", "/snap/", "/var/lib/docker", "/var/lib/containers", "/var/lib/kubelet"} {
			if mount == strings.TrimRight(p, "/") || strings.HasPrefix(mount, p) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		if fi, err := os.Stat(mount); err != nil || !fi.IsDir() {
			continue
		}
		var st syscall.Statfs_t
		if err := syscall.Statfs(mount, &st); err != nil || st.Bsize <= 0 || st.Blocks == 0 {
			continue
		}
		bs := uint64(st.Bsize)
		total := st.Blocks * bs
		used := (st.Blocks - st.Bfree) * bs
		free := st.Bavail * bs
		if total == 0 || used > total {
			continue
		}
		d := Disk{Device: device, Mount: mount, FS: fs, Total: total, Used: used, Free: free}
		key := device
		if key == "overlay" || key == "none" || key == "" {
			key = "mount:" + mount
		}
		if old, ok := byDev[key]; ok {
			// 同一设备的 bind mount，保留路径更短、更接近根的那个
			if len(old.Mount) <= len(mount) {
				continue
			}
		} else {
			order = append(order, key)
		}
		byDev[key] = d
	}

	out := make([]Disk, 0, len(byDev))
	for _, k := range order {
		out = append(out, byDev[k])
	}
	sort.Slice(out, func(i, j int) bool {
		if (out[i].Mount == "/") != (out[j].Mount == "/") {
			return out[i].Mount == "/"
		}
		return out[i].Mount < out[j].Mount
	})
	if limit > 0 && len(out) > limit {
		root := out[0]
		rest := out[1:]
		sort.Slice(rest, func(i, j int) bool { return rest[i].Total > rest[j].Total })
		chosen := append([]Disk{root}, rest[:limit-1]...)
		sort.Slice(chosen, func(i, j int) bool {
			if (chosen[i].Mount == "/") != (chosen[j].Mount == "/") {
				return chosen[i].Mount == "/"
			}
			return chosen[i].Mount < chosen[j].Mount
		})
		out = chosen
	}
	return out
}
