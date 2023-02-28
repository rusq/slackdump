package dump

import (
	"strings"
	"text/template"

	"github.com/rusq/slackdump/v2/types"
)

// namer is a helper type to generate filenames for conversations.
type namer struct {
	t   *template.Template
	ext string
}

// newNamer returns a new namer.  It must be called with a valid template.
func newNamer(tmpl string, ext string) (namer, error) {
	t, err := template.New("name").Parse(tmpl)
	if err != nil {
		return namer{}, err
	}
	return namer{t: t, ext: ext}, nil
}

// Filename returns the filename for the given conversation.
func (n namer) Filename(conv *types.Conversation) string {
	var buf strings.Builder
	if err := n.t.Execute(&buf, conv); err != nil {
		panic(err)
	}
	return buf.String() + "." + n.ext
}
