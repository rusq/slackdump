package list

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/rusq/dlog"
	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/auth"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/convert/format"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v2/fsadapter"
	"github.com/rusq/slackdump/v2/types"
	"github.com/slack-go/slack"
)

var CmdList = &base.Command{
	UsageLine: "slackdump list",
	Short:     "list users or channels",
	Long: `
List lists users or channels for the Slack Workspace.  It may take a while on
large workspaces, as Slack limits the amount of requests on it's own discretion,
which is sometimes unreasonably slow.
`,
	Commands: []*base.Command{
		CmdListUsers,
		CmdListChannels,
	},
}

// common flags
var (
	listType     format.Type = format.CText
	screenOutput bool
)

func init() {
	for _, cmd := range CmdList.Commands {
		addCommonFlags(&cmd.Flag)
	}
}

// addCommonFlags adds common flags to the flagset.
func addCommonFlags(fs *flag.FlagSet) {
	fs.Var(&listType, "format", fmt.Sprintf("listing format, should be one of: %v", format.All()))
	fs.BoolVar(&screenOutput, "screen", false, "output to screen instead of file")
}

func serialise(fs fsadapter.FS, name string, a any) error {
	f, err := fs.Create(name)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	if err := enc.Encode(a); err != nil {
		return err
	}
	return nil
}

// listFunc is a function that lists something from the Slack API.  It should
// return the object from the api, a filename to save the data to and an
// error.
type listFunc func(ctx context.Context, sess *slackdump.Session) (a any, filename string, err error)

// list authenticates and creates a slackdump instance, then calls a listFn.
// listFn must return the object from the api, a JSON filename and an error.
func list(ctx context.Context, listFn listFunc) error {
	if listType == format.CUnknown {
		return errors.New("unknown listing format, seek help")
	}

	prov, err := auth.FromContext(ctx)
	if err != nil {
		base.SetExitStatus(base.SAuthError)
		return err
	}
	sess, err := slackdump.NewWithOptions(ctx, prov, cfg.SlackOptions)
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}

	data, filename, err := listFn(ctx, sess)
	if err != nil {
		return err
	}

	// if screenOutput is true, print to stdout, otherwise save to a file.
	if screenOutput {
		return fmtPrint(ctx, os.Stdout, data, listType, sess.Users)
	} else {
		// save to a filesystem.
		fs, err := fsadapter.New(cfg.BaseLoc)
		if err != nil {
			base.SetExitStatus(base.SApplicationError)
			return err
		}
		defer fs.Close()
		if err := serialise(fs, filename, data); err != nil {
			return err
		}
		dlog.FromContext(ctx).Printf("data saved to %q\n", filepath.Join(cfg.BaseLoc, filename))
	}

	return nil
}

// fmtPrint prints the given data to the given writer, using the given format.
// It should be supplied with prepopulated users, as it may need to look up
// users by ID.
func fmtPrint(ctx context.Context, w io.Writer, a any, typ format.Type, u []slack.User) error {
	// get the converter
	initFn, ok := format.Converters[typ]
	if !ok {
		return fmt.Errorf("unknown converter type: %s", typ)
	}
	cvt := initFn()

	// currently there's no list function for conversations, because it
	// requires additional options, and I don't want to clutter the flags -
	// there's already too many.
	switch val := a.(type) {
	case types.Channels:
		return cvt.Channels(ctx, w, u, val)
	case types.Users:
		return cvt.Users(ctx, w, val)

	default:
		return fmt.Errorf("unsupported data type: %T", a)
	}
	// unreachable
}

// extmap maps a format.Type to a file extension.
var extmap = map[format.Type]string{
	format.CText: "txt",
	format.CJSON: "json",
	format.CCSV:  "csv",
}

// makeFilename makes a filename for the given prefix, teamID and listType.
func makeFilename(prefix string, teamID string, listType format.Type) string {
	ext, ok := extmap[listType]
	if !ok {
		panic(fmt.Sprintf("unknown list type: %v", listType))
	}
	return fmt.Sprintf("%s-%s.%s", prefix, teamID, ext)
}
