package cache

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/rusq/encio"
	"github.com/slack-go/slack"

	"github.com/rusq/slackdump/v2/types"
)

// loadUsers tries to load the users from the file
func loadUsers(cacheDir, filename string, suffix string, maxAge time.Duration) (types.Users, error) {
	filename = makeCacheFilename(cacheDir, filename, suffix)

	if err := checkCacheFile(filename, maxAge); err != nil {
		return nil, err
	}

	f, err := encio.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", filename, err)
	}
	defer f.Close()

	uu, err := ReadUsers(f)
	if err != nil {
		return nil, fmt.Errorf("failed to decode users from %s: %w", filename, err)
	}

	return uu, nil
}

// ReadUsers reads users from JSONL data in Reader r.
func ReadUsers(r io.Reader) (types.Users, error) {
	dec := json.NewDecoder(r)
	var uu = make(types.Users, 0, 500) // 500 users. reasonable?
	for {
		var u slack.User
		if err := dec.Decode(&u); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		uu = append(uu, u)
	}
	return uu, nil
}

func saveUsers(cacheDir, filename string, suffix string, uu types.Users) error {
	filename = makeCacheFilename(cacheDir, filename, suffix)

	f, err := encio.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filename, err)
	}
	defer f.Close()

	if err := WriteUsers(f, uu); err != nil {
		return fmt.Errorf("file: %s, error: %w", filename, err)
	}
	return nil
}

func WriteUsers(w io.Writer, uu types.Users) error {
	enc := json.NewEncoder(w)
	for _, u := range uu {
		if err := enc.Encode(u); err != nil {
			return fmt.Errorf("failed to encode data: %w", err)
		}
	}
	return nil
}

// makeCacheFilename converts filename.ext to filename-suffix.ext.
func makeCacheFilename(cacheDir, filename, suffix string) string {
	ne := filenameSplit(filename)
	return filepath.Join(cacheDir, filenameJoin(nameExt{ne[0] + "-" + suffix, ne[1]}))
}

type nameExt [2]string

// filenameSplit splits the "path/to/filename.ext" into nameExt{"path/to/filename", ".ext"}
func filenameSplit(filename string) nameExt {
	ext := filepath.Ext(filename)
	name := filename[:len(filename)-len(ext)]
	return nameExt{name, ext}
}

// filenameJoin combines nameExt{"path/to/filename", ".ext"} to "path/to/filename.ext".
func filenameJoin(split nameExt) string {
	return split[0] + split[1]
}

func checkCacheFile(filename string, maxAge time.Duration) error {
	if filename == "" {
		return errors.New("no cache filename")
	}
	fi, err := os.Stat(filename)
	if err != nil {
		return err
	}

	return validateCache(fi, maxAge)
}

func validateCache(fi os.FileInfo, maxAge time.Duration) error {
	if fi.IsDir() {
		return errors.New("cache file is a directory")
	}
	if fi.Size() == 0 {
		return errors.New("empty cache file")
	}
	if time.Since(fi.ModTime()) > maxAge {
		return errors.New("cache expired")
	}
	return nil
}
