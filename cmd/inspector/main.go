// Command inspector runs the debug inspector standalone — the "debug any
// charm based app" demo from tui-base ROADMAP SP-12. Inside a real app the
// inspector rides along as an overlay (tui-base wires it to Ctrl+D); here it
// fills the terminal so every tab can be explored on its own: message log,
// runtime stats, terminal diagnostics, accessibility, and the i/w/e test
// notification keys.
package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/jarvisfriends/inspector"
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
	v.AltScreen = true
	return v
}

func main() {
	ins := inspector.New()
	ins.ToggleVisible()
	if _, err := tea.NewProgram(app{ins: ins}).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
