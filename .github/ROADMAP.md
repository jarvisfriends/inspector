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

## Layout review (2026-08-22) — queued improvements

A code-level pass over every tab's renderer. Ordered by impact; each item is
independent.

1. **Fixed footer for help + status.** The settings help line renders under
   the selected row (shifting all rows below — the source of the click-mapping
   bug fixed 2026-08-22) and `settingsMessage` renders after the LAST row,
   off-screen whenever the list is scrolled up. Move both to a pinned 1–2 line
   footer under the section viewport: stable geometry (the line↔row hit-test
   collapses back to 1:1), and pprof action feedback becomes always visible.
2. **Reclaim the title rows.** `"<Section> (Inspector)"` + a full-width
   separator spend two lines duplicating what the highlighted tab already
   says. Fold the branding into the tab bar row (right-aligned "Inspector")
   and drop the separator — two extra content lines in an overlay that is
   usually height-starved.
3. **Runtime/Input tabs: key-value sections, not cursor tables.** Both tabs
   render fixed-shape stats through a bubbles table (4 repeated Metric/Value
   column pairs, a row cursor that selects nothing actionable). The Terminal
   tab's `termSection` pattern — titled groups of aligned k/v rows, warn rows
   styled — reads better, drops the repeated headers, and reflows to 1/2/3
   column pairs by width instead of falling back to a separate flat renderer.
   Keep the table only where rows are actually data (Disks).
4. **Settings: collapsible categories.** 33+ fixed rows plus one row per
   feature gate in one flat scroll. Mirror tui-base's SP-9 pattern: `▸ Title
   (n)` headers that toggle on Enter/click, framework sections collapsed by
   default. The pprof block (19 rows, most of them gated on the server being
   enabled) is the immediate winner — collapse it while the server is off.
5. **Log tab density.** Each entry costs 3 lines (header, content, blank).
   Offer a compact single-line mode (time · type · truncated content · ×N),
   dim the timestamp, and drop the blank separator — triples visible history.
   The 'f' WARN+ toggle generalizes to a level cycle (all → info+ → warn+).
6. **Click-target polish.** Tab hit ranges already account for the border
   padding (`debugBorderPaddingX`); section-row hit-tests use raw X. Full-row
   targets make it moot today, but any future inline buttons (e.g. the ↵
   action rows) need the same X normalization the tabs got.

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
