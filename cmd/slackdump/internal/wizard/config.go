package wizard

import (
	"errors"
)

// initFlags initializes flags based on the key-value pairs.
// Example:
//
//	var (
//		enterpriseMode bool
//		downloadFiles  bool
//	)
//
//	flags, err := initFlags(enterpriseMode, "enterprise", downloadFiles, "files")
//	if err != nil {
//		return err
//	}
func initFlags(keyval ...any) ([]string, error) {
	var flags []string
	if len(keyval)%2 != 0 {
		return flags, errors.New("initFlags: odd number of key-value pairs")
	}
	for i := 0; i < len(keyval); i += 2 {
		if keyval[i].(bool) {
			flags = append(flags, keyval[i+1].(string))
		}
	}
	return flags, nil
}
