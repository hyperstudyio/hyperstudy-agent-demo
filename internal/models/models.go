package models

import (
	"sort"

	"github.com/hyperstudyio/hyperstudy-agent-demo/internal/hw"
)

type Spec struct {
	Name  string
	HFRef string
}

// Presets maps short, memorable names to HF-verified -hf refs. These are
// opt-in overrides for serve --model; the hardware ladder in Pick remains
// the no-flag default.
var Presets = map[string]Spec{
	"qwen3.6-moe": {Name: "Qwen3.6-35B-A3B (MoE)", HFRef: "unsloth/Qwen3.6-35B-A3B-GGUF:UD-Q4_K_XL"},
	"gemma4-moe":  {Name: "Gemma 4 26B-A4B (MoE)", HFRef: "unsloth/gemma-4-26B-A4B-it-GGUF:UD-Q4_K_M"},
	"gemma4-4b":   {Name: "Gemma 4 E4B", HFRef: "unsloth/gemma-4-E4B-it-GGUF:Q4_K_M"},
}

// PresetKeys returns the Presets keys sorted for stable help text.
func PresetKeys() []string {
	keys := make([]string, 0, len(Presets))
	for k := range Presets {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Resolve maps a --model flag value to a Spec. If flag names a known preset,
// it returns that preset's Spec and found=true. Otherwise it passes the flag
// through unchanged as both Name and HFRef (found=false) — this covers raw
// -hf refs (containing "/") and any other value the caller wants to try
// verbatim.
func Resolve(flag string) (Spec, bool) {
	if spec, ok := Presets[flag]; ok {
		return spec, true
	}
	return Spec{Name: flag, HFRef: flag}, false
}

// Pick implements the spec's model ladder. Order matters: Spark first (its
// VRAM often reads 0 — unified memory), then discrete-GPU tier, then RAM.
func Pick(info hw.Info) Spec {
	switch {
	case info.IsSpark():
		return Spec{Name: "Qwen3.6-35B-A3B (MoE)", HFRef: "unsloth/Qwen3.6-35B-A3B-GGUF:UD-Q4_K_XL"}
	case info.VRAMGB >= 24:
		return Spec{Name: "GLM-4.7-Flash (MoE)", HFRef: "unsloth/GLM-4.7-Flash-GGUF:UD-Q4_K_XL"}
	case info.RAMGB >= 16:
		return Spec{Name: "Qwen3-14B", HFRef: "unsloth/Qwen3-14B-GGUF:Q4_K_M"}
	default:
		return Spec{Name: "Qwen3-4B-Instruct-2507", HFRef: "unsloth/Qwen3-4B-Instruct-2507-GGUF:Q4_K_M"}
	}
}
