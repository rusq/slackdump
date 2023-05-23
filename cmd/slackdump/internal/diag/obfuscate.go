package diag

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v2/internal/chunk/obfuscate"
)

// CmdObfuscate is the command to obfuscate sensitive data in a slackdump
// recording.
var CmdObfuscate = &base.Command{
	UsageLine: "slackdump tools obfuscate [options] [file]",
	Short:     "obfuscate sensitive data in a slackdump recording",
	Long: `
# Obfuscate tool

Obfuscate tool obfuscates sensitive data in a slackdump chunk recording.

To record the API output into a chunk, you can run ` + "`slackdump tools record stream`" + `.
`,
	CustomFlags: true,
	PrintFlags:  true,
}

var obfuscateParams struct {
	input     string
	output    string
	overwrite bool
	seed      int64
}

func init() {
	CmdObfuscate.Run = runObfuscate

	CmdObfuscate.Flag.StringVar(&obfuscateParams.input, "i", "", "input file or directory, if not specified, stdin is used")
	CmdObfuscate.Flag.StringVar(&obfuscateParams.output, "o", "", "output file or directory, if not specified, stdout is used")
	CmdObfuscate.Flag.BoolVar(&obfuscateParams.overwrite, "f", false, "force overwrite")
	CmdObfuscate.Flag.Int64Var(&obfuscateParams.seed, "seed", time.Now().UnixNano(), "seed for the random number generator")
}

const (
	otTerm = iota
	otFile
	otDir
)

func objtype(name string) (int, error) {
	if name == "-" || name == "" {
		return otTerm, nil
	}
	fi, err := os.Stat(name)
	if err != nil {
		return otTerm, err
	}
	if fi.IsDir() {
		return otDir, nil
	}
	return otFile, nil
}

func runObfuscate(ctx context.Context, cmd *base.Command, args []string) error {
	if err := CmdObfuscate.Flag.Parse(args); err != nil {
		return err
	}

	inType, err := objtype(obfuscateParams.input)
	if err != nil {
		base.SetExitStatus(base.SInvalidParameters)
		return err
	}

	if inType == otFile || inType == otTerm {
		return obfFile(ctx)
	} else {
		return obfDir(ctx)
	}
}

var (
	ErrObfTargetExist = errors.New("target exists, and overwrite flag not set")
)

type ErrObfIncompat struct {
	Output string
	Input  string
	Name   string
}

func (e *ErrObfIncompat) Error() string {
	return fmt.Sprint("%s output %s is incompatible with %s input: %s", e.Output, e.Name, e.Input)
}

func obfFile(ctx context.Context) error {
	var (
		in  io.ReadCloser
		out io.WriteCloser
		err error
	)
	if obfuscateParams.input == "" {
		in = os.Stdin
	} else {
		in, err = os.Open(obfuscateParams.input)
		if err != nil {
			return err
		}
		defer in.Close()
	}

	outType, err := objtype(obfuscateParams.output)
	if err != nil && !os.IsNotExist(err) {
		base.SetExitStatus(base.SGenericError)
		return err
	} else if err == nil && !obfuscateParams.overwrite {
		// object exists but overwrite not set
		base.SetExitStatus(base.SUserError)
		return ErrObfTargetExist
	} else if outType == otDir {
		base.SetExitStatus(base.SInvalidParameters)
		return &ErrObfIncompat{
			Output: "directory",
			Input:  "non-directory",
			Name:   obfuscateParams.output,
		}
	}

	if outType == otTerm {
		out = os.Stdout
	} else {
		out, err = os.Create(obfuscateParams.output)
		if err != nil {
			return err
		}
		defer out.Close()
	}
	if err := obfuscate.Do(ctx, out, in, obfuscate.WithSeed(obfuscateParams.seed)); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	return nil
}

func obfDir(ctx context.Context) error {
	outType, err := objtype(obfuscateParams.output)
	if err == nil {
		if outType != otDir {
			base.SetExitStatus(base.SInvalidParameters)
			return &ErrObfIncompat{
				Output: "non-directory",
				Input:  "directory",
				Name:   obfuscateParams.output,
			}
		}
		if !obfuscateParams.overwrite {
			base.SetExitStatus(base.SUserError)
			return ErrObfTargetExist
		}
	}
	return obfuscate.DoDir(
		ctx,
		obfuscateParams.input,
		obfuscateParams.output,
		obfuscate.WithSeed(obfuscateParams.seed),
	)
}
