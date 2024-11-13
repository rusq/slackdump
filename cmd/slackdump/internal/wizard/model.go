package wizard

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/bootstrap"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui"
)

type model struct {
	form     *huh.Form
	val      string
	finished bool
}

const kSelection = "selection" // selection key

func newModel(m *menu) model {
	var options []huh.Option[string]
	for i, name := range m.names {
		var text = fmt.Sprintf("%-10s - %s", name, m.items[i].Description)
		if m.items[i].Description == "" {
			text = fmt.Sprintf("%-10s", name)
		}
		options = append(options, huh.NewOption(text, name))
	}
	return model{
		form: huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Key(kSelection).
					Title(m.title).
					Description("Slack workspace:  " + bootstrap.CurrentWsp()).
					Options(options...),
			),
		).WithTheme(ui.HuhTheme()),
	}
}

func (m *model) Init() tea.Cmd {
	return m.form.Init()
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+c", "q":
			m.finished = true
			return m, tea.Quit
		}
	}

	var cmds []tea.Cmd
	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
		cmds = append(cmds, cmd)
	}

	if m.form.State == huh.StateCompleted {
		m.val = m.form.GetString(kSelection)
		cmds = append(cmds, tea.Quit)
	}

	return m, tea.Batch(cmds...)
}

func (m *model) View() string {
	if m.finished {
		return ""
	}
	return m.form.View()
}
