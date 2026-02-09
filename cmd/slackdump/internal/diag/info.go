// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package diag

import (
	"context"
	"encoding/json"
	"io"
	"os"

	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/diag/info"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/golang/base"
)

// cmdInfo is the information command.
var cmdInfo = &base.Command{
	UsageLine:  "slackdump tools info [flags]",
	Short:      "show information about Slackdump environment",
	Run:        runInfo,
	FlagMask:   cfg.OmitAll &^ cfg.OmitCacheDir,
	PrintFlags: true,

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
