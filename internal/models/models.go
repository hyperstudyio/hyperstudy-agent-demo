package models

import (
	"sort"

	"github.com/hyperstudyio/hyperstudy-agent/internal/hw"
)

type Spec struct {
	Name  string
	HFRef string
	// MTPRepo and MTPFile locate the model's co-trained Multi-Token-Prediction
	// draft GGUF (HF repo + in-repo path) for speculative decoding via
	// serve --mtp. Both empty when the model ships no MTP drafter.
	MTPRepo string
	MTPFile string
}

// MTPURL returns the HuggingFace download URL for the model's MTP draft GGUF,
// or "" when the model has no MTP drafter. serve --mtp fetches this once and
// passes the local file to llama-server via -md.
func (s Spec) MTPURL() string {
	if s.MTPRepo == "" || s.MTPFile == "" {
		return ""
	}
	return "https://huggingface.co/" + s.MTPRepo + "/resolve/main/" + s.MTPFile
}

// Presets maps short, memorable names to HF-verified -hf refs. These are
// opt-in overrides for serve --model; the hardware ladder in Pick remains
// the no-flag default. The gemma4 presets carry MTP draft coordinates so
// serve --mtp can enable lossless speculative decoding (~1.4–2.3x faster);
// Qwen3.6 ships no MTP drafter.
var Presets = map[string]Spec{
	"qwen3.6-moe": {Name: "Qwen3.6-35B-A3B (MoE)", HFRef: "unsloth/Qwen3.6-35B-A3B-GGUF:UD-Q4_K_XL"},
	"gemma4-moe": {
		Name: "Gemma 4 26B-A4B (MoE)", HFRef: "unsloth/gemma-4-26B-A4B-it-GGUF:UD-Q4_K_M",
		MTPRepo: "unsloth/gemma-4-26B-A4B-it-GGUF", MTPFile: "MTP/mtp-gemma-4-26B-A4B-it-Q8_0.gguf",
	},
	"gemma4-4b": {
		Name: "Gemma 4 E4B", HFRef: "unsloth/gemma-4-E4B-it-GGUF:Q4_K_M",
		MTPRepo: "unsloth/gemma-4-E4B-it-GGUF", MTPFile: "MTP/mtp-gemma-4-E4B-it-Q8_0.gguf",
	},
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
