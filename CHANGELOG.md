# Changelog

All notable changes to the `hyperstudy-agent` CLI are documented in this file.

## [Unreleased]

### Changed
- Repository renamed from `hyperstudyio/hyperstudy-agent-demo` to `hyperstudyio/hyperstudy-agent`. The Go module path, install script, and documentation links now use the new name. GitHub redirects the old URLs, so existing installs and clones keep working.

### Added
- Release notes are now synced automatically to [docs.hyperstudy.io](https://docs.hyperstudy.io/developers/custom-agents) when a release is published.

## [v0.1.3] - 2026-07-15

### Added
- `serve --mtp`: speculative decoding (multi-token prediction) for Gemma 4 models.

### Removed
- `serve --no-thinking` (deprecated): superseded by per-model handling.

## [v0.1.2] - 2026-07-14

### Added
- `serve --no-thinking`: disable reasoning-model thinking for faster turns.

## [v0.1.1] - 2026-07-14

### Fixed
- `serve` output clarifies that the LAN URL is local-network-only and steers users to `tunnel` for hosted HyperStudy.

## [v0.1.0] - 2026-07-14

Initial release.

### Added
- `serve`: launches `llama-server` with hardware-appropriate settings — hardware detection (RAM, NVIDIA VRAM, DGX Spark), automatic model-tier selection, named model presets (`qwen3.6-moe`, `gemma4-moe`, `gemma4-4b`), API-key generation and persistence (`~/.hyperstudy-agent/config.json`), readiness polling.
- `verify`: contract verification against any endpoint — reachability (`/models`), auth enforcement, tool calling via the `respond` tool (arguments validated against the schema), and concurrency latency checks (p50/p95).
- `tunnel`: public HTTPS URL via a Cloudflare quick tunnel (`cloudflared`).
- Install script, goreleaser release pipeline (darwin/arm64, linux/amd64, linux/arm64), CI workflows, MIT license.

[Unreleased]: https://github.com/hyperstudyio/hyperstudy-agent/compare/v0.1.3...HEAD
[v0.1.3]: https://github.com/hyperstudyio/hyperstudy-agent/compare/v0.1.2...v0.1.3
[v0.1.2]: https://github.com/hyperstudyio/hyperstudy-agent/compare/v0.1.1...v0.1.2
[v0.1.1]: https://github.com/hyperstudyio/hyperstudy-agent/compare/v0.1.0...v0.1.1
[v0.1.0]: https://github.com/hyperstudyio/hyperstudy-agent/releases/tag/v0.1.0
