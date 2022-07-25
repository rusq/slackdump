package export

// Message transformations

import (
	"fmt"
	"sort"
	"time"

	"github.com/rusq/slackdump/v2/internal/structures"
	"github.com/rusq/slackdump/v2/types"
)

const dateFmt = "2006-01-02"

// byDate sorts the messages by date and returns a map date->[]slack.Message.
// users should contain the users in the conversation for population of required
// fields.
// Threads are flattened.
func (Export) byDate(c *types.Conversation, userIdx structures.UserIndex) (map[string][]ExportMessage, error) {
	msgsByDate := make(map[string][]ExportMessage)
	if err := populateMsgs(msgsByDate, c.Messages, userIdx); err != nil {
		return nil, err
	}

	// sort messages by Time within each date.
	for date, messages := range msgsByDate {
		sort.Slice(msgsByDate[date], func(i, j int) bool {
			return messages[i].Time().Before(messages[j].Time())
		})
	}

	return msgsByDate, nil
}

type messagesByDate map[string][]ExportMessage

// validate checks if mbd keys are valid dates.
func (mbd messagesByDate) validate() error {
	for k := range mbd {
		_, err := time.Parse(dateFmt, k)
		if err != nil {
			return fmt.Errorf("validation failed for %q: %w", k, err)
		}
	}
	return nil
}

// populateMsgs takes the messages input, splits them by the date and
// populates the msgsByDate map.
func populateMsgs(msgsByDate messagesByDate, messages []types.Message, usrIdx structures.UserIndex) error {
	for _, msg := range messages {
		expMsg := newExportMessage(&msg, usrIdx)

		if len(msg.ThreadReplies) > 0 {
			// Recursive call:  are you ready, mr. stack?
			if err := populateMsgs(msgsByDate, msg.ThreadReplies, usrIdx); err != nil {
				return fmt.Errorf("thread ID %s: %w", msg.Timestamp, err)
			}
		}

		dt, err := msg.Datetime()
		if err != nil {
			return fmt.Errorf("updateDateMsgs: unable to parse message timestamp (%s): %w", msg.Timestamp, err)
		}

		formattedDt := dt.Format(dateFmt)
		msgsByDate[formattedDt] = append(msgsByDate[formattedDt], *expMsg)
	}

	return nil
}
