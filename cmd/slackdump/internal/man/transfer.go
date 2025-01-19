package man

import (
	_ "embed"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
)

//go:embed assets/transfer.md
var transferMd string

var Transfer = &base.Command{
	UsageLine: "transfer",
	Short:     "transfering credentials to another system",
	Long:      transferMd,
}
