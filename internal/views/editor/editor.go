package editor

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	bw "bitwarden-tui/internal/backend"
)

type Styles struct {
	Title lipgloss.Style
}

type Model struct {
	// components
	Help   help.Model
	KeyMap *KeyMap
	Inputs []textinput.Model

	// data
	Item       bw.Item
	FocusIndex int
	Styles     Styles
}

type KeyMap struct {
	Quit          key.Binding
	Back          key.Binding
	NextField     key.Binding
	PreviousField key.Binding
}

func newKeyMap() *KeyMap {
	return &KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c", "ctrl+d"),
			key.WithHelp("q", "quit"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "go back"),
		),
		NextField: key.NewBinding(
			key.WithKeys("down", "tab"),
			key.WithHelp("tab", "next field"),
		),
		PreviousField: key.NewBinding(
			key.WithKeys("up", "shift+tab"),
			key.WithHelp("shift+tab", "previous field"),
		),
	}
}

func (m *Model) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.Inputs))

	for i := 0; i < len(m.Inputs); i++ {
		m.Inputs[i], cmds[i] = m.Inputs[i].Update(msg)
	}

	return tea.Batch(cmds...)
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.KeyMap.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.KeyMap.NextField), key.Matches(msg, m.KeyMap.PreviousField):
			if key.Matches(msg, m.KeyMap.PreviousField) {
				m.FocusIndex--
			} else {
				m.FocusIndex++
			}

			if m.FocusIndex >= len(m.Inputs) {
				m.FocusIndex = 0
			} else if m.FocusIndex < 0 {
				m.FocusIndex = len(m.Inputs) - 1
			}

			navCmds := make([]tea.Cmd, len(m.Inputs))
			for i := 0; i < len(m.Inputs); i++ {
				if i == m.FocusIndex {
					// Set focused state
					navCmds[i] = m.Inputs[i].Focus()
					continue
				}
				// Remove focused state
				m.Inputs[i].Blur()
			}

			cmds = append(cmds, tea.Batch(navCmds...))
		}
	}
	cmd := m.updateInputs(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	out := "  "

	out += m.Styles.Title.Render("CREATE")

	out += "\n\n"

	for _, inp := range m.Inputs {
		out += inp.View() + "\n"
	}
	return out
}

func New() Model {
	m := Model{
		Help:   help.New(),
		KeyMap: newKeyMap(),
		Inputs: make([]textinput.Model, 4),
	}

	var t textinput.Model
	for i := 0; i < len(m.Inputs); i++ {
		t = textinput.New()

		switch i {
		case 0:
			t.Placeholder = "Name"
			t.Focus()
		case 1:
			t.Placeholder = "Username"
		case 2:
			t.Placeholder = "Password"
		}

		m.Inputs[i] = t
	}

	return m
}
