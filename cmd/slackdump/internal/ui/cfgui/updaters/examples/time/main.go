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

	m := btime.NewTime(time.Now())
	m.Focused = true
	m.ShowHelp = true
	p, err := tea.NewProgram(m).Run()
	if err != nil {
		log.Fatal(err)
	}
	_ = p
}
