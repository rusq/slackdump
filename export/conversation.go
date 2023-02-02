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

// byDate sorts the messages by date and returns a map date->[]ExportMessage.
// userIdx should contain the users in the conversation for populating the
// required fields.  Threads are flattened.
func (Export) byDate(c *types.Conversation, userIdx structures.UserIndex) (messagesByDate, error) {
	msgsByDate := make(map[string][]*ExportMessage, 0)
	if err := flattenMsgs(msgsByDate, c.Messages, userIdx); err != nil {
		return nil, err
	}

	// sort messages by Time within each date.
	for date, messages := range msgsByDate {
		sort.Slice(msgsByDate[date], func(i, j int) bool {
			return messages[i].slackdumpTime.Before(messages[j].slackdumpTime)
		})
	}

	return msgsByDate, nil
}

type messagesByDate map[string][]*ExportMessage

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

// flattenMsgs takes the messages input, splits them by the date and
// populates the msgsByDate map.
func flattenMsgs(msgsByDate messagesByDate, messages []types.Message, usrIdx structures.UserIndex) error {
	for i := range messages {
		expMsg := newExportMessage(&messages[i], usrIdx)

		if len(messages[i].ThreadReplies) > 0 {
			// Recursive call:  are you ready, mr. stack?
			if err := flattenMsgs(msgsByDate, messages[i].ThreadReplies, usrIdx); err != nil {
				return fmt.Errorf("thread ID %s: %w", messages[i].Timestamp, err)
			}
		}

		formattedDt := expMsg.slackdumpTime.Format(dateFmt)
		msgsByDate[formattedDt] = append(msgsByDate[formattedDt], expMsg)
	}

	return nil
}
