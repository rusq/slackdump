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

package obfuscate

import (
	"compress/gzip"
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rusq/slackdump/v4/internal/chunk"
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
	rand.New(rand.NewSource(opts.seed))

	lg := slog.Default()
	files, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	rng := rand.New(rand.NewSource(opts.seed))
	var obf = newObfuscator(rng)

	var once sync.Once
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		if !strings.HasSuffix(f.Name(), ".json.gz") {
			lg.DebugContext(ctx, "skipping", "filename", f.Name())
		}
		lg.DebugContext(ctx, "processing %s", "filename", f.Name())
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
