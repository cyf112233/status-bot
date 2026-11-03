package sysinfo

import "strings"

// collectSensors 读取 hwmon 下的温度与风扇。
func collectSensors() []Sensor {
	hwmons, _ := filepathGlob("/sys/class/hwmon/hwmon*")
	var out []Sensor
	for _, h := range hwmons {
		chip := readTrim(h + "/name")
		if chip == "" {
			chip = "sensor"
		}
		temps, _ := filepathGlob(h + "/temp*_input")
		for _, p := range temps {
			v := float64(readInt(p)) / 1000
			if v < 1 || v > 150 {
				continue
			}
			label := readTrim(strings.TrimSuffix(p, "_input") + "_label")
			name := label
			if name != "" {
				name = name + " · " + chip
			} else {
				name = chip + " 温度"
			}
			out = append(out, Sensor{Name: name, Value: v, Unit: "°C"})
		}
		fans, _ := filepathGlob(h + "/fan*_input")
		for _, p := range fans {
			v := float64(readInt(p))
			if v <= 0 || v > 30000 {
				continue
			}
			label := readTrim(strings.TrimSuffix(p, "_input") + "_label")
			name := label
			if name != "" {
				name = name + " · " + chip
			} else {
				name = chip + " 风扇"
			}
			out = append(out, Sensor{Name: name, Value: v, Unit: "RPM"})
		}
	}
	if len(out) > 12 {
		out = out[:12]
	}
	return out
}
