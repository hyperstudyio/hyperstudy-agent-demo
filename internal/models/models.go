package models

import "github.com/hyperstudyio/hyperstudy-agent-demo/internal/hw"

type Spec struct {
	Name  string
	HFRef string
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
