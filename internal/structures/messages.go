package structures

import (
	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v3/internal/fasttime"
)

type Messages []slack.Message

func (m Messages) Len() int { return len(m) }
func (m Messages) Less(i, j int) bool {
	tsi, err := fasttime.TS2int(m[i].Timestamp)
	if err != nil {
		return false
	}
	tsj, err := fasttime.TS2int(m[j].Timestamp)
	if err != nil {
		return false
	}
	return tsi < tsj
}
func (m Messages) Swap(i, j int) { m[i], m[j] = m[j], m[i] }
