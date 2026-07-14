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

func TestResolvePresets(t *testing.T) {
	cases := []struct {
		key     string
		wantRef string
	}{
		{"qwen3.6-moe", "unsloth/Qwen3.6-35B-A3B-GGUF:UD-Q4_K_XL"},
		{"gemma4-moe", "unsloth/gemma-4-26B-A4B-it-GGUF:UD-Q4_K_M"},
		{"gemma4-4b", "unsloth/gemma-4-E4B-it-GGUF:Q4_K_M"},
	}
	for _, c := range cases {
		spec, found := Resolve(c.key)
		if !found {
			t.Errorf("%s: expected found=true", c.key)
		}
		if spec.HFRef != c.wantRef {
			t.Errorf("%s: got HFRef %s want %s", c.key, spec.HFRef, c.wantRef)
		}
	}
}

func TestResolveRawRefPassesThrough(t *testing.T) {
	raw := "unsloth/Foo-GGUF:Q4_K_M"
	spec, found := Resolve(raw)
	if found {
		t.Errorf("raw ref: expected found=false, got true")
	}
	if spec.HFRef != raw {
		t.Errorf("raw ref: got HFRef %s want %s", spec.HFRef, raw)
	}
	if spec.Name != raw {
		t.Errorf("raw ref: got Name %s want %s", spec.Name, raw)
	}
}

func TestResolveUnknownBareWordPassesThrough(t *testing.T) {
	word := "some-unknown-word"
	spec, found := Resolve(word)
	if found {
		t.Errorf("unknown word: expected found=false, got true")
	}
	if spec.HFRef != word {
		t.Errorf("unknown word: got HFRef %s want %s", spec.HFRef, word)
	}
}

func TestPresetKeysSorted(t *testing.T) {
	keys := PresetKeys()
	if len(keys) != len(Presets) {
		t.Fatalf("got %d keys want %d", len(keys), len(Presets))
	}
	for i := 1; i < len(keys); i++ {
		if keys[i-1] >= keys[i] {
			t.Errorf("keys not sorted: %v", keys)
		}
	}
}
