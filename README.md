# hyperstudy-agent

`hyperstudy-agent` turns your own hardware — a Mac, a workstation GPU, or a DGX Spark — into a [HyperStudy](https://docs.hyperstudy.io)-compatible LLM inference endpoint for running custom AI agent participants in your experiments. It detects your hardware, launches a properly-configured `llama-server`, proves the endpoint meets HyperStudy's contract, and exposes it publicly so you can paste the URL straight into HyperStudy Settings.

## Install

Pick one:

**A. Install script** (downloads a prebuilt binary into `/usr/local/bin`):

```bash
curl -fsSL https://raw.githubusercontent.com/hyperstudyio/hyperstudy-agent-demo/main/install.sh | bash
```

**B. Download a binary** from the [releases page](https://github.com/hyperstudyio/hyperstudy-agent-demo/releases) — `darwin/arm64`, `linux/amd64`, or `linux/arm64` — untar it, and put `hyperstudy-agent` on your `PATH`.

**C. Build from source** (needs [Go](https://go.dev/dl/) 1.26+):

```bash
git clone https://github.com/hyperstudyio/hyperstudy-agent-demo
cd hyperstudy-agent-demo
go build -o hyperstudy-agent .
sudo mv hyperstudy-agent /usr/local/bin/   # optional — or run ./hyperstudy-agent in place
```

Or as a one-liner (installs to `$(go env GOPATH)/bin` as `hyperstudy-agent-demo`):

```bash
go install github.com/hyperstudyio/hyperstudy-agent-demo@latest
```

**Prerequisite:** the `serve` command also needs `llama-server` (llama.cpp) — `brew install llama.cpp` on macOS, or see [Hardware notes](#hardware-notes) for Linux/GPU/Spark. `tunnel` needs `cloudflared`. Verify your install with `hyperstudy-agent --version`.

## Quickstart

Three commands: `serve` a model, `verify` the endpoint, `tunnel` it to the internet.

### 1. `hyperstudy-agent serve`

```
$ hyperstudy-agent serve
Generated API key (saved to /Users/you/.hyperstudy-agent/config.json):
  hsa_3f9a1c2e...

Hardware: darwin/arm64 ram=36GB vram=0GB spark=false
Model:    Qwen3-14B (unsloth/Qwen3-14B-GGUF:Q4_K_M)

Waiting for llama-server (first run downloads the model — may take a while)...

READY
  baseUrl (LAN):   http://192.168.1.23:8080/v1
  API key:         hsa_3f9a1c2e...
Next:
  hyperstudy-agent verify                     # prove the endpoint meets the contract
  hyperstudy-agent tunnel                     # get a public URL for HyperStudy
```

The model is picked automatically from a ladder based on detected hardware (see [Hardware notes](#hardware-notes)) unless you pass `--model`. The API key is generated once and reused across restarts unless you pass `--regenerate-key`.

### 2. `hyperstudy-agent verify`

```
$ hyperstudy-agent verify
Verifying http://localhost:8080/v1

  [PASS] reachable (/models)          note: /models is served without auth on llama-server; auth is checked next
  [PASS] auth enforced                wrong key rejected with 401 Unauthorized
  [PASS] tool calling                 model called respond({"value":2})
  [PASS] concurrency (8 parallel)     p50=1.2s p95=2.1s

All checks passed — paste the baseUrl and key into HyperStudy Settings → API Keys → Custom Agent Endpoint.
```

`verify` runs four checks against the endpoint contract HyperStudy requires (see [Endpoint contract](#endpoint-contract)). If any check fails, it prints the reason and exits non-zero — see [Troubleshooting](#troubleshooting) for what each failure means.

### 3. `hyperstudy-agent tunnel`

```
$ hyperstudy-agent tunnel
PUBLIC ENDPOINT
  baseUrl: https://tender-lights-glow.trycloudflare.com/v1
Next:
  hyperstudy-agent verify --base-url https://tender-lights-glow.trycloudflare.com/v1
Then paste the baseUrl + your API key into HyperStudy Settings → API Keys.
(Quick tunnels get a NEW hostname each run — re-save the credential if you restart.)
```

`tunnel` wraps a `cloudflared` quick tunnel and requires `cloudflared` on your `PATH` (`brew install cloudflared` on macOS, or a release binary on Linux). For a stable hostname across restarts, see [Tailscale Funnel](#tailscale-funnel-alternative) below.

## Hardware notes

Hardware is auto-detected (`hyperstudy-agent serve` prints the detected profile) and used to pick a model from the ladder: **Qwen3-4B-Instruct-2507** (default/low-RAM) → **Qwen3-14B** (16GB+ RAM) → **GLM-4.7-Flash**, an MoE model (24GB+ VRAM) → **Qwen3.6-35B-A3B**, an MoE model (DGX Spark only).

| Hardware | Detection | Model picked | Setup |
|---|---|---|---|
| Mac (Apple Silicon) | `sysctl hw.memsize` for unified RAM | Qwen3-14B (≥16GB), else Qwen3-4B-Instruct-2507 | `brew install llama.cpp` — Metal backend is automatic, no flags needed |
| RTX 3090 (or any 24GB+ discrete GPU) | `nvidia-smi --query-gpu=memory.total` | GLM-4.7-Flash (MoE) | Download a CUDA-enabled `llama-server` release from [ggml-org/llama.cpp releases](https://github.com/ggml-org/llama.cpp/releases), or build from source with `-DGGML_CUDA=ON` |
| DGX Spark (linux/arm64 + NVIDIA unified memory) | `runtime.GOARCH == arm64` on Linux with `nvidia-smi` present | Qwen3.6-35B-A3B (MoE) | Must build `llama-server` from source — no prebuilt binary targets Spark's Blackwell architecture. See recipe below. |

DGX Spark's VRAM often reports as unified/shared memory rather than a discrete pool, so it is matched first in the ladder regardless of the `nvidia-smi` VRAM reading.

### DGX Spark source build

```bash
git clone https://github.com/ggml-org/llama.cpp
cd llama.cpp
cmake -B build -DGGML_CUDA=ON -DCMAKE_CUDA_ARCHITECTURES=121
cmake --build build --config Release -j
sudo install build/bin/llama-server /usr/local/bin/llama-server
```

`121` is the CUDA compute capability for Spark's GB10 Blackwell GPU — a generic build will not target it correctly.

## Model overrides

`serve` accepts flags to bypass the automatic ladder or tune server behavior:

```bash
# Force a specific model instead of the auto-picked one
hyperstudy-agent serve --model unsloth/Qwen3-14B-GGUF:Q4_K_M

# Any -hf-compatible llama.cpp reference works
hyperstudy-agent serve --model unsloth/GLM-4.7-Flash-GGUF:UD-Q4_K_XL

# Tune concurrency and context (defaults: --parallel 8 --ctx 32768)
hyperstudy-agent serve --model unsloth/Qwen3-4B-Instruct-2507-GGUF:Q4_K_M --parallel 4 --ctx 16384

# Serve on a different port
hyperstudy-agent serve --port 9090

# Rotate the saved API key
hyperstudy-agent serve --regenerate-key
```

### Model presets

`--model` also accepts a short preset name instead of a raw `-hf` ref:

```bash
hyperstudy-agent serve --model qwen3.6-moe
```

| Preset key | Model | Size (approx) | Tool calling |
|---|---|---|---|
| `qwen3.6-moe` | Qwen3.6-35B-A3B (MoE) | ~22GB | Reliable, no reasoning overhead |
| `gemma4-moe` | Gemma 4 26B-A4B (MoE) | ~17GB | Works — see note below |
| `gemma4-4b` | Gemma 4 E4B | ~5GB | Works — see note below |

> **Gemma 4 tool-calling note:** Gemma 4 is a reasoning model — it thinks before calling a tool, so it needs adequate `max_tokens` (agents should allow >=256, ideally 512) or it may exhaust the budget before emitting the call. With that headroom, single-turn tool calling works well (empirically 8/8 clean tool calls at a 512-token budget). There is a known llama.cpp issue ([ggml-org/llama.cpp#25072](https://github.com/ggml-org/llama.cpp/issues/25072)), but it affects only **multi-turn** tool-calling sessions (a second turn after a tool response is fed back) — HyperStudy's agent decisions are single-turn and are not affected. Run `hyperstudy-agent verify` to confirm on your hardware. Qwen3.6 (`qwen3.6-moe`) remains the simplest choice if you want to avoid reasoning-token overhead entirely. Also note `gemma4-4b` is Gemma 4 **E4B** (elastic MatFormer, ~4.5B effective parameters), not a dense 4B model — there is no dense Gemma 4 4B.

Raw `-hf` refs (e.g. `unsloth/GLM-4.7-Flash-GGUF:UD-Q4_K_XL`) still work unchanged, and omitting `--model` keeps the hardware auto-detect ladder as the default (see [Hardware notes](#hardware-notes)).

## Endpoint contract

HyperStudy agents call an OpenAI-compatible `/v1/chat/completions` endpoint **with function calling**. Every agent turn submits its response via a `respond` tool call, not free-text — a model or server that never returns `tool_calls` cannot be used as a HyperStudy custom agent endpoint, regardless of how good its text output is.

- **`/v1/models` is unauthenticated on `llama-server`** — `verify`'s `reachable (/models)` check will pass even with a wrong API key. This is expected: `llama-server` only enforces `Authorization` on completion endpoints, so `verify` checks auth separately by sending a deliberately wrong key to `/chat/completions`.
- **Bare Ollama does not meet this contract**: it doesn't enforce inbound `Authorization` (no auth), silently drops `tool_choice`, serializes requests by default (breaking the concurrency budget below), and truncates context past the model's window without warning. Use `llama-server` (or put a reverse proxy with real auth in front of Ollama) instead.
- **`mlx-lm`** has no built-in auth either — don't expose it directly to the internet.
- HyperStudy's platform timeout budget is **60s (default) / 300s (max)** per request, checked against p50/p95 latency under `N`-parallel load by the `concurrency` check.
- `verify` sends `"model": "default"` in its request body. `llama-server` ignores this field and serves whatever model it loaded, but some servers (e.g. vLLM) reject an unrecognized model name with a 404 — if you're fronting one of those, point `verify` at a request shape carrying your actual served model name.

## Troubleshooting

Each row corresponds to a `verify` failure message (from `internal/verify/verify.go`).

| Failure message (from `verify`) | What it means | Fix |
|---|---|---|
| `endpoint unreachable: ...` | Nothing is listening at `--base-url` | Confirm `hyperstudy-agent serve` is running and the port matches |
| `GET /models returned N — not an OpenAI-compatible baseUrl?` | The URL doesn't speak the OpenAI API shape | Double check `--base-url` includes `/v1` and points at `llama-server`, not some other service |
| `a WRONG key was accepted (N). Your server does not validate Authorization ...` | The server ignores `Authorization` entirely | You're likely running bare Ollama or forgot `--api-key`; use `llama-server --api-key <key>` (this is what `serve` does automatically) |
| `chat/completions returned N` | The completions endpoint errored | Check server logs; often an out-of-memory or malformed model load |
| `endpoint rejected the API key — pass --api-key or re-run serve to see the current key` | The tool-call check got a 401/403 | The API key is wrong, not the model/server — pass the correct `--api-key`, or re-run `hyperstudy-agent serve` to print the saved key |
| `response is not OpenAI-shaped` | The JSON response doesn't match the OpenAI schema | The proxy/server in front of the model isn't OpenAI-compatible |
| `no tool_calls in response — HyperStudy agents REQUIRE function calling ...` | The model ignored the `tools` param and replied with plain text | The model doesn't support tool calling or its chat template is broken; switch to a Qwen3 GGUF from `unsloth`, or update llama.cpp (`brew upgrade llama.cpp`) — tool-call parsing requires a recent build |
| `tool_calls arguments are not valid JSON` | The model emitted malformed JSON in a tool call | Usually a chat-template or quantization issue; try a different quant or model |
| `tool_calls arguments ... do not match the respond schema — missing required field "value"` | The model's tool call omitted a required argument | Same as above — try a different model/quant, or increase `--ctx` if the schema is being truncated |
| `tool_calls arguments ... do not match the respond schema — "value" must be a number, got ...` | The model returned the wrong argument type | Model isn't reliably following the JSON schema; try a larger model in the ladder |
| `request failed under load: ...` | A request failed during the N-parallel concurrency test | Check `--parallel` (`-np`) is high enough for your test's `N`, and that the machine isn't out of memory under load |
| `... exceeds the platform's 300s maximum timeout` | p95 latency under load exceeds HyperStudy's hard timeout | Reduce `--parallel`/concurrency, use a smaller/faster model, or add more compute |
| `... above the 60s default; raise the credential's timeoutMs in HyperStudy Settings` | p95 latency exceeds the default timeout but not the hard max (still a PASS) | Optional: raise `timeoutMs` on the credential in HyperStudy Settings, or use a faster model |

## Tailscale Funnel alternative

`hyperstudy-agent tunnel` uses `cloudflared` quick tunnels, which are free but assign a **new hostname every time you restart** — you'd need to re-save the credential in HyperStudy Settings after every restart.

For a stable hostname, use [Tailscale Funnel](https://tailscale.com/kb/1223/funnel) instead:

```bash
tailscale up
tailscale funnel 8080
```

This publishes `https://<your-machine>.<your-tailnet>.ts.net` — save that hostname once in HyperStudy Settings and it stays valid across restarts (unlike the Cloudflare quick tunnel).

## Privacy note

When this endpoint is used as a HyperStudy custom agent, the **full experiment perception payload** — including other participants' chat messages and experiment state — is sent to whatever machine is running this server. Only run `hyperstudy-agent` on hardware you trust, and treat the generated API key as a credential to your experiment data: anyone with the key and endpoint URL can both use your compute and see everything sent to it.

## Setup in HyperStudy

Once `verify` passes, go to **Settings → API Keys → Custom Agent Endpoint** in HyperStudy and paste in the `baseUrl` and API key printed by `serve` (or `tunnel`, if running remotely).
