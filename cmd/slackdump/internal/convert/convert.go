package convert

import (
	"archive/zip"
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/convert/format"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
)

var CmdConvert = &base.Command{
	Run:       runConvert,
	UsageLine: "slackdump convert [flags] <format> <file.json>",
	Short:     "converts the json files to other formats",
	Long: base.Render(`
# Convert Command
`),
	CustomFlags: false,
	FlagMask:    cfg.OmitAll & ^cfg.OmitWorkspaceFlag,
	PrintFlags:  true,
	RequireAuth: false,
}

type datatype uint8

const (
	dtUnknown = iota
	dtConversation
	dtChannels
	dtUsers
)

var typemap = map[string]datatype{
	"C":        dtConversation, // channel
	"G":        dtConversation, // group conversation
	"D":        dtConversation, // private msg
	"channels": dtChannels,
	"users":    dtUsers,
}

var ErrUnknown = errors.New("unknown data type")

var (
	archive string
)

func init() {
	CmdConvert.Flag.StringVar(&archive, "zip", "", "access the file within the ZIP `archive.zip`")
}

func runConvert(ctx context.Context, cmd *base.Command, args []string) error {
	if len(args) < 2 {
		base.SetExitStatus(base.SInvalidParameters)
		return errors.New("must specify output format and a file to convert")
	}

	var convType format.Type
	if err := convType.Set(args[0]); err != nil {
		base.SetExitStatus(base.SInvalidParameters)
		return err
	}
	file := args[1]
	// get users somehow.

	// get converter
	if err := convertData(ctx, format.NewText(3*time.Minute), archive, file); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	return nil
}

func convertData(ctx context.Context, cvt format.Converter, archive string, filename string) error {
	dt := typeof(filename)
	if dt == dtUnknown {
		return ErrUnknown
	}
	var fsys fs.FS
	if archive != "" {
		zr, err := zip.OpenReader(archive)
		if err != nil {
			return err
		}
		defer zr.Close()
		fsys = zr
	} else {
		fsys = os.DirFS(filepath.Dir(filename))
		filename = filepath.Base(filename)
	}

	// open target
	f, err := fsys.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	// convert
	if err := convert(ctx, os.Stdout, dt, f); err != nil {
		return err
	}
	return nil
}

func convert(ctx context.Context, w io.Writer, dt datatype, r io.Reader) (err error) {
	switch dt {
	default:
		err = ErrUnknown
	case dtConversation:
		// convert convo
	case dtChannels:
		// convert Channels
	case dtUsers:
		// convert Users
	}
	return
}

func typeof(filename string) (dt datatype) {
	name := filepath.Base(filename)
	if filepath.Ext(name) != ".json" {
		return
	}
	for k, v := range typemap {
		if strings.HasPrefix(name, k) {
			dt = v
			break
		}
	}
	return
}
