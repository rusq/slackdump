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
package cache

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// makeCacheFilename converts filename.ext to filename-suffix.ext.
func makeCacheFilename(cacheDir, filename, suffix string) string {
	ne := filenameSplit(filename)
	return filepath.Join(cacheDir, filenameJoin(nameExt{ne[0] + "-" + suffix, ne[1]}))
}

// nameExt is a pair of filename and extension.
type nameExt [2]string

// filenameSplit splits the "path/to/filename.ext" into
// nameExt{"path/to/filename", ".ext"}.
func filenameSplit(filename string) nameExt {
	ext := filepath.Ext(filename)
	name := filename[:len(filename)-len(ext)]
	return nameExt{name, ext}
}

// filenameJoin combines nameExt{"path/to/filename", ".ext"} to "path/to/filename.ext".
func filenameJoin(split nameExt) string {
	return split[0] + split[1]
}

// checkCacheFile checks the cache file to see if it is valid.
// The file is considered valid if it exists and is not older than maxAge.
// If the file does not exist, this function returns an error.
// If the file is older than maxAge, this function also returns an error.
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

var (
	ErrEmpty   = errors.New("empty cache file")
	ErrExpired = errors.New("cache expired")
)

// validateCache tests whether the provided file info meets the requirements
// for a valid cache file. It returns an error if the file does not meet the
// requirements.
func validateCache(fi os.FileInfo, maxAge time.Duration) error {
	if fi.IsDir() {
		return errors.New("cache file is a directory")
	}
	if fi.Size() == 0 {
		return ErrEmpty
	}
	if time.Since(fi.ModTime()) > maxAge {
		return ErrExpired
	}
	return nil
}

func writeSlice[T any](w io.Writer, tt []T) error {
	for _, t := range tt {
		if err := json.NewEncoder(w).Encode(t); err != nil {
			return fmt.Errorf("failed to encode data: %w", err)
		}
	}
	return nil
}

// save saves the users to a file, naming the file based on the filename
// and the suffix. The file will be saved in the cache directory.
func save[T any](cacheDir, filename string, suffix string, uu []T, co createOpener) error {
	filename = makeCacheFilename(cacheDir, filename, suffix)
	f, err := co.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filename, err)
	}
	defer f.Close()

	if err := writeSlice(f, uu); err != nil {
		return fmt.Errorf("file: %s, error: %w", filename, err)
	}
	return nil
}

// read reads the data from the reader r until it reaches the EOF and returns
// it as a slice of T.
func read[T any](r io.Reader) ([]T, error) {
	dec := json.NewDecoder(r)
	tt := make([]T, 0, 500) // 500 T. reasonable?
	for {
		var t T
		if err := dec.Decode(&t); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		tt = append(tt, t)
	}
	return tt, nil
}

// load loads the data from the file in the cache directory, and returns
// the data as a slice of T.
func load[T any](cacheDir, filename, suffix string, maxAge time.Duration, co createOpener) ([]T, error) {
	filename = makeCacheFilename(cacheDir, filename, suffix)

	if err := checkCacheFile(filename, maxAge); err != nil {
		return nil, fmt.Errorf("%s: %w", filename, err)
	}

	f, err := co.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", filename, err)
	}
	defer f.Close()

	tt, err := read[T](f)
	if err != nil {
		return nil, fmt.Errorf("failed to decode data from %s: %w", filename, err)
	}

	return tt, nil
}
