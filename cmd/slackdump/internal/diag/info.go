package diag

import (
	"context"
	"encoding/json"
	"io"
	"os"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/diag/info"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
)

// cmdInfo is the information command.
var cmdInfo = &base.Command{
	UsageLine: "slackdump tools info",
	Short:     "show information about Slackdump environment",
	Run:       runInfo,

	Long: `# Info Command
	
**Info** shows information about Slackdump environment, such as local system paths, etc.
`,
}

var infoParams = struct {
	auth bool
}{
	auth: false,
}

func init() {
	cmdInfo.Flag.BoolVar(&infoParams.auth, "auth", false, "show authentication diagnostic information")
}

func runInfo(ctx context.Context, cmd *base.Command, args []string) error {
	switch {
	case infoParams.auth:
		return runAuthInfo(ctx, os.Stdout)
	default:
		return runGeneralInfo(ctx, os.Stdout)
	}
}

func runAuthInfo(ctx context.Context, w io.Writer) error {
	return info.CollectAuth(ctx, w)
}

func runGeneralInfo(_ context.Context, w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(info.Collect()); err != nil {
		return err
	}

	return nil
}
