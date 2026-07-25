# inspector — Roadmap

A living, intentionally short view of where `inspector` is headed. It reflects
current thinking, not a commitment; dates are targets, not promises. Proposals
and corrections are welcome via an issue or PR.

## Now (in progress)

- **Test coverage → 90%** across the package, favouring behaviour-level tests
  over brittle snapshot assertions.
- **Signed, reproducible releases** — GoReleaser + cosign (Sigstore bundle
  format) producing checksummed, signed artifacts on every tag.
- **OpenSSF Best Practices** passing badge, then silver.

## Next

- Broaden the runtime panels (goroutine/heap deltas over time, alloc-rate
  sparklines) while keeping the zero-config, drop-in embedding contract.
- Configurable capture buffers and redaction hooks so the inspector is safe to
  leave wired into production-facing builds.
- Documented, stable public API surface with examples for each tab.

## Later

- Optional export of a debug session (message log + runtime samples) to a file
  for offline review or bug reports.
- Pluggable custom tabs so host apps can surface their own diagnostics beside
  the built-ins.

## Non-goals

- Being a general APM/telemetry backend — inspector stays an in-process,
  developer-facing overlay for Charm Bubble Tea v2 apps.
- Coupling to any single host application; it must remain embeddable in any
  Bubble Tea v2 program.

See also [CONTRIBUTING.md](../CONTRIBUTING.md) and the [CHANGELOG.md](../CHANGELOG.md).
