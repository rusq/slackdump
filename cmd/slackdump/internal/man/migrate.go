package man

import (
	_ "embed"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
)

//go:embed assets/v2migr.md
var mdMigrate string

var Migration = &base.Command{
	UsageLine: "v2migrate",
	Short:     "Migrating from V2 notes",
	Long:      mdMigrate,
}
