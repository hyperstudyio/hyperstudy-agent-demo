package hw

import (
	"errors"
	"runtime"
	"testing"
)

func fake(resp map[string]string) CommandRunner {
	return func(name string, args ...string) (string, error) {
		if v, ok := resp[name]; ok {
			return v, nil
		}
		return "", errors.New("not found: " + name)
	}
}

func TestDetectMacRAM(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin-shaped fixture") // Detect uses runtime.GOOS; fixture matches darwin path
	}
	info, err := Detect(fake(map[string]string{"sysctl": "34359738368\n"}))
	if err != nil {
		t.Fatal(err)
	}
	if info.RAMGB != 32 {
		t.Fatalf("want 32GB, got %d", info.RAMGB)
	}
	if info.HasNvidia {
		t.Fatal("no nvidia-smi fixture → HasNvidia must be false")
	}
}

func TestParseHelpers(t *testing.T) {
	// OS-independent parsing helpers so CI on any platform covers both paths.
	if gb := ramGBFromMeminfo("MemTotal:       131072000 kB\nMemFree: 1 kB\n"); gb != 125 {
		t.Fatalf("meminfo parse: want 125, got %d", gb)
	}
	if gb := ramGBFromSysctl("34359738368\n"); gb != 32 {
		t.Fatalf("sysctl parse: want 32, got %d", gb)
	}
	if gb := vramGBFromSmi("24576\n"); gb != 24 {
		t.Fatalf("smi parse: want 24, got %d", gb)
	}
}

func TestIsSpark(t *testing.T) {
	spark := Info{OS: "linux", Arch: "arm64", HasNvidia: true}
	if !spark.IsSpark() {
		t.Fatal("linux/arm64 + nvidia should be Spark")
	}
	if (Info{OS: "linux", Arch: "amd64", HasNvidia: true}).IsSpark() {
		t.Fatal("amd64 is not Spark")
	}
}
