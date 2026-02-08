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
package info

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/rusq/slackdump/v4/auth"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/workspace"
)

func CollectAuth(ctx context.Context, w io.Writer) error {
	fmt.Fprintln(os.Stderr, "To confirm the operation, please enter your OS password.")
	if err := osValidateUser(ctx, os.Stderr); err != nil {
		return err
	}
	m, err := workspace.CacheMgr()
	if err != nil {
		return fmt.Errorf("cache error: %w", err)
	}
	cur, err := m.Current()
	if err != nil {
		return fmt.Errorf("cache error: %w", err)
	}
	fi, err := m.FileInfo(cur)
	if err != nil {
		return fmt.Errorf("cache error: %w", err)
	}
	f, err := m.Open(filepath.Join(cfg.CacheDir(), fi.Name()))
	if err != nil {
		return fmt.Errorf("cache error: %w", err)
	}
	defer f.Close()
	prov, err := auth.Load(f)
	if err != nil {
		return fmt.Errorf("cache error: %w", err)
	}
	fmt.Fprintf(w, "TOKEN=%s\n", prov.Token)
	if err := dumpCookiesMozilla(ctx, w, prov.Cookies()); err != nil {
		return err
	}
	return nil
}

// dumpCookiesMozilla dumps cookies in Mozilla format.
func dumpCookiesMozilla(_ context.Context, w io.Writer, cookies []*http.Cookie) error {
	for _, c := range cookies {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%s\t%s\n", c.Domain, "TRUE", c.Path, strings.ToUpper(fmt.Sprintf("%v", c.Secure)), c.Expires.Unix(), c.Name, c.Value)
	}
	return nil
}
