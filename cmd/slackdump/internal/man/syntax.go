package man

import (
	_ "embed"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
)

//go:embed assets/syntax.md
var syntaxMD string

var Syntax = &base.Command{
	Run:       nil,
	Wizard:    nil,
	UsageLine: "slackdump syntax",
	Short:     "how to specify channels to dump",
	Long:      syntaxMD,
}
