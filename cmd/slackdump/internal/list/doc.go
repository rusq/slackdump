package list

import (
	"fmt"

	"github.com/rusq/slackdump/v3/internal/format"
)

var (
	sectListFormat = `
## Listing format

By default, the data is being output in TEXT format.  You can choose the listing
format by specifying "-format X" flag, where X is one of: ` + fmt.Sprint(format.All())
)
