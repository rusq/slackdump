package source

import (
	"encoding/json"
	"io/fs"
)

func unmarshalOne[T any](fsys fs.FS, name string) (T, error) {
	var v T
	f, err := fsys.Open(name)
	if err != nil {
		return v, err
	}
	defer f.Close()
	if err := json.NewDecoder(f).Decode(&v); err != nil {
		return v, err
	}
	return v, nil
}

func unmarshal[T ~[]S, S any](fsys fs.FS, name string) (T, error) {
	f, err := fsys.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var v T
	if err := json.NewDecoder(f).Decode(&v); err != nil {
		return nil, err
	}
	return v, nil
}
