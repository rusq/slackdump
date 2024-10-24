package main

import (
	"log"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/bubbles/btime"
)

const logname = "time.log"

func main() {
	logf, err := os.Create(logname)
	if err != nil {
		log.Fatal(err)
	}
	defer logf.Close()
	log.SetOutput(logf)

	m := btime.New(time.Now())
	m.Focused = true
	m.ShowHelp = true
	p, err := tea.NewProgram(timeModel{m}).Run()
	if err != nil {
		log.Fatal(err)
	}
	_ = p
}

type timeModel struct {
	*btime.Model
}

// Update wraps the Update method of the embedded btime.Model, to satisfy
// the tea.Model interface.
func (m timeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	mod, cmd := m.Model.Update(msg)
	m.Model = mod
	return m, cmd
}
