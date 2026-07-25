# Getting started with inspector

## Install

```bash
go get github.com/jarvisfriends/inspector
```

Requires Go 1.26+ and [Bubble Tea v2](https://github.com/charmbracelet/bubbletea)
(`charm.land/bubbletea/v2`) — inspector is a plain `tea.Model`, so it embeds
into any Bubble Tea v2 app the same way any other component does.

## Wire it into your app

```go
import "github.com/jarvisfriends/inspector"

insp := inspector.New()
// forward key/mouse/resize messages to insp like any other tea.Model,
// and render its View() as an overlay on top of your own UI.
```

Inspector is host-agnostic:

- **Theme**: implement `styles.ColorAware` (snap's shared style contract) and
  hand inspector the live palette pointer so it matches your app's theme.
- **Theme switching**: the accessibility tab emits `inspector.ApplyThemeMsg{ID}`
  — translate that into your own theme plumbing.
- **Extra tabs**: register app-specific `Provider`s (see `provider.go`) to
  surface your own metrics alongside the built-in tabs.

## Try it without writing code

```bash
go run ./cmd/inspector
```

Fills the terminal with the inspector itself — no host app required. Or
download a prebuilt, signed binary for your OS from the
[Releases page](https://github.com/jarvisfriends/inspector/releases).

## What's inside

Live message log with deduplication, runtime log streaming, Go runtime stats
(GC, goroutines, memory), terminal diagnostics, link-rate metrics via
pluggable providers, feature gates, and an accessibility panel that previews
every registered tint against CVD-simulated contrast.

## Where to go next

- [README.md](../README.md) — full feature tour and demo GIF.
- [CONTRIBUTING.md](../CONTRIBUTING.md) — development setup, test/lint bar.
- [CHANGELOG.md](../CHANGELOG.md) — release history.
- [pkg.go.dev](https://pkg.go.dev/github.com/jarvisfriends/inspector) — API reference.
