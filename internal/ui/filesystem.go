package ui

import (
	"errors"
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
)

type fileSelectorOpt struct {
	defaultFilename string // if set, the empty filename will be replaced to this value
	mustExist       bool
}

func WithDefaultFilename(s string) Option {
	return func(io *inputOptions) {
		io.fileSelectorOpt.defaultFilename = s
	}
}

func WithMustExist(b bool) Option {
	return func(io *inputOptions) {
		io.mustExist = b
	}
}

func FileSelector(msg, descr string, opt ...Option) (string, error) {
	var opts = defaultOpts().apply(opt...)

	var resp struct {
		Filename string
	}
	q := huh.NewInput().
		Title(msg).
		Description(descr).
		Value(&resp.Filename).
		Validate(func(ans string) error {
			filename := ans
			if filename == "" {
				if opts.defaultFilename == "" {
					return errors.New("empty filename")
				} else {
					if !opts.mustExist {
						return nil
					} else {
						return checkExists(opts.defaultFilename)
					}
				}
			}
			if opts.mustExist {
				return checkExists(filename)
			}
			return nil
		})

	for {
		if err := q.Run(); err != nil {
			return "", err
		}
		if resp.Filename == "" && opts.defaultFilename != "" {
			resp.Filename = opts.defaultFilename
		}
		if opts.mustExist {
			break
		}
		if _, err := os.Stat(resp.Filename); err != nil {
			break
		}
		overwrite, err := Confirm(fmt.Sprintf("File %q exists. Overwrite?", resp.Filename), false, opt...)
		if err != nil {
			return "", err
		}
		if overwrite {
			break
		}
	}
	return resp.Filename, nil
}

func checkExists(filename string) error {
	if _, err := os.Stat(filename); err != nil {
		if os.IsNotExist(err) {
			return errors.New("file must exist")
		} else {
			return err
		}
	}
	return nil
}
