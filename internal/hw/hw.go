package hw

import (
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

type CommandRunner func(name string, args ...string) (string, error)

func ExecRunner(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).Output()
	return string(out), err
}

type Info struct {
	OS, Arch  string
	RAMGB     int
	VRAMGB    int
	HasNvidia bool
}

// IsSpark: DGX Spark is the only linux/arm64 + NVIDIA target we support.
func (i Info) IsSpark() bool { return i.OS == "linux" && i.Arch == "arm64" && i.HasNvidia }

func ramGBFromSysctl(out string) int {
	b, _ := strconv.ParseInt(strings.TrimSpace(out), 10, 64)
	return int(b / (1 << 30))
}

func ramGBFromMeminfo(out string) int {
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "MemTotal:") {
			f := strings.Fields(line)
			if len(f) >= 2 {
				kb, _ := strconv.ParseInt(f[1], 10, 64)
				return int(kb / (1 << 20))
			}
		}
	}
	return 0
}

func vramGBFromSmi(out string) int {
	first := strings.TrimSpace(strings.Split(strings.TrimSpace(out), "\n")[0])
	mib, _ := strconv.ParseInt(first, 10, 64)
	return int(mib / 1024)
}

func Detect(run CommandRunner) (Info, error) {
	info := Info{OS: runtime.GOOS, Arch: runtime.GOARCH}
	switch info.OS {
	case "darwin":
		if out, err := run("sysctl", "-n", "hw.memsize"); err == nil {
			info.RAMGB = ramGBFromSysctl(out)
		}
	default:
		if out, err := run("cat", "/proc/meminfo"); err == nil {
			info.RAMGB = ramGBFromMeminfo(out)
		}
	}
	if out, err := run("nvidia-smi", "--query-gpu=memory.total", "--format=csv,noheader,nounits"); err == nil {
		info.HasNvidia = true
		info.VRAMGB = vramGBFromSmi(out)
	}
	return info, nil
}
