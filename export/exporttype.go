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
	case strings.ToLower(TStandard.String()):
		*e = TStandard
	case strings.ToLower(TMattermost.String()):
		*e = TMattermost
	}
	return nil
}
