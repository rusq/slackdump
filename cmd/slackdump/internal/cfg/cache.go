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

package cfg

import (
	"log/slog"
	"os"
	"path/filepath"
)

const (
	cacheDirName = "slackdump"
)

// ucd detects user cache dir and returns slack cache directory name.
func ucd(ucdFn func() (string, error)) string {
	ucd, err := ucdFn()
	if err != nil {
		slog.Debug("ucd", "error", err)
		return "."
	}
	return filepath.Join(ucd, cacheDirName)
}

func CacheDir() string {
	if LocalCacheDir == "" {
		return ucd(os.UserCacheDir)
	}
	return LocalCacheDir
}
