package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/paginator"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	l "github.com/charmbracelet/lipgloss"

	bw "bitwarden-tui/internal"
	"bitwarden-tui/internal/ui"
)

const (
	navyBlue     = "#175ddc"
	skyBlue      = "116"
	dimYellow    = "#b36300"
	brightYellow = "#DC9617"
)

var (
	appStyle   = l.NewStyle().Padding(1, 2)
	titleStyle = l.NewStyle().
			Foreground(l.Color("#efefef")).
			Background(l.Color(navyBlue)).
			Padding(0, 1)

	// subtitleStyle = l.NewStyle().Padding(0, 1).MarginLeft(2).Background(l.Color("#444"))
	subtitleStyle = l.NewStyle().MarginLeft(2).Border(l.NormalBorder(), false, false, true, false).BorderBottomForeground(l.Color("#666"))

	statusMessageStyle = l.NewStyle().
				Foreground(l.AdaptiveColor{Light: "14", Dark: "14"})

	itemLabelStyle = l.NewStyle().Foreground(l.Color("#888")).MarginRight(1)

	selectedPropertyStyle = l.NewStyle().Foreground(l.Color(brightYellow))
)

type listItem struct {
	id          string
	title       string
	description string
}

func (i listItem) Id() string          { return i.id }
func (i listItem) Title() string       { return i.title }
func (i listItem) Description() string { return i.description }
func (i listItem) FilterValue() string { return i.title + " " + i.description }

type listKeyMap struct {
	newItem  key.Binding
	openItem key.Binding
	sync     key.Binding
}

func newListKeyMap() *listKeyMap {
	return &listKeyMap{
		newItem: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "add item"),
		),
		openItem: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "open item"),
		),
		sync: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "sync vault"),
		),
	}
}

type view int

const (
	PASSINPUT view = iota
	PASSLIST
	PASSITEM
)

type inputView struct {
	textInput textinput.Model
	spinner   spinner.Model
	isLoading bool
	error     error
}

type itemView struct {
	item item.Model
}

type listView struct {
	list list.Model
	keys *listKeyMap
}

type model struct {
	view      view
	listView  listView
	inputView inputView
	itemView  itemView
	bwContext *bw.Context
}

// == MSG ==

type sessionMsg *bw.Context
type itemMsg bw.Item
type itemsMsg []bw.Item
type errorMsg struct{ err error }

// == CMD ==

func raiseErr(s string) tea.Cmd {
	return func() tea.Msg {
		return errorMsg{errors.New(s)}
	}
}

func (m *model) login() tea.Cmd {
	return func() tea.Msg {
		ctx, err := bw.InitializeClient(m.inputView.textInput.Value())
		if err != nil {
			m.inputView.isLoading = false
			return errorMsg{errors.New("Invalid master password!")}
		}
		m.inputView.isLoading = false
		return sessionMsg(ctx)
	}
}

func (m *model) getItem() tea.Cmd {
	return func() tea.Msg {
		i := m.listView.list.SelectedItem().(listItem)
		item, err := m.bwContext.GetItem(i.Id())
		if err != nil || item == nil {
			return errorMsg{errors.New("Failed to fetch item!")}
		}
		return itemMsg(*item)
	}
}

func (m *model) getItems() tea.Cmd {
	return func() tea.Msg {
		items, err := m.bwContext.GetItems(bw.FilterOptions{})
		if err != nil {
			return errorMsg{errors.New("Failed to fetch items")}
		}
		return itemsMsg(items)
	}
}

func (m *model) sync() tea.Cmd {
	err := m.bwContext.Sync()
	if err != nil {
		return raiseErr("Sync failed!")
	}
	m.listView.list.StopSpinner()
	return m.getItems()
}

func newModel() model {
	var (
		listKeys = newListKeyMap()
	)

	items := []list.Item{}

	listDelegate := list.NewDefaultDelegate()
	listDelegate.Styles.SelectedTitle.Foreground(l.Color(brightYellow))
	listDelegate.Styles.SelectedDesc.Foreground(l.Color(dimYellow))
	listDelegate.Styles.SelectedTitle.BorderForeground(l.Color(brightYellow))
	listDelegate.Styles.SelectedDesc.BorderForeground(l.Color(brightYellow))
	listDelegate.Styles.NormalTitle.Foreground(l.AdaptiveColor{Light: "#222222", Dark: "#efefef"})
	passList := list.New(items, listDelegate, 0, 0)
	passList.Title = "BITWARDEN"
	passList.Styles.Title = titleStyle
	passList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			listKeys.openItem,
			listKeys.newItem,
			listKeys.sync,
		}
	}
	passList.Styles.PaginationStyle.Foreground(l.Color("#666"))
	passList.Paginator.Type = paginator.Arabic
	passList.Paginator.ArabicFormat = "page %d of %d"
	passList.Styles.FilterCursor.Foreground(l.Color(dimYellow))
	passList.SetSpinner(spinner.MiniDot)
	passList.StatusMessageLifetime = time.Duration(3 * time.Second)
	listView := listView{
		list: passList,
		keys: listKeys,
	}

	inputViewInput := textinput.New()
	inputViewInput.EchoMode = textinput.EchoPassword
	inputViewInput.EchoCharacter = '•'
	inputViewInput.Placeholder = "Master password"
	inputViewInput.Prompt = "🢒 "
	inputViewInput.PromptStyle = l.NewStyle().Foreground(l.Color(brightYellow))
	inputViewInput.Focus()
	inputViewSpinner := spinner.New()
	inputViewSpinner.Spinner = spinner.MiniDot
	inputViewSpinner.Style = l.NewStyle().Foreground(l.Color("8"))
	inputView := inputView{
		textInput: inputViewInput,
		spinner:   inputViewSpinner,
	}

	itemView := itemView{}
	itemView.item = item.New()
	itemView.item.Styles = item.Styles{
		Title:            titleStyle,
		Subtitle:         subtitleStyle,
		Label:            itemLabelStyle,
		SelectedProperty: selectedPropertyStyle,
	}

	return model{
		listView:  listView,
		inputView: inputView,
		itemView:  itemView,
		view:      PASSINPUT,
	}
}

func (m model) Init() tea.Cmd {
	return m.inputView.spinner.Tick
}
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		topGap, rightGap, bottomGap, leftGap := appStyle.GetPadding()
		finalW, finalH := msg.Width-leftGap-rightGap, msg.Height-topGap-bottomGap
		m.listView.list.SetSize(finalW, finalH)
		m.itemView.item.SetSize(finalW, finalH)

		m.itemView.item.Help.Width = msg.Width
		m.listView.list.Help.Width = msg.Width
	}

	switch m.view {
	case PASSINPUT:
		{
			switch msg := msg.(type) {
			case tea.KeyMsg:
				m.inputView.error = nil
				switch msg.String() {
				case "ctrl+c", "ctrl+d":
					return m, tea.Quit
				case "enter":
					if m.view == PASSINPUT {
						m.inputView.isLoading = true
						return m, m.login()
					}
				}
			case sessionMsg:
				m.bwContext = msg
				return m, m.getItems()
			case itemsMsg:
				items := listItemsFromBwItems(msg)
				m.view = PASSLIST
				m.inputView.isLoading = false
				listCmd := m.listView.list.SetItems(items)
				return m, listCmd
			case errorMsg:
				m.inputView.textInput.SetValue("")
				m.inputView.error = msg.err
				m.inputView.isLoading = false
				m.listView.list.StopSpinner()
				return m, nil
			}
			var (
				inputCmd, spinnerCmd tea.Cmd
			)
			m.inputView.spinner, spinnerCmd = m.inputView.spinner.Update(msg)
			m.inputView.textInput, inputCmd = m.inputView.textInput.Update(msg)
			return m, tea.Batch(inputCmd, spinnerCmd)
		}
	case PASSLIST:
		{
			switch msg := msg.(type) {
			case tea.KeyMsg:
				if m.listView.list.FilterState() == list.Filtering {
					break
				}
				switch {
				case key.Matches(msg, m.listView.keys.openItem):
					spinnerCmd := m.listView.list.StartSpinner()
					return m, tea.Batch(spinnerCmd, m.getItem())
				case key.Matches(msg, m.listView.keys.newItem):
					cmd := m.listView.list.NewStatusMessage("new item!")
					return m, cmd
				case key.Matches(msg, m.listView.keys.sync):
					spinnerCmd := m.listView.list.StartSpinner()
					statusCmd := m.listView.list.NewStatusMessage("started syncing")
					return m, tea.Batch(spinnerCmd, statusCmd, m.getItems())
				}
			case itemsMsg:
				items := listItemsFromBwItems(msg)
				listCmd := m.listView.list.SetItems(items)
				m.listView.list.StopSpinner()
				return m, listCmd
			case itemMsg:
				m.view = PASSITEM
				m.listView.list.StopSpinner()
				m.itemView.item.Item = bw.Item(msg)
				return m, nil
			case errorMsg:
				statusCmd := m.listView.list.NewStatusMessage(msg.err.Error())
				return m, statusCmd
			}
			var listCmd tea.Cmd
			m.listView.list, listCmd = m.listView.list.Update(msg)
			return m, listCmd
		}
	case PASSITEM:
		{
			switch msg := msg.(type) {
			case tea.KeyMsg:
				switch {
				case key.Matches(msg, m.itemView.item.KeyMap.Back):
					m.view = PASSLIST
				}
			}
			var itemCmd tea.Cmd
			m.itemView.item, itemCmd = m.itemView.item.Update(msg)
			return m, itemCmd
		}
	}

	return m, nil
}

// == VIEW ==

func renderInput(m model) string {
	var b strings.Builder
	if m.inputView.isLoading {
		b.WriteString(m.inputView.spinner.View() + " ")
		titleStyle.MarginLeft(0)
	} else {
		titleStyle.MarginLeft(2)
	}
	b.WriteString(titleStyle.Render("BITWARDEN"))
	b.WriteString("\n\n" + m.inputView.textInput.View())
	if m.inputView.error != nil {
		b.WriteString("\n\n" + l.NewStyle().Foreground(l.Color("9")).Render(m.inputView.error.Error()))
	}
	return appStyle.Render(b.String())
}
func renderList(m model) string {
	out := m.listView.list.View()
	return appStyle.Render(out)
}
func renderItem(m model) string {
	out := m.itemView.item.View()
	return appStyle.Render(out)
}

func (m model) View() string {
	switch m.view {
	case PASSINPUT:
		return renderInput(m)
	case PASSLIST:
		return renderList(m)
	case PASSITEM:
		return renderItem(m)
	}
	return "why am i here?"
}

// == MAIN ==

func main() {
	if err := tea.NewProgram(newModel(), tea.WithAltScreen()).Start(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}

// == UTILS ==

func listItemsFromBwItems(bwItems []bw.Item) []list.Item {
	var items []list.Item
	for _, pass := range bwItems {
		i := listItem{
			id:          pass.Id,
			title:       pass.Name,
			description: pass.Login.Username,
		}
		items = append(items, i)
	}
	return items
}
