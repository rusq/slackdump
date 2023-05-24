package obfuscate

import (
	"compress/gzip"
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rusq/dlog"
	"github.com/rusq/slackdump/v2/internal/chunk"
)

// DoDir obfuscates all files in the directory src, placing obfuscated
// files in the directory trg.
func DoDir(ctx context.Context, src string, trg string, options ...Option) error {
	var opts = doOpts{
		seed: time.Now().UnixNano(),
	}
	for _, optFn := range options {
		optFn(&opts)
	}
	rand.Seed(opts.seed)

	lg := dlog.FromContext(ctx)
	files, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	var obf = newObfuscator()

	var once sync.Once
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		if !strings.HasSuffix(f.Name(), ".json.gz") {
			lg.Printf("skipping %s", f.Name())
		}
		lg.Debugf("processing %s", f.Name())
		once.Do(func() {
			err = os.MkdirAll(trg, 0755)
		})
		if err != nil {
			return err
		}
		if err := doFile(ctx, obf, trg, filepath.Join(src, f.Name())); err != nil {
			return fmt.Errorf("error on file %s: %w", f.Name(), err)
		}
	}
	return nil
}

// doFile obfuscates the file src, placing the obfuscated file in trg.
func doFile(ctx context.Context, obf obfuscator, trgDir string, src string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	in, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer in.Close()

	fileid := chunk.FileID(strings.TrimSuffix(filepath.Base(src), ".json.gz"))
	switch fileid {
	case "users", "channels", "workspace":
	default:
		// channel or thread
		channel, thread := fileid.Split()
		fileid = chunk.ToFileID(obf.ChannelID(channel), thread, len(thread) > 0) // export won't have a thread
	}
	w, err := os.Create(filepath.Join(trgDir, string(fileid)+".json.gz"))
	if err != nil {
		return err
	}
	defer w.Close()
	out := gzip.NewWriter(w)
	defer out.Close()

	return obfuscate(ctx, obf, out, in)
}
