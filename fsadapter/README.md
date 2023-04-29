# fsadapter - File System adapter

[![Go Reference](https://pkg.go.dev/badge/github.com/rusq/fsadapter.svg)](https://pkg.go.dev/github.com/rusq/fsadapter)

fsadapter is a wrapper for writing to directory or a ZIP file. 

There are currently 2 adapters:

- Directory
- ZIP

Each adapter exposes the following methods:

- Create(string) (io.WriteCloser, error)
- WriteFile(name string, data []byte, perm os.FileMode) error
- Close() error

It is meant to be a drop-in replacement for os.* functions for [Slackdump](https://github.com/rusq/slackdump).
