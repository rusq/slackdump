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
