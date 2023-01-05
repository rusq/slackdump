// Package format provides formatting fuctions for different output format
// types.
package format

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/rusq/slackdump/v2/internal/structures"
	"github.com/rusq/slackdump/v2/types"
	"github.com/slack-go/slack"
)

// Type is the converter type.
//
//go:generate stringer -type Type -trimprefix C format.go
type Type int

const (
	CUnknown Type = iota // Unknown converter type
	CText                // CText is the plain text converter
	CCSV                 // CCSV is the CSV converter
)

var AllTypes = []Type{CText, CCSV}

// Converter is a converter interface that each formatter must implement.
type Converter interface {
	Conversation(ctx context.Context, w io.Writer, u []slack.User, conv *types.Conversation) error
	Channels(ctx context.Context, w io.Writer, u []slack.User, chans types.Channels) error
	Users(ctx context.Context, w io.Writer, u []slack.User) error
}

type options struct {
	textOptions
	csvOptions
}

// Option is the converter option.
type Option func(*options)

var Converters = make(map[Type]func(opts ...Option) Converter)

func (e *Type) Set(v string) error {
	v = strings.ToLower(v)
	for i := 0; i < len(_Type_index)-1; i++ {
		if strings.ToLower(_Type_name[_Type_index[i]:_Type_index[i+1]]) == v {
			*e = Type(i)
			return nil
		}
	}
	return fmt.Errorf("unknown converter: %s", v)
}

// userReplacer returns a replacer that replaces all user IDs with their
// DisplayNames.
func userReplacer(userIdx structures.UserIndex) *strings.Replacer {
	if len(userIdx) == 0 {
		return strings.NewReplacer()
	}
	var replacements = make([]string, 0, len(userIdx)*2)
	for k := range userIdx {
		replacements = append(replacements, k, userIdx.DisplayName(k))
	}
	return strings.NewReplacer(replacements...)
}
