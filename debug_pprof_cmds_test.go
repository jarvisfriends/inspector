// Copyright (c) 2026 Jarvis Friends contributors
// SPDX-License-Identifier: MIT

package inspector

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/jarvisfriends/snap/notifications"
)

// runCmd executes a tea.Cmd and returns its message, failing the test on a
// nil cmd.
func runCmd(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()
	if cmd == nil {
		t.Fatal("expected a non-nil cmd")
	}
	return cmd()
}

// goBin is the go command's absolute path and origPath the PATH it was found
// on, both captured at package init. The tests below replace PATH to stub the
// toolchain, so resolving either one later would fail.
var (
	goBin    = lookGoBin()
	origPath = os.Getenv("PATH")
)

func lookGoBin() string {
	if p, err := exec.LookPath("go"); err == nil {
		return p
	}
	return "go"
}

// stubExe builds a no-op executable once per run. The stub has to be a real
// binary rather than a shell script: Windows resolves PATH lookups through
// PATHEXT (so an extensionless `go` is invisible) and CreateProcess cannot
// execute a `#!/bin/sh` file at all. The launchers under test only call
// Start(), so a program that exits immediately is enough.
var stubExe = sync.OnceValues(func() ([]byte, error) {
	src, err := os.MkdirTemp("", "inspector-stub")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(src) }()

	for name, body := range map[string]string{
		"main.go": "package main\n\nfunc main() {}\n",
		"go.mod":  "module stub\n\ngo 1.26\n",
	} {
		if err := os.WriteFile(filepath.Join(src, name), []byte(body), 0o600); err != nil {
			return nil, err
		}
	}

	out := filepath.Join(src, "stub"+exeExt())
	build := exec.Command(goBin, "build", "-o", out, ".")
	build.Dir = src
	build.Env = append(os.Environ(), "PATH="+origPath, "GOWORK=off", "CGO_ENABLED=0")
	if combined, err := build.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("build stub tool: %w\n%s", err, combined)
	}
	return os.ReadFile(filepath.Clean(out))
})

func exeExt() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

// setTempDir redirects os.TempDir to dir on every OS: Unix reads TMPDIR,
// Windows reads TMP and then TEMP.
func setTempDir(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("TMPDIR", dir)
	t.Setenv("TMP", dir)
	t.Setenv("TEMP", dir)
}

// stubToolDir drops a no-op executable named after each tool into a temp dir
// and returns the dir, suitable as a PATH override for the exec-based cmds.
func stubToolDir(t *testing.T, names ...string) string {
	t.Helper()
	payload, err := stubExe()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for _, name := range names {
		path := filepath.Join(dir, name+exeExt())
		if err := os.WriteFile(path, payload, 0o700); err != nil { //nolint:gosec // test stub must be executable
			t.Fatal(err)
		}
	}
	return dir
}

// TestPprofServerStartStopRoundTrip starts the pprof HTTP server on an
// ephemeral loopback port, routes the lifecycle msgs through Update, and shuts
// it down again.
func TestPprofServerStartStopRoundTrip(t *testing.T) {
	t.Parallel()

	m := New()
	m.pprof.Addr = "127.0.0.1:0"

	msg := runCmd(t, m.startPprofServerCmd())
	started, ok := msg.(pprofServerStartedMsg)
	if !ok {
		t.Fatalf("start cmd produced %T; want pprofServerStartedMsg", msg)
	}
	if started.Err != nil || started.Server == nil {
		t.Fatalf("start failed: err=%v server=%v", started.Err, started.Server)
	}
	if !strings.HasPrefix(started.URL, "http://127.0.0.1:") {
		t.Fatalf("server URL = %q; want loopback http URL", started.URL)
	}

	_, _ = m.Update(started)
	if m.pprof.server == nil || m.pprof.ServerURL != started.URL {
		t.Fatal("Update(pprofServerStartedMsg) did not record the running server")
	}
	if !strings.Contains(m.settingsMessage, "running") {
		t.Fatalf("settingsMessage = %q; want running notice", m.settingsMessage)
	}

	msg = runCmd(t, m.stopPprofServerCmd())
	stopped, ok := msg.(pprofServerStoppedMsg)
	if !ok {
		t.Fatalf("stop cmd produced %T; want pprofServerStoppedMsg", msg)
	}
	if stopped.Err != nil {
		t.Fatalf("stop failed: %v", stopped.Err)
	}
	_, _ = m.Update(stopped)
	if m.pprof.server != nil || m.pprof.ServerURL != "" {
		t.Fatal("Update(pprofServerStoppedMsg) did not clear the server state")
	}
	if !strings.Contains(m.settingsMessage, "stopped") {
		t.Fatalf("settingsMessage = %q; want stopped notice", m.settingsMessage)
	}
}

// TestPprofServerStartFailure covers the listen-error path and its Update
// handling.
func TestPprofServerStartFailure(t *testing.T) {
	t.Parallel()

	m := New()
	m.pprof.Addr = "127.0.0.1:-1" // invalid port: Listen must fail

	msg := runCmd(t, m.startPprofServerCmd())
	started, ok := msg.(pprofServerStartedMsg)
	if !ok {
		t.Fatalf("start cmd produced %T; want pprofServerStartedMsg", msg)
	}
	if started.Err == nil {
		t.Fatal("expected a listen error for an invalid port")
	}
	_, _ = m.Update(started)
	if !strings.Contains(m.settingsMessage, "start failed") {
		t.Fatalf("settingsMessage = %q; want start-failed notice", m.settingsMessage)
	}
}

// TestUpdatePprofMsgBranches covers the remaining pprof lifecycle msg branches
// in Update: stop failure and the three pprofActionMsg outcomes.
func TestUpdatePprofMsgBranches(t *testing.T) {
	t.Parallel()

	m := New()

	_, _ = m.Update(pprofServerStoppedMsg{Err: errors.New("boom")})
	if !strings.Contains(m.settingsMessage, "stop failed") {
		t.Fatalf("settingsMessage = %q; want stop-failed notice", m.settingsMessage)
	}

	_, _ = m.Update(pprofActionMsg{Kind: pprofKindSnapshot, Err: errors.New("disk full")})
	if !strings.Contains(m.settingsMessage, "failed") {
		t.Fatalf("settingsMessage = %q; want action failure", m.settingsMessage)
	}

	_, _ = m.Update(pprofActionMsg{Kind: pprofKindSnapshot, Text: "saved!", Path: "/tmp/x.pprof"})
	if m.settingsMessage != "saved!" {
		t.Fatalf("settingsMessage = %q; want the action text", m.settingsMessage)
	}
	if m.pprof.LastProfilePath != "/tmp/x.pprof" {
		t.Fatalf("LastProfilePath = %q; want recorded path", m.pprof.LastProfilePath)
	}

	_, _ = m.Update(pprofActionMsg{Kind: pprofKindGoTool})
	if !strings.Contains(m.settingsMessage, "complete") {
		t.Fatalf("settingsMessage = %q; want completion notice", m.settingsMessage)
	}
}

// TestWriteProfileSnapshotCmd covers heap snapshot success and the mkdir
// failure path.
func TestWriteProfileSnapshotCmd(t *testing.T) {
	t.Parallel()

	m := New()
	m.pprof.OutputDir = filepath.Join(t.TempDir(), "pprof-out")

	msg := runCmd(t, m.writeProfileSnapshotCmd())
	action, ok := msg.(pprofActionMsg)
	if !ok {
		t.Fatalf("snapshot cmd produced %T; want pprofActionMsg", msg)
	}
	if action.Err != nil {
		t.Fatalf("snapshot failed: %v", action.Err)
	}
	if action.Path == "" || !strings.HasSuffix(action.Path, ".pprof") {
		t.Fatalf("snapshot path = %q; want a .pprof file", action.Path)
	}
	if fi, err := os.Stat(action.Path); err != nil || fi.Size() == 0 {
		t.Fatalf("snapshot file missing or empty: fi=%v err=%v", fi, err)
	}

	// MkdirAll failure: a regular file where the directory should be.
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	m.pprof.OutputDir = filepath.Join(blocker, "sub")
	msg = runCmd(t, m.writeProfileSnapshotCmd())
	if action, ok = msg.(pprofActionMsg); !ok || action.Err == nil {
		t.Fatalf("snapshot into a file path should fail; got %+v", msg)
	}
}

// TestCaptureCPUProfileCmd covers the CPU capture success path (zero-second
// capture keeps it instant) and the already-profiling failure path.
func TestCaptureCPUProfileCmd(t *testing.T) {
	t.Parallel()

	m := New()
	m.pprof.OutputDir = filepath.Join(t.TempDir(), "cpu-out")
	m.pprof.CPUCaptureSecs = 0 // sleep(0): deterministic and instant

	msg := runCmd(t, m.captureCPUProfileCmd())
	action, ok := msg.(pprofActionMsg)
	if !ok {
		t.Fatalf("capture cmd produced %T; want pprofActionMsg", msg)
	}
	if action.Err != nil {
		t.Fatalf("capture failed: %v", action.Err)
	}
	if _, err := os.Stat(action.Path); err != nil {
		t.Fatalf("cpu profile file missing: %v", err)
	}

	// StartCPUProfile fails while a profile is already running.
	if err := pprof.StartCPUProfile(io.Discard); err != nil {
		t.Fatalf("could not start blocking profile: %v", err)
	}
	msg = runCmd(t, m.captureCPUProfileCmd())
	pprof.StopCPUProfile()
	if action, ok = msg.(pprofActionMsg); !ok || action.Err == nil {
		t.Fatalf("capture during active profiling should fail; got %+v", msg)
	}

	// MkdirAll failure mirrors the snapshot cmd.
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	m.pprof.OutputDir = filepath.Join(blocker, "sub")
	msg = runCmd(t, m.captureCPUProfileCmd())
	if action, ok = msg.(pprofActionMsg); !ok || action.Err == nil {
		t.Fatalf("capture into a file path should fail; got %+v", msg)
	}
}

// TestOpenGoToolPprofCmds drives the three "go tool pprof -http" launchers
// through their precondition-error, exec-error, and success paths using a
// stubbed toolchain on PATH.
func TestOpenGoToolPprofCmds(t *testing.T) {
	m := New()

	// Precondition errors: nothing saved yet / server not running.
	for name, cmd := range map[string]tea.Cmd{
		"latest without file": m.openGoToolPprofLatestCmd(),
		"live heap no server": m.openGoToolPprofLiveHeapCmd(),
		"live cpu no server":  m.openGoToolPprofLiveCPUCmd(),
	} {
		msg := runCmd(t, cmd)
		action, ok := msg.(pprofActionMsg)
		if !ok || action.Err == nil {
			t.Errorf("%s: want precondition error; got %+v", name, msg)
		}
	}

	m.pprof.LastProfilePath = filepath.Join(t.TempDir(), "heap.pprof")
	m.pprof.ServerURL = testServerURL

	// Exec failure: empty PATH means the go toolchain cannot be found.
	t.Setenv("PATH", t.TempDir())
	for name, cmd := range map[string]tea.Cmd{
		"latest exec fail":    m.openGoToolPprofLatestCmd(),
		"live heap exec fail": m.openGoToolPprofLiveHeapCmd(),
		"live cpu exec fail":  m.openGoToolPprofLiveCPUCmd(),
	} {
		msg := runCmd(t, cmd)
		action, ok := msg.(pprofActionMsg)
		if !ok || action.Err == nil {
			t.Errorf("%s: want exec error; got %+v", name, msg)
		}
	}

	// Success: a stub `go` on PATH starts and exits immediately.
	t.Setenv("PATH", stubToolDir(t, "go"))
	for name, cmd := range map[string]tea.Cmd{
		"latest ok":    m.openGoToolPprofLatestCmd(),
		"live heap ok": m.openGoToolPprofLiveHeapCmd(),
		"live cpu ok":  m.openGoToolPprofLiveCPUCmd(),
	} {
		msg := runCmd(t, cmd)
		action, ok := msg.(pprofActionMsg)
		if !ok || action.Err != nil {
			t.Errorf("%s: want success; got %+v", name, msg)
			continue
		}
		if !strings.Contains(action.Text, "http://"+m.pprof.ToolUIAddr) {
			t.Errorf("%s: text %q missing UI address", name, action.Text)
		}
	}
}

// TestOpenBrowserCmd covers the browser launcher's success (stubbed xdg-open)
// and failure (empty PATH) paths.
func TestOpenBrowserCmd(t *testing.T) {
	t.Setenv("PATH", stubToolDir(t, "xdg-open", "open", "rundll32"))
	msg := runCmd(t, openBrowserCmd("http://127.0.0.1:1/"))
	action, ok := msg.(pprofActionMsg)
	if !ok || action.Err != nil {
		t.Fatalf("open browser with stub: want success; got %+v", msg)
	}
	if !strings.Contains(action.Text, "http://127.0.0.1:1/") {
		t.Fatalf("open browser text = %q; want URL echoed", action.Text)
	}

	t.Setenv("PATH", t.TempDir())
	msg = runCmd(t, openBrowserCmd("http://127.0.0.1:1/"))
	if action, ok = msg.(pprofActionMsg); !ok || action.Err == nil {
		t.Fatalf("open browser without opener should fail; got %+v", msg)
	}
}

// TestExportLogCmd covers the log export success path (file written with all
// entries) and the unwritable-directory failure path.
func TestExportLogCmd(t *testing.T) {
	tmp := t.TempDir()
	setTempDir(t, tmp)

	m := New()
	m.LogMessageForDebugging("export-me-1")
	m.LogMessageForDebugging("export-me-2")

	msg := runCmd(t, m.exportLogCmd())
	note, ok := msg.(notifications.AddMsg)
	if !ok {
		t.Fatalf("export produced %T; want notifications.AddMsg", msg)
	}
	if note.Severity != notifications.SeverityInfo {
		t.Fatalf("export severity = %v; want info", note.Severity)
	}
	logPath := strings.TrimPrefix(note.Content, "Inspector log exported to ")
	data, err := os.ReadFile(logPath) //nolint:gosec // path produced by the cmd under test
	if err != nil {
		t.Fatalf("exported log unreadable: %v", err)
	}
	for _, want := range []string{"export-me-1", "export-me-2"} {
		if !strings.Contains(string(data), want) {
			t.Errorf("exported log missing %q", want)
		}
	}

	// Failure: TMPDIR is a regular file, so MkdirAll cannot create the dir.
	blocker := filepath.Join(tmp, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	setTempDir(t, blocker)
	msg = runCmd(t, m.exportLogCmd())
	if note, ok = msg.(notifications.AddMsg); !ok || note.Severity != notifications.SeverityError {
		t.Fatalf("export into a file path should fail with an error notification; got %+v", msg)
	}
}
