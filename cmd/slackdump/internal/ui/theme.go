package ui

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

var HuhTheme = ThemeBase16Ext() // Theme is the default Wizard theme.

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

	// Cursor is the pointer to the selected item, i.e. the ">" in a list.
	Cursor lipgloss.Style
	// SelectedLine is the style for the selected line in a list, next to the pointer.
	SelectedLine lipgloss.Style
	Unselected   lipgloss.Style

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

var (
	black  = lipgloss.Color("0")
	red    = lipgloss.Color("1")
	green  = lipgloss.Color("2")
	blue   = lipgloss.Color("4")
	yellow = lipgloss.Color("3")
	cyan   = lipgloss.AdaptiveColor{Light: "4", Dark: "6"}
	purple = lipgloss.Color("5")
	white  = lipgloss.AdaptiveColor{Light: "0", Dark: "7"}
	gray   = lipgloss.Color("8")
	ltred  = lipgloss.Color("9")
)

func DefaultTheme() Theme {
	// https://gist.github.com/JBlond/2fea43a3049b38287e5e9c`efc87b2124
	return Theme{
		Focused: ControlStyle{
			Border:         lipgloss.NewStyle().BorderLeft(true).BorderForeground(cyan).BorderStyle(lipgloss.ThickBorder()).Padding(0, 1),
			Title:          lipgloss.NewStyle().Foreground(green).Bold(true),
			Description:    lipgloss.NewStyle().Foreground(gray),
			Cursor:         lipgloss.NewStyle().Foreground(yellow),
			SelectedLine:   lipgloss.NewStyle().Background(green).Foreground(black),
			Unselected:     lipgloss.NewStyle().Foreground(white),
			SelectedFile:   lipgloss.NewStyle().Foreground(green),
			UnselectedFile: lipgloss.NewStyle().Foreground(white),
			DisabledFile:   lipgloss.NewStyle().Foreground(gray),
			Directory:      lipgloss.NewStyle().Foreground(blue),
			Text:           lipgloss.NewStyle().Foreground(white),
			Options: OptionStyle{
				Section:       lipgloss.NewStyle().Foreground(cyan).Bold(true),
				Name:          lipgloss.NewStyle().Foreground(green),
				EnabledValue:  lipgloss.NewStyle().Foreground(white),
				DisabledValue: lipgloss.NewStyle().Foreground(green),
				SelectedName:  lipgloss.NewStyle().Foreground(black).Underline(true).Background(green),
			},
		},
		Blurred: ControlStyle{
			Border:         lipgloss.NewStyle().BorderLeft(true).BorderStyle(lipgloss.NormalBorder()).Padding(0, 1),
			Title:          lipgloss.NewStyle().Foreground(gray).Bold(true),
			Description:    lipgloss.NewStyle().Foreground(gray),
			Cursor:         lipgloss.NewStyle().Foreground(gray),
			SelectedLine:   lipgloss.NewStyle().Background(gray).Foreground(black),
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

// ThemeBase16Ext returns a modified Base16 theme based on huh.ThemeBase16.
func ThemeBase16Ext() *huh.Theme {
	t := huh.ThemeBase()

	t.Focused.Base = t.Focused.Base.BorderForeground(gray)
	t.Focused.Title = t.Focused.Title.Foreground(cyan)
	t.Focused.NoteTitle = t.Focused.NoteTitle.Foreground(cyan)
	t.Focused.Directory = t.Focused.Directory.Foreground(cyan)
	t.Focused.Description = t.Focused.Description.Foreground(gray)
	t.Focused.ErrorIndicator = t.Focused.ErrorIndicator.Foreground(ltred)
	t.Focused.ErrorMessage = t.Focused.ErrorMessage.Foreground(ltred)
	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(yellow)
	t.Focused.NextIndicator = t.Focused.NextIndicator.Foreground(yellow)
	t.Focused.PrevIndicator = t.Focused.PrevIndicator.Foreground(yellow)
	t.Focused.Option = t.Focused.Option.Foreground(white)
	t.Focused.MultiSelectSelector = t.Focused.MultiSelectSelector.Foreground(yellow)
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(black).Background(green)
	t.Focused.SelectedPrefix = t.Focused.SelectedPrefix.Foreground(green)
	t.Focused.UnselectedOption = t.Focused.UnselectedOption.Foreground(white)
	t.Focused.FocusedButton = t.Focused.FocusedButton.Foreground(white).Background(purple)
	t.Focused.BlurredButton = t.Focused.BlurredButton.Foreground(white).Background(black)

	t.Focused.TextInput.Cursor.Foreground(purple)
	t.Focused.TextInput.Placeholder.Foreground(gray)
	t.Focused.TextInput.Prompt.Foreground(yellow)

	t.Blurred = t.Focused
	t.Blurred.Base = t.Blurred.Base.BorderStyle(lipgloss.HiddenBorder())
	t.Blurred.NoteTitle = t.Blurred.NoteTitle.Foreground(gray)
	t.Blurred.Title = t.Blurred.NoteTitle.Foreground(gray)

	t.Blurred.TextInput.Prompt = t.Blurred.TextInput.Prompt.Foreground(gray)
	t.Blurred.TextInput.Text = t.Blurred.TextInput.Text.Foreground(white)

	t.Blurred.NextIndicator = lipgloss.NewStyle()
	t.Blurred.PrevIndicator = lipgloss.NewStyle()

	return t
}
