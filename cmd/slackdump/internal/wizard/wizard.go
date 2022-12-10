package wizard

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
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
			Name:        parent + name,
			Description: cmd.Short,
			cmd:         cmd,
		}
		if len(cmd.Commands) > 0 {
			item.Submenu = makeMenu(cmd.Commands, name, name)
		}

		m.Add(item)
	}
	if parent == "" {
		m.Add(menuitem{
			Name: "Exit",
		})
	} else {
		m.Add(menuitem{
			Name: "<< Back",
		})
	}
	return
}

func show(m *menu, onMatch func(cmd *base.Command) error) error {
	for {
		mode := &survey.Select{
			Message: m.title,
			Options: m.names,
			Description: func(value string, index int) string {
				return m.items[index].Description
			},
		}
		var resp string
		if err := survey.AskOne(mode, &resp); err != nil {
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
