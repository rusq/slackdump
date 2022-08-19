package export

import (
	"fmt"
	"strings"
)

//go:generate go install golang.org/x/tools/cmd/stringer@latest

//go:generate stringer -type=ExportType -linecomment
type ExportType uint8

const (
	TStandard   ExportType = iota // Standard
	TMattermost                   // Mattermost
)

func (e *ExportType) Set(v string) error {
	v = strings.ToLower(v)
	switch v {
	default:
		return fmt.Errorf("unknown format: %s", v)
	case "standard":
		*e = TStandard
	case "mattermost":
		*e = TMattermost
	}
	return nil
}
