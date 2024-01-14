package wizard

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
)

var CmdWizard = &base.Command{
	Run:       nil, //initialised in init to prevent initialisation cycle
	UsageLine: "wiz",
	Short:     "Slackdump Wizard",
	Long: `
Slackdump Wizard guides through the dumping process.
`,
	RequireAuth: true,
}

var titlecase = cases.Title(language.English)

type menuitem struct {
	Name        string
	Description string
	cmd         *base.Command
	Submenu     *menu
}

type menu struct {
	title string
	names []string
	items []menuitem
}

func (m *menu) Add(item menuitem) {
	m.items = append(m.items, item)
	m.names = append(m.names, item.Name)
}

func init() {
	CmdWizard.Run = runWizard
}

func runWizard(ctx context.Context, cmd *base.Command, args []string) error {
	baseCommands := base.Slackdump.Commands
	if len(baseCommands) == 0 {
		panic("internal error:  no commands")
	}

	menu := makeMenu(baseCommands, "", "What would you like to do?")
	if err := show(menu, func(cmd *base.Command) error {
		return cmd.Wizard(ctx, cmd, args)
	}); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return fmt.Errorf("error running wizard: %s", err)
	}
	return nil
}

var (
	miBack = menuitem{
		Name:        "<< Back",
		Description: "",
	}
	miExit = menuitem{
		Name:        "Exit",
		Description: "",
	}
)

func makeMenu(cmds []*base.Command, parent string, title string) (m *menu) {
	m = &menu{
		title: title,
		names: make([]string, 0, len(cmds)),
		items: make([]menuitem, 0, len(cmds)),
	}
	if parent != "" {
		parent += " "
	}
	for _, cmd := range cmds {
		hasSubcommands := len(cmd.Commands) > 0
		hasWizard := cmd.Wizard != nil
		isMe := strings.EqualFold(cmd.Name(), CmdWizard.Name())
		if !(hasWizard || hasSubcommands) || isMe {
			continue
		}
		name := titlecase.String(cmd.Name())
		item := menuitem{
			// Name:        parent + name,
			Name:        name,
			Description: cmd.Short,
			cmd:         cmd,
		}
		if len(cmd.Commands) > 0 {
			item.Submenu = makeMenu(cmd.Commands, name, name)
		}

		m.Add(item)
	}
	if parent == "" {
		m.Add(miExit)
	} else {
		m.Add(miBack)
	}
	return
}

func show(m *menu, onMatch func(cmd *base.Command) error) error {
	var options []huh.Option[string]
	for i, name := range m.names {
		var text = fmt.Sprintf("%-10s - %s", name, m.items[i].Description)
		if m.items[i].Description == "" {
			text = fmt.Sprintf("%-10s", name)
		}
		options = append(options, huh.NewOption(text, name))
	}
	for {
		var resp string
		err := huh.NewSelect[string]().
			Title(m.title).
			// Options(huh.NewOptions(m.names...)...).
			Options(options...).
			Value(&resp).
			Run()
		if err != nil {
			return err
		}
		if err := run(m, resp, onMatch); err != nil {
			if errors.Is(err, errBack) {
				return nil
			} else if errors.Is(err, errInvalid) {
				return err
			} else {
				return err
			}
		}
	}

}

var (
	errBack    = errors.New("back")
	errInvalid = errors.New("invalid choice")
)

func run(m *menu, choice string, onMatch func(cmd *base.Command) error) error {
	for _, mi := range m.items {
		if choice != mi.Name {
			continue
		}
		if mi.Submenu != nil {
			return show(mi.Submenu, onMatch)
		}
		// found
		if mi.cmd == nil { // only Exit and back won't have this.
			return errBack
		}
		return onMatch(mi.cmd)
	}
	return errInvalid
}
