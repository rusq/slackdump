package export

import (
	"context"
	"errors"
	"regexp"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/cfgui"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/dumpui"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/updaters"
)

func wizExport(ctx context.Context, cmd *base.Command, args []string) error {
	w := &dumpui.Wizard{
		Title:       "Export Slack Workspace",
		Name:        "Export",
		Cmd:         cmd,
		LocalConfig: options.configuration,
	}
	return w.Run(ctx)
}

func (fl *exportFlags) configuration() cfgui.Configuration {
	return cfgui.Configuration{
		{
			Name: "Optional",
			Params: []cfgui.Parameter{
				{
					Name:        "Export Storage Type",
					Value:       fl.ExportStorageType.String(),
					Description: "Export file storage type",
					Inline:      true,
					// TODO: V3 Implement Updater for ExportStorageType
				},
				{
					Name:        "Member Only",
					Value:       cfgui.Checkbox(fl.MemberOnly),
					Description: "Export only channels, which current user belongs to",
					Inline:      true,
					Updater:     updaters.NewBool(&fl.MemberOnly),
				},
				{
					Name:        "Export Token",
					Value:       fl.ExportToken,
					Description: "File export token to append to each of the file URLs",
					Inline:      true,
					Updater:     updaters.NewString(&fl.ExportToken, "", false, validateToken),
				},
			},
		},
	}
}

// tokenRe is a loose regular expression to match Slack API tokens.
// a - app, b - bot, c - client, e - export, p - legacy
var tokenRE = regexp.MustCompile(`xox[abcep]-[0-9]+-[0-9]+-[0-9]+-[0-9a-z]{64}`)

var errInvalidToken = errors.New("token must start with xoxa-, xoxb-, xoxc- or xoxe- and be followed by 4 numbers and 64 lowercase letters")

func validateToken(token string) error {
	if !tokenRE.MatchString(token) {
		return errInvalidToken
	}
	return nil
}
