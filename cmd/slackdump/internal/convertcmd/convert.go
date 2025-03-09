package convertcmd

import (
	"context"
	_ "embed"
	"errors"
	"os"
	"time"

	"github.com/rusq/fsadapter"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/bootstrap"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/chunk/dbproc"
	"github.com/rusq/slackdump/v3/internal/chunk/dirproc"
	"github.com/rusq/slackdump/v3/internal/chunk/transform/fileproc"
	"github.com/rusq/slackdump/v3/internal/convert"
	"github.com/rusq/slackdump/v3/internal/source"
)

//go:embed assets/convert.md
var convertMd string

var CmdConvert = &base.Command{
	Run:         runConvert,
	UsageLine:   "slackdump convert [flags] <source>",
	Short:       "convert slackdump chunks to various formats",
	Long:        convertMd,
	CustomFlags: false,
	FlagMask:    cfg.OmitAll & ^cfg.OmitDownloadFlag &^ cfg.OmitOutputFlag &^ cfg.OmitDownloadAvatarsFlag,
	PrintFlags:  true,
}

var (
	// ErrFormat is returned when the target format is not supported.
	ErrFormat = errors.New("unsupported target format")
	// ErrSource is returned when the source type is not supported for the chosen target type.
	ErrSource = errors.New("unsupported source type")
	// ErrStorage is returned when the storage type is not supported.
	ErrStorage = errors.New("unsupported storage type")
)

type tparams struct {
	storageType fileproc.StorageType
	outputfmt   datafmt
}

var params = tparams{
	storageType: fileproc.STmattermost,
	outputfmt:   Fexport,
}

func init() {
	CmdConvert.Flag.Var(&params.storageType, "storage", "storage type")
	CmdConvert.Flag.Var(&params.outputfmt, "output", "output format")
}

func runConvert(ctx context.Context, cmd *base.Command, args []string) error {
	if len(args) < 1 {
		base.SetExitStatus(base.SInvalidParameters)
		return errors.New("source and destination are required")
	}
	fn, exist := converter(params.outputfmt)
	if !exist {
		base.SetExitStatus(base.SInvalidParameters)
		return ErrFormat
	}

	lg := cfg.Log
	lg.InfoContext(ctx, "converting", "source", args[0], "output_format", params.outputfmt, "output", cfg.Output)

	cflg := convertflags{
		withFiles:   cfg.WithFiles,
		withAvatars: cfg.WithAvatars,
		stt:         params.storageType,
	}
	start := time.Now()
	if err := fn(ctx, args[0], cfg.Output, cflg); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}

	lg.InfoContext(ctx, "completed", "took", time.Since(start))
	return nil
}

func converter(output datafmt) (convertFunc, bool) {
	if cvt, ok := converters[output]; ok {
		return cvt, true
	}
	return nil, false
}

type convertFunc func(ctx context.Context, input, output string, cflg convertflags) error

var converters = map[datafmt]convertFunc{
	Fexport: toExport,
	Fchunk:  toChunk,
}

type convertflags struct {
	withFiles   bool
	withAvatars bool
	stt         fileproc.StorageType
}

func toExport(ctx context.Context, src, trg string, cflg convertflags) error {
	// detect source type
	st, err := source.Type(src)
	if err != nil {
		return err
	}

	if st == source.FUnknown {
		return ErrSource
	}

	fsa, err := fsadapter.New(trg)
	if err != nil {
		return err
	}
	defer fsa.Close()

	sttFn, ok := fileproc.StorageTypeFuncs[cflg.stt]
	if !ok {
		return ErrStorage
	}

	var (
		includeFiles   = cflg.withFiles && (st&source.FMattermost != 0)
		includeAvatars = cflg.withAvatars && (st&source.FAvatars != 0)
	)

	s, err := source.Load(ctx, src)
	if err != nil {
		return err
	}

	cvt := convert.NewToExport(
		s,
		fsa,
		convert.WithIncludeFiles(includeFiles),
		convert.WithIncludeAvatars(includeAvatars),
		convert.WithSrcFileLoc(sttFn),
		convert.WithTrgFileLoc(sttFn),
		convert.WithLogger(cfg.Log),
	)
	if err := cvt.Convert(ctx); err != nil {
		return err
	}

	return nil
}

func toChunk(ctx context.Context, src, trg string, cflg convertflags) error {
	// detect source type
	st, err := source.Type(src)
	if err != nil {
		return err
	}
	if !st.Has(source.FDatabase) {
		return ErrSource
	}
	wconn, _, err := bootstrap.Database(src, "convert")
	if err != nil {
		return err
	}
	defer wconn.Close()

	dir := cfg.StripZipExt(trg)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	remove := true
	defer func() {
		// remove on failed conversion
		if remove {
			_ = os.RemoveAll(dir)
		}
	}()

	cd, err := chunk.OpenDir(dir)
	if err != nil {
		return err
	}
	erc := dirproc.NewERC(cd, cfg.Log)
	defer erc.Close()

	remove = false // init succeeded

	// TODO: how to find the session id?
	ch := dbproc.Chunker{SessionID: 1}
	if err := ch.ToChunk(ctx, wconn, erc); err != nil {
		return err
	}
	return nil
}
