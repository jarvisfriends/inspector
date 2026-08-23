// Command inspector runs the debug inspector standalone. Inside a host app the
// inspector usually rides as an overlay; here it fills the terminal so every
// tab can be explored on its own: message log, runtime stats, terminal
// diagnostics, accessibility, and the i/w/e test notification keys.
package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/jarvisfriends/inspector/inspector"
)

// app hosts the inspector model fullscreen and keeps it visible.
type app struct {
	ins *inspector.InspectorModel
}

func (a app) Init() tea.Cmd { return a.ins.Init() }

func (a app) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "q", "ctrl+c":
			return a, tea.Quit
		}
	}
	model, cmd := a.ins.Update(msg)
	if ins, ok := model.(*inspector.InspectorModel); ok {
		a.ins = ins
	}
	return a, cmd
}

func (a app) View() tea.View {
	v := a.ins.View()
	// Standalone runs never enabled mouse reporting, so the inspector's own
	// OnMouse handler (tab clicks, wheel tab/row scrolling, settings-row
	// activation) was unreachable outside a host app. Cell-motion mode is
	// what tui-base's program runs with.
	v.MouseMode = tea.MouseModeCellMotion
	if inner := v.OnMouse; inner != nil {
		v.OnMouse = func(mm tea.MouseMsg) tea.Cmd {
			// The inspector's hit zones are content-relative: its host
			// subtracts the overlay origin plus one border cell (see
			// tui-base's inspectorOverlay.OverlayMouse). Standalone, the view
			// sits at 0,0, so only the border cell needs removing.
			return inner(shiftMouse(mm, -1, -1))
		}
	}
	v.AltScreen = true
	return v
}

// shiftMouse translates a pointer event so the inspector sees coordinates
// relative to its content area rather than the terminal.
func shiftMouse(mm tea.MouseMsg, dx, dy int) tea.MouseMsg {
	switch e := mm.(type) {
	case tea.MouseClickMsg:
		e.X += dx
		e.Y += dy
		return e
	case tea.MouseReleaseMsg:
		e.X += dx
		e.Y += dy
		return e
	case tea.MouseWheelMsg:
		e.X += dx
		e.Y += dy
		return e
	case tea.MouseMotionMsg:
		e.X += dx
		e.Y += dy
		return e
	}
	return mm
}

func main() {
	ins := inspector.New()
	ins.ToggleVisible()
	if _, err := tea.NewProgram(app{ins: ins}).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
