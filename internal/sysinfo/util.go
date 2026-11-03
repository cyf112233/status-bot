package sysinfo

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func readFile(path string) []byte {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return b
}

func readTrim(path string) string {
	return strings.TrimSpace(string(readFile(path)))
}

func readInt(path string) int64 {
	v, _ := strconv.ParseInt(strings.TrimSpace(string(readFile(path))), 10, 64)
	return v
}

func readUint(path string) uint64 {
	v, _ := strconv.ParseUint(strings.TrimSpace(string(readFile(path))), 10, 64)
	return v
}

func parseUint(s string) uint64 {
	v, _ := strconv.ParseUint(strings.TrimSpace(s), 10, 64)
	return v
}

func parseFloat(s string) float64 {
	v, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return v
}

// decodeMount 处理 /proc/mounts 中的八进制转义。
func decodeMount(s string) string {
	if strings.IndexByte(s, 92) < 0 {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == 92 && i+3 < len(s) {
			if v, err := strconv.ParseUint(s[i+1:i+4], 8, 8); err == nil {
				b.WriteByte(byte(v))
				i += 3
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func filepathGlob(pattern string) ([]string, error) {
	return filepath.Glob(pattern)
}
