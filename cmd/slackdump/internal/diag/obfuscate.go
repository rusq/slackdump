package diag

import (
	"context"
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

To record the chunk, you can run ` + "`slackdump tools record stream`" + `.
`,
	CustomFlags: true,
	PrintFlags:  true,
}

var obfuscateParams struct {
	inputFile  string
	outputFile string
	seed       int64
}

func init() {
	CmdObfuscate.Run = runObfuscate

	CmdObfuscate.Flag.StringVar(&obfuscateParams.inputFile, "i", "", "input file, if not specified, stdin is used")
	CmdObfuscate.Flag.StringVar(&obfuscateParams.outputFile, "o", "", "output file, if not specified, stdout is used")
	CmdObfuscate.Flag.Int64Var(&obfuscateParams.seed, "seed", time.Now().UnixNano(), "seed for the random number generator")
}

func runObfuscate(ctx context.Context, cmd *base.Command, args []string) error {
	if err := CmdObfuscate.Flag.Parse(args); err != nil {
		return err
	}

	var (
		in  io.ReadCloser
		out io.WriteCloser
		err error
	)
	if obfuscateParams.inputFile == "" {
		in = os.Stdin
	} else {
		in, err = os.Open(obfuscateParams.inputFile)
		if err != nil {
			return err
		}
		defer in.Close()
	}

	if obfuscateParams.outputFile == "" {
		out = os.Stdout
	} else {
		out, err = os.Create(obfuscateParams.outputFile)
		if err != nil {
			return err
		}
		defer out.Close()
	}
	return obfuscate.Do(ctx, out, in, obfuscate.WithSeed(obfuscateParams.seed))
}
