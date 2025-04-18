package source

import (
	"fmt"
	"strings"

	"github.com/rusq/slack"
)

// StorageType is the type of storage used for the files within the source.
type StorageType uint8

//go:generate stringer -type=StorageType -trimprefix=ST
const (
	// STnone is the storage type for no storage.
	STnone StorageType = iota
	// STstandard is the storage type for the standard file storage.
	STstandard
	// STmattermost is the storage type for Mattermost.
	STmattermost
	// STdump is the storage type for the dump format.
	STdump
	// STAvatar is the storage type for the avatar storage.
	STAvatar
)

// Set translates the string value into the ExportType, satisfies flag.Value
// interface.  It is based on the declarations generated by stringer.
//
// It is imperative that the stringer is generated prior to calling this
// function, if any new storage methods are added.
func (e *StorageType) Set(v string) error {
	v = strings.ToLower(v)
	for i := 0; i < len(_StorageType_index)-1; i++ {
		if strings.ToLower(_StorageType_name[_StorageType_index[i]:_StorageType_index[i+1]]) == v {
			*e = StorageType(i)
			return nil
		}
	}
	return fmt.Errorf("unknown format: %s", v)
}

// Func returns the "resolve path" function that returns the file path for the given
// channel and file.  It returns false if the storage type is not recognised.
func (e *StorageType) Func() (pathFn func(*slack.Channel, *slack.File) string, ok bool) {
	pathFn, ok = storageTypeFuncs[*e]
	return
}

// storageTypeFuncs is a map of storage types to functions that return the
// file path for the given channel and file.
var storageTypeFuncs = map[StorageType]func(_ *slack.Channel, f *slack.File) string{
	STmattermost: MattermostFilepath,
	STstandard:   StdFilepath,
	STdump:       DumpFilepath,
	STnone:       func(*slack.Channel, *slack.File) string { return "" },
}
