package ui

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/lipgloss"
)

type Theme struct {
	Focused ControlStyle
	Blurred ControlStyle

	StatusText lipgloss.Style
	Error      lipgloss.Style
	Help       help.Styles
}

type ControlStyle struct {
	// Border defines the border style for the control.  It should not have
	// any color, otherwise it will paint everything that does not have
	// the color set.
	Border      lipgloss.Style
	Title       lipgloss.Style
	Description lipgloss.Style

	Text lipgloss.Style

	Cursor       lipgloss.Style
	SelectedLine lipgloss.Style

	Selected   lipgloss.Style
	Unselected lipgloss.Style

	SelectedFile   lipgloss.Style
	UnselectedFile lipgloss.Style
	DisabledFile   lipgloss.Style
	Directory      lipgloss.Style

	Options OptionStyle
}

type OptionStyle struct {
	Section       lipgloss.Style
	Name          lipgloss.Style
	EnabledValue  lipgloss.Style
	DisabledValue lipgloss.Style
	SelectedName  lipgloss.Style
}

func DefaultTheme() Theme {
	black := lipgloss.Color("0")
	red := lipgloss.Color("1")
	green := lipgloss.Color("2")
	blue := lipgloss.Color("4")
	white := lipgloss.Color("7")
	gray := lipgloss.Color("8")
	ltgreen := lipgloss.Color("10")
	ltmagenta := lipgloss.Color("13")

	return Theme{
		Focused: ControlStyle{
			Border:         lipgloss.NewStyle().Border(lipgloss.DoubleBorder()),
			Title:          lipgloss.NewStyle().Foreground(green).Bold(true),
			Description:    lipgloss.NewStyle().Foreground(white),
			Cursor:         lipgloss.NewStyle().Foreground(ltmagenta),
			SelectedLine:   lipgloss.NewStyle().Background(green).Foreground(black),
			Selected:       lipgloss.NewStyle().Foreground(green),
			Unselected:     lipgloss.NewStyle().Foreground(white),
			SelectedFile:   lipgloss.NewStyle().Foreground(green),
			UnselectedFile: lipgloss.NewStyle().Foreground(white),
			DisabledFile:   lipgloss.NewStyle().Foreground(gray),
			Directory:      lipgloss.NewStyle().Foreground(blue),
			Text:           lipgloss.NewStyle().Foreground(white),
			Options: OptionStyle{
				Section:       lipgloss.NewStyle().Foreground(ltgreen).Bold(true),
				Name:          lipgloss.NewStyle().Foreground(green),
				EnabledValue:  lipgloss.NewStyle().Foreground(white),
				DisabledValue: lipgloss.NewStyle().Foreground(green),
				SelectedName:  lipgloss.NewStyle().Foreground(black).Background(green).Underline(true),
			},
		},
		Blurred: ControlStyle{
			Border:         lipgloss.NewStyle().Border(lipgloss.NormalBorder()),
			Title:          lipgloss.NewStyle().Foreground(gray).Bold(true),
			Description:    lipgloss.NewStyle().Foreground(gray),
			Cursor:         lipgloss.NewStyle().Foreground(gray),
			SelectedLine:   lipgloss.NewStyle().Background(gray).Foreground(black),
			Selected:       lipgloss.NewStyle().Foreground(gray),
			Unselected:     lipgloss.NewStyle().Foreground(gray),
			SelectedFile:   lipgloss.NewStyle().Foreground(gray),
			UnselectedFile: lipgloss.NewStyle().Foreground(gray),
			DisabledFile:   lipgloss.NewStyle().Foreground(gray),
			Directory:      lipgloss.NewStyle().Foreground(gray),
			Text:           lipgloss.NewStyle().Foreground(gray),
			Options: OptionStyle{
				Section:       lipgloss.NewStyle().Foreground(gray).Bold(true),
				Name:          lipgloss.NewStyle().Foreground(gray),
				EnabledValue:  lipgloss.NewStyle().Foreground(gray),
				DisabledValue: lipgloss.NewStyle().Foreground(gray),
				SelectedName:  lipgloss.NewStyle().Foreground(gray).Underline(true).UnderlineSpaces(false),
			},
		},
		StatusText: lipgloss.NewStyle().Foreground(green),
		Error:      lipgloss.NewStyle().Foreground(red).Bold(true),
		Help: help.Styles{
			ShortDesc:      lipgloss.NewStyle().Foreground(white),
			FullDesc:       lipgloss.NewStyle().Foreground(white),
			Ellipsis:       lipgloss.NewStyle().Foreground(white),
			ShortKey:       lipgloss.NewStyle().Foreground(green),
			FullKey:        lipgloss.NewStyle().Foreground(green),
			ShortSeparator: lipgloss.NewStyle().Foreground(white),
			FullSeparator:  lipgloss.NewStyle().Foreground(white),
		},
	}
}
