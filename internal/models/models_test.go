package models

import (
	"testing"

	"github.com/hyperstudyio/hyperstudy-agent-demo/internal/hw"
)

func TestLadder(t *testing.T) {
	cases := []struct {
		name string
		info hw.Info
		want string
	}{
		{"8GB laptop", hw.Info{OS: "darwin", Arch: "arm64", RAMGB: 8}, "unsloth/Qwen3-4B-Instruct-2507-GGUF:Q4_K_M"},
		{"32GB mac", hw.Info{OS: "darwin", Arch: "arm64", RAMGB: 32}, "unsloth/Qwen3-14B-GGUF:Q4_K_M"},
		{"3090 box", hw.Info{OS: "linux", Arch: "amd64", RAMGB: 64, HasNvidia: true, VRAMGB: 24}, "unsloth/GLM-4.7-Flash-GGUF:UD-Q4_K_XL"},
		{"spark", hw.Info{OS: "linux", Arch: "arm64", RAMGB: 119, HasNvidia: true, VRAMGB: 0}, "unsloth/Qwen3.6-35B-A3B-GGUF:UD-Q4_K_XL"},
		{"small gpu falls back to RAM tier", hw.Info{OS: "linux", Arch: "amd64", RAMGB: 32, HasNvidia: true, VRAMGB: 8}, "unsloth/Qwen3-14B-GGUF:Q4_K_M"},
	}
	for _, c := range cases {
		if got := Pick(c.info).HFRef; got != c.want {
			t.Errorf("%s: got %s want %s", c.name, got, c.want)
		}
	}
}
