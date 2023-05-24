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
	UsageLine: "slackdump tools obfuscate [options] [input] [output]",
	Short:     "obfuscate sensitive data in a slackdump recording",
	Long: `
# Obfuscate tool

Obfuscate tool obfuscates sensitive data in a slackdump chunk recording.

To record the API output into a chunk, you can run ` + "`slackdump tools record stream`" + `.
`,
	CustomFlags: true,
	PrintFlags:  true,
}

var obfparam struct {
	input     string
	output    string
	overwrite bool
	seed      int64
}

func init() {
	CmdObfuscate.Run = runObfuscate

	CmdObfuscate.Flag.BoolVar(&obfparam.overwrite, "f", false, "force overwrite")
	CmdObfuscate.Flag.Int64Var(&obfparam.seed, "seed", time.Now().UnixNano(), "seed for the random number generator")
}

const (
	otUnknown = iota
	otTerm
	otFile
	otDir
	otNotExist
)

func objtype(name string) int {
	if isTerm(name) {
		return otTerm
	}
	fi, err := os.Stat(name)
	if err != nil {
		if os.IsNotExist(err) {
			return otNotExist
		}
		return otUnknown
	}
	if fi.IsDir() {
		return otDir
	}
	return otFile
}

func runObfuscate(ctx context.Context, cmd *base.Command, args []string) error {
	if err := CmdObfuscate.Flag.Parse(args); err != nil {
		return err
	}

	if CmdObfuscate.Flag.NArg() == 2 {
		obfparam.input = CmdObfuscate.Flag.Arg(0)
		obfparam.output = CmdObfuscate.Flag.Arg(1)
	} else if CmdObfuscate.Flag.NArg() == 1 {
		obfparam.input = CmdObfuscate.Flag.Arg(0)
		obfparam.output = "-"
	} else {
		obfparam.input = "-"
		obfparam.output = "-"
	}

	inType := objtype(obfparam.input)

	var fn func(context.Context) error
	if inType == otFile || inType == otTerm {
		fn = obfFile
	} else if inType == otDir {
		fn = obfDir
	} else {
		base.SetExitStatus(base.SInvalidParameters)
		return fmt.Errorf("input %s is invalid", obfparam.input)
	}
	if err := fn(ctx); err != nil {
		return err
	}
	fmt.Println("OK")
	return nil
}

var (
	ErrObfTargetExist = errors.New("target exists, and overwrite flag not set")
	ErrObfSame        = errors.New("input and output are the same")
)

type ErrObfIncompat struct {
	OutType string
	InType  string
	OutName string
	InName  string
}

func (e *ErrObfIncompat) Error() string {
	return fmt.Sprintf("%s output %s is incompatible with %s input: %s", e.OutType, e.OutName, e.InType, e.InName)
}

func isTerm(name string) bool {
	return name == "-" || name == ""
}

func obfFile(ctx context.Context) error {
	var (
		in  io.ReadCloser
		out io.WriteCloser
		err error
	)
	if isTerm(obfparam.input) {
		in = os.Stdin
	} else {
		in, err = os.Open(obfparam.input)
		if err != nil {
			return err
		}
		defer in.Close()
	}

	outType := objtype(obfparam.output)
	switch outType {
	case otDir:
		base.SetExitStatus(base.SInvalidParameters)
		return &ErrObfIncompat{
			OutType: "directory",
			InType:  "non-directory",
			OutName: obfparam.output,
			InName:  obfparam.input,
		}
	case otFile:
		if obfparam.input == obfparam.output {
			base.SetExitStatus(base.SInvalidParameters)
			return ErrObfSame
		}
		if !obfparam.overwrite {
			base.SetExitStatus(base.SUserError)
			return ErrObfTargetExist
		}
		// ok
	case otNotExist, otTerm:
		// ok
	case otUnknown:
		fallthrough
	default:
		base.SetExitStatus(base.SInvalidParameters)
		return fmt.Errorf("output %s is invalid", obfparam.output)
	}

	if outType == otTerm {
		out = os.Stdout
	} else {
		out, err = os.Create(obfparam.output)
		if err != nil {
			return err
		}
		defer out.Close()
	}
	if err := obfuscate.Do(ctx, out, in, obfuscate.WithSeed(obfparam.seed)); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	return nil
}

func obfDir(ctx context.Context) error {
	outType := objtype(obfparam.output)
	switch outType {
	case otFile:
		base.SetExitStatus(base.SInvalidParameters)
		return &ErrObfIncompat{
			OutType: "non-directory",
			InType:  "directory",
			OutName: obfparam.output,
			InName:  obfparam.input,
		}
	case otNotExist:
		if err := os.MkdirAll(obfparam.output, 0755); err != nil {
			return err
		}
	case otDir:
		if obfparam.input == obfparam.output {
			base.SetExitStatus(base.SInvalidParameters)
			return ErrObfSame
		}
		if !obfparam.overwrite {
			base.SetExitStatus(base.SUserError)
			return ErrObfTargetExist
		}
		if err := os.RemoveAll(obfparam.output); err != nil {
			base.SetExitStatus(base.SApplicationError)
			return err
		}

	case otUnknown, otTerm:
		fallthrough
	default:
		base.SetExitStatus(base.SInvalidParameters)
		return fmt.Errorf("output %s is invalid", obfparam.output)
	}

	return obfuscate.DoDir(
		ctx,
		obfparam.input,
		obfparam.output,
		obfuscate.WithSeed(obfparam.seed),
	)
}
