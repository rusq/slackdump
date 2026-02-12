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

//go:build !windows

package info

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
)

func osValidateUser(ctx context.Context, w io.Writer) error {
	cmd := exec.CommandContext(ctx, "sudo", "-v")
	cmd.Stdin = os.Stdin
	cmd.Stdout = w
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}
	return nil
}
