package info

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/rusq/encio"

	"github.com/rusq/slackdump/v3/auth"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/internal/cache"
)

func CollectAuth(ctx context.Context, w io.Writer) error {
	// lg := logger.FromContext(ctx)
	fmt.Fprintln(os.Stderr, "To confirm the operation, please enter your OS password.")
	if err := osValidateUser(ctx, os.Stderr); err != nil {
		return err
	}
	m, err := cache.NewManager(cfg.CacheDir())
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
	f, err := encio.Open(filepath.Join(cfg.CacheDir(), fi.Name()))
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
