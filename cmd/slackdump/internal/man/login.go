package man

import (
	_ "embed"

	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
)

//go:embed assets/login.md
var mdLogin string

var Login = &base.Command{
	UsageLine: "login",
	Short:     "login related information",
	Long:      mdLogin,
}
