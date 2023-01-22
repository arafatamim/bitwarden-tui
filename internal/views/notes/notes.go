package notes

type Model struct {
}

func (m Model) View() string {
	return "notes view"
}

func New() Model {
	return Model{}
}
