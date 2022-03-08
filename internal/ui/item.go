package item

import (
	bw "bitwarden-tui/internal"
	"net/url"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	marginLeft = lipgloss.NewStyle().MarginLeft(2)
)

type statusTimeoutMsg struct{}

type Styles struct {
	Title            lipgloss.Style
	Subtitle         lipgloss.Style
	Label            lipgloss.Style
	SelectedProperty lipgloss.Style
}

type ItemKeyMap struct {
	Up            key.Binding
	Down          key.Binding
	Back          key.Binding
	Copy          key.Binding
	Quit          key.Binding
	OpenFullHelp  key.Binding
	CloseFullHelp key.Binding
}

func newItemKeyMap() *ItemKeyMap {
	return &ItemKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "move up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "move down"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "go back"),
		),
		Copy: key.NewBinding(
			key.WithKeys("enter", "c"),
			key.WithHelp("enter/c", "copy property"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "quit"),
		),
		OpenFullHelp: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "more"),
		),
		CloseFullHelp: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "close"),
		),
	}
}

func (k ItemKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Up, k.Down, k.Copy, k.OpenFullHelp,
	}
}
func (k ItemKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Back},
		{k.Copy},
		{k.CloseFullHelp, k.Quit},
	}
}

type SelectedProperty int

const (
	USERNAME SelectedProperty = iota
	PASSWORD
	FIELDS
	URI
)

type Model struct {
	Item   bw.Item
	Help   help.Model
	KeyMap *ItemKeyMap
	Styles Styles

	cursor             SelectedProperty
	height             int
	width              int
	selectedUriIndex   uint8
	selectedFieldIndex uint8

	statusMessage      string
	statusMessageTimer *time.Timer
}

func (m *Model) SetSize(width, height int) {
	m.setSize(width, height)
}

func (m *Model) SetWidth(v int) {
	m.setSize(v, m.height)
}

func (m *Model) SetHeight(v int) {
	m.setSize(m.width, v)
}

func (m *Model) setSize(width, height int) {
	m.width = width
	m.height = height
	m.Help.Width = width
}

func (m *Model) Cursor() SelectedProperty {
	return m.cursor
}

func (m *Model) CursorDown() {
	uriLen := len(m.Item.Login.Uris)
	fieldLen := len(m.Item.Fields)
	switch m.cursor {
	case USERNAME:
		m.cursor = PASSWORD
	case PASSWORD:
		if fieldLen > 0 {
			m.cursor = FIELDS
		} else if uriLen > 0 {
			m.cursor = URI
		} else {
			m.cursor = USERNAME
		}
		m.selectedUriIndex = 0
		m.selectedFieldIndex = 0
	case FIELDS:
		if m.selectedFieldIndex < uint8(fieldLen)-1 {
			m.selectedFieldIndex += 1
		} else {
			m.cursor = URI
		}
	case URI:
		if m.selectedUriIndex < uint8(uriLen)-1 {
			m.selectedUriIndex += 1
		} else {
			m.cursor = USERNAME
		}
	}
}

func (m *Model) CursorUp() {
	uriLen := len(m.Item.Login.Uris)
	fieldLen := len(m.Item.Fields)
	switch m.cursor {
	case USERNAME:
		if uriLen > 0 {
			m.cursor = URI
		} else {
			m.cursor = PASSWORD
		}
		m.selectedUriIndex = uint8(uriLen) - 1
		m.selectedFieldIndex = uint8(fieldLen) - 1
	case PASSWORD:
		m.cursor = USERNAME
	case FIELDS:
		if m.selectedFieldIndex == 0 {
			m.cursor = PASSWORD
		} else {
			m.selectedFieldIndex--
		}
	case URI:
		if m.selectedUriIndex == 0 {
			if fieldLen > 0 {
				m.cursor = FIELDS
			} else {
				m.cursor = PASSWORD
			}
		} else {
			m.selectedUriIndex--
		}
	}
}

func (m *Model) NewStatusMessage(s string) tea.Cmd {
	m.statusMessage = s
	if m.statusMessageTimer != nil {
		m.statusMessageTimer.Stop()
	}
	m.statusMessageTimer = time.NewTimer(1 * time.Second)
	return func() tea.Msg {
		<-m.statusMessageTimer.C
		return statusTimeoutMsg{}
	}
}

func (m *Model) hideStatusMessage() {
	m.statusMessage = ""
	if m.statusMessageTimer != nil {
		m.statusMessageTimer.Stop()
	}
}

func (m *Model) copySelected() tea.Cmd {
	if clipboard.Unsupported {
		return m.NewStatusMessage("clipboard unsupported!")
	}
	var toCopy string
	var prop string
	switch m.cursor {
	case USERNAME:
		toCopy = m.Item.Login.Username
		prop = "username"
	case PASSWORD:
		toCopy = m.Item.Login.Password
		prop = "password"
	case FIELDS:
		toCopy = m.Item.Fields[m.selectedFieldIndex].Value
		prop = "field"
	case URI:
		toCopy = m.Item.Login.Uris[m.selectedUriIndex].Uri
		prop = "url"
	}
	err := clipboard.WriteAll(toCopy)
	if err != nil {
		return m.NewStatusMessage("failed to copy!")
	}
	return m.NewStatusMessage("copied " + prop)
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case statusTimeoutMsg:
		m.hideStatusMessage()
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.KeyMap.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.KeyMap.Back):
			m.cursor = USERNAME
		case key.Matches(msg, m.KeyMap.Down):
			m.CursorDown()
		case key.Matches(msg, m.KeyMap.Up):
			m.CursorUp()
		case key.Matches(msg, m.KeyMap.OpenFullHelp), key.Matches(msg, m.KeyMap.CloseFullHelp):
			m.Help.ShowAll = !m.Help.ShowAll
		case key.Matches(msg, m.KeyMap.Copy):
			cmds = append(cmds, m.copySelected())
		}
	}
	return m, tea.Batch(cmds...)
}

func (m *Model) renderCreds() string {
	userLabel := m.Styles.Label.Render("Username")
	passwordLabel := m.Styles.Label.Render("Password")

	var b strings.Builder
	usernameValue := m.Item.Login.Username
	if usernameValue == "" {
		usernameValue = "(no username)"
	}
	if m.cursor == USERNAME {
		b.WriteString(m.Styles.SelectedProperty.Render("🢒 ") + userLabel)
	} else {
		b.WriteString("  " + userLabel)
	}
	if m.cursor == USERNAME {
		b.WriteString(m.Styles.SelectedProperty.Render(usernameValue))
	} else {
		b.WriteString(usernameValue)
	}

	passwordValue := m.Item.Login.Password
	if passwordValue == "" {
		passwordValue = "(no password)"
	}
	b.WriteString("\n")
	if m.cursor == PASSWORD {
		b.WriteString(m.Styles.SelectedProperty.Render("🢒 ") + passwordLabel)
	} else {
		b.WriteString("  " + passwordLabel)
	}
	if m.cursor == PASSWORD {
		b.WriteString(m.Styles.SelectedProperty.Render(m.Item.Login.Password))
	} else {
		b.WriteString(strings.Repeat("•", 4))
	}

	return b.String()
}

func (m *Model) renderFields() string {
	var b strings.Builder

	maxTitleChars := getMax(bw.Map(m.Item.Fields, func(f bw.Field) string { return f.Name }))
	for i, f := range m.Item.Fields {
		titleChars := len(f.Name)
		remainingChars := maxTitleChars - titleChars
		isSelected := m.selectedFieldIndex == uint8(i) && m.cursor == FIELDS

		val := f.Value
		if f.Value == "" {
			val = "(empty)"
		}

		b.WriteString("\n")
		if isSelected {
			b.WriteString(m.Styles.SelectedProperty.Render("🢒 ") + m.Styles.Label.Render(f.Name))
		} else {
			b.WriteString("  " + m.Styles.Label.Render(f.Name))
		}

		b.WriteString(strings.Repeat(" ", remainingChars))
		if isSelected {
			b.WriteString(m.Styles.SelectedProperty.Render(val))
		} else {
			if f.Type == 1 && f.Value != "" {
				b.WriteString(strings.Repeat("•", 4))
			} else {
				b.WriteString(val)
			}
		}
	}

	return b.String()
}

func (m *Model) renderUri() string {
	var b strings.Builder
	b.WriteString(m.Styles.Subtitle.Render("URIs") + "\n")
	for i, u := range m.Item.Login.Uris {
		parsed, err := url.Parse(u.Uri)
		if m.cursor == URI && m.selectedUriIndex == uint8(i) {
			b.WriteString(m.Styles.SelectedProperty.Render("🢒 "+u.Uri) + "\n")
		} else {
			b.WriteString("  ")
			if err != nil || parsed.Host == "" {
				b.WriteString(u.Uri)
			} else {
				b.WriteString(parsed.Host)
			}
			b.WriteString("\n")
		}
	}
	return b.String()
}

func (m *Model) renderNotes() string {
	var b strings.Builder
	b.WriteString(m.Styles.Subtitle.Render("Notes") + "\n")
	b.WriteString(marginLeft.Render(m.Item.Notes))
	return b.String()
}

func (m Model) View() string {
	item := m.Item

	// title
	title := m.Styles.Title.Copy().MarginLeft(2).Render(item.Name)
	title += " " + m.statusMessage

	creds := m.renderCreds()

	// help
	helpView := m.Help.View(*newItemKeyMap())

	// gluing it together
	var b strings.Builder
	b.WriteString(title)
	b.WriteString("\n\n" + creds)
	if len(item.Fields) > 0 {
		fields := m.renderFields()
		b.WriteString("\n" + fields)
	}
	if len(item.Login.Uris) > 0 {
		uris := m.renderUri()
		b.WriteString("\n\n" + uris)
	}
	if item.Notes != "" {
		notes := m.renderNotes()
		b.WriteString("\n" + notes)
	}

	remainingHeight := m.height - (lipgloss.Height(b.String()) + lipgloss.Height(helpView) - 1)
	if remainingHeight < 1 {
		remainingHeight = 0
	}
	b.WriteString(strings.Repeat("\n", remainingHeight))

	b.WriteString(marginLeft.Render(helpView))

	return b.String()
}

func New() Model {
	return Model{
		Item:               bw.Item{},
		Help:               help.New(),
		KeyMap:             newItemKeyMap(),
		cursor:             USERNAME,
		selectedUriIndex:   0,
		selectedFieldIndex: 0,
	}
}

func getMax(fields []string) int {
	max := 0
	for _, v := range fields {
		chars := len(v)
		if chars > max {
			max = chars
		}
	}
	return max
}
