package diag

import (
	"context"
	"encoding/json"
	"os"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/diag/info"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
)

// CmdInfo is the information command.
var CmdInfo = &base.Command{
	UsageLine: "slackdump tools info",
	Short:     "show information about slackdump environment",
	Run:       runInfo,
	Long: `# Info Command
	
**Info** shows information about Slackdump environment, such as local system paths, etc.
`,
}

func runInfo(ctx context.Context, cmd *base.Command, args []string) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(info.Collect()); err != nil {
		return err
	}

	return nil
}
