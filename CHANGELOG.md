# Changelog

All notable changes to this project are documented in this file. The format
follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the
project adheres to semantic versioning (breaking changes allowed before v1.0).

The full, authoritative release history — including every tagged version — is
on the [GitHub releases page](https://github.com/jarvisfriends/inspector/releases).

## [Unreleased]

### Added

- The Settings tab's "Output dir" row is now editable: Enter (or a click)
  opens snap's folder picker as a modal over the section — arrow/wheel
  navigation, Space selects the highlighted folder, Ctrl+S the browsed one,
  Esc cancels.

### Fixed

- Standalone runs (`inspector` binary) never enabled mouse reporting, so tab
  clicks, wheel navigation, and settings-row clicks only worked inside a host
  app. The root program now runs cell-motion mouse mode and applies the same
  border-offset translation the tui-base host does.
- Settings rows below the selected row activated the wrong row when clicked:
  the selected row's help line shifts every following row down one line, but
  the click hit-test assumed a 1:1 line↔row layout. Clicking "open in
  browser" rows therefore hit the row above (often a no-op). The hit-test now
  maps rendered lines through the real layout.

## [0.0.4]

### Changed

- Bumped `github.com/jarvisfriends/snap` to v0.2.2.
- Continuous-integration hardening: all GitHub Actions pinned to commit SHAs,
  an MIT `LICENSE`, an OpenSSF Scorecard workflow, CodeQL scanning,
  `govulncheck`, dependency review, a native Go fuzz target, the full shared
  golangci-lint suite, and a Codecov coverage upload.

Earlier releases predate this changelog; see the GitHub releases page for their
notes.

[Unreleased]: https://github.com/jarvisfriends/inspector/compare/v0.0.4...HEAD
[0.0.4]: https://github.com/jarvisfriends/inspector/releases/tag/v0.0.4
