package format

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/rusq/slackdump/v2/types"
	"github.com/slack-go/slack"
)

type Converter interface {
	Conversation(ctx context.Context, w io.Writer, u []slack.User, conv *types.Conversation) error
	Channels(ctx context.Context, w io.Writer, u []slack.User, chans types.Channels) error
	Users(ctx context.Context, w io.Writer, u []slack.User) error
}

// Type is the converter type.
//
//go:generate stringer -type Type -trimprefix C format.go
type Type int

const (
	CUnknown Type = iota // Unknown converter type
	CText                // CText is the plain text converter
)

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
