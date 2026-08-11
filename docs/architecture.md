# Architecture

Inspector is a drop-in debug/inspector overlay for Bubble Tea v2 applications. This document describes the system at the
level the
OpenSSF Baseline asks for: the actors involved, the actions they can take, and every external interface of
the released software.

## Actors

- **Host application** — a Go program that embeds `inspector.Model` behind a feature gate or key binding.
- **Developer at the terminal** — toggles the overlay, browses tabs, and triggers profiling actions.
- **Local tools** — `go tool pprof` and a browser, launched only on explicit request against 127.0.0.1.

## Actions and data flow

The host application drives a Bubble Tea event loop: terminal input arrives as messages, the model updates,
and a new frame is rendered to the terminal. This library sits inside that loop — it does not spawn its own
event sources beyond those documented below, and it holds no global mutable state that outlives the model.

## External interfaces

- The public Go API (`inspector.Model`, `Provider`, accessibility helpers). See the
  [Go reference](https://pkg.go.dev/github.com/jarvisfriends/inspector).
- An optional pprof HTTP endpoint on `127.0.0.1:<ephemeral>` while enabled by the user.
- Files written on request: heap/CPU profiles and exported logs (timestamped names).
- The released demo binary takes no flags that change trust boundaries and opens no non-loopback sockets.

## Security-relevant surfaces

- **pprof server.** The overlay can start Go's pprof HTTP server **bound to 127.0.0.1 only**, and only after an
  explicit keypress by the terminal user. It is never started automatically and stops with the host app.
- **Profile/log export.** Heap, CPU, and log exports write timestamped files to the working directory or the
  system temp directory, only on explicit user action.
- **Disk statistics.** The disks tab reads mount/usage information read-only via platform APIs.
- **Rendered content.** The overlay renders values captured from the host application (messages, env vars);
  these are shown to the same user who owns the process, so no trust boundary is crossed.

See [threat-model.md](threat-model.md) for the corresponding threat analysis and mitigations.
