package cfg

import (
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

var (
	Theme = huh.ThemeCharm() // Theme is the default Wizard theme.
)

var (
	// https://gist.github.com/JBlond/2fea43a3049b38287e5e9cefc87b2124
	cDarkgray  = lipgloss.Color("237")
	cPurewhite = lipgloss.Color("255")
)

type Style struct {
	FocusedBorder lipgloss.Style
	BlurredBorder lipgloss.Style
	ShadedCursor  lipgloss.Style
}

var WizStyle = Style{
	FocusedBorder: lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(Theme.Focused.Title.GetForeground()),
	BlurredBorder: lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(Theme.Blurred.Description.GetForeground()),
	ShadedCursor:  lipgloss.NewStyle().Background(cDarkgray).Foreground(cPurewhite),
}
