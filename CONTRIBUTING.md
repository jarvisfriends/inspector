# Contributing to inspector

Thanks for helping improve inspector. This document covers the practical bar
for changes.

## Requirements

- Go **1.26.5 or newer**.
- `golangci-lint` v2, `gofumpt`, `shellcheck`, and `actionlint` for the full
   local gate.

## Workflow

1. Branch from `main`.
2. Make the change with tests. Bug fixes need regression coverage.
3. Run the full local gate before pushing:

   ```bash
   bash tools/local_verify.sh
   ```

4. Open a PR against `main`. CI must pass on Linux, Windows, and macOS.

## Code conventions

- Charm v2 imports only.
- Runtime I/O belongs in `tea.Cmd` paths, not blocking model updates.
- Keep the inspector host-agnostic: new tabs and messages should work both in
   the standalone demo and in embedded hosts.
- Prefer extending existing provider and tab surfaces over adding parallel
   integration mechanisms.
