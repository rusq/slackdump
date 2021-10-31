package slackdump

import (
	"fmt"
	"io"
	"log"
	"sort"

	"github.com/slack-go/slack"
)

// SlackDumper stores basic session parameters.
type SlackDumper struct {
	api *slack.Client

	// Users contains the list of users and populated on NewSlackDumper
	Users     Users                  `json:"users"`
	Channels  []slack.Channel        `json:"channels"`
	UserForID map[string]*slack.User `json:"-"`
}

var allChanTypes = []string{"mpim", "im", "public_channel", "private_channel"}

// Reporter is an interface defining output functions
type Reporter interface {
	ToText(w io.Writer) error
}

// New creates new client and fills the structure with data
func New(token string, cookie string) (*SlackDumper, error) {
	var err error
	sd := &SlackDumper{
		api: slack.New(token, slack.OptionCookie(cookie)),
	}
	errC := make(chan error, 1)

	var chans *Channels

	go func() {
		var err error
		chanTypes := allChanTypes
		chans, err = sd.getChannels(chanTypes)
		if err != nil {
			errC <- err
		}
		close(errC)
	}()

	if _, err := sd.GetUsers(); err != nil {
		return nil, fmt.Errorf("error fetching users: %s", err)
	}

	if err = <-errC; err != nil {
		return nil, fmt.Errorf("error fetching channels: %s", err)
	}

	sd.Channels = chans.Channels

	return sd, nil
}

// IsDeletedUser checks if the user is deleted and returns appropriate value
func (sd *SlackDumper) IsDeletedUser(id string) bool {
	thisUser, ok := sd.UserForID[id]
	if !ok {
		return false
	}
	return thisUser.Deleted
}

// DumpMessages fetches messages from the specified channel
func (sd *SlackDumper) DumpMessages(channelID string, dumpFiles bool) (*Messages, error) {

	params := &slack.GetConversationHistoryParameters{
		ChannelID: channelID,
	}

	filesC := make(chan *slack.File, 20)
	done := make(chan bool)
	errC := make(chan error, 1)

	if dumpFiles {
		go func() {
			errC <- sd.fileDownloader(channelID, filesC, done)
		}()
	}

	throttle := getThrottler(slackTier3)
	allMessages := make([]slack.Message, 0, 2000)
LOOP:
	for i := 1; ; i++ {
		select {
		case err := <-errC:
			// stop the goroutine gracefully if it's running
			close(filesC)
			<-done
			return nil, err
		case <-throttle:
			hist, err := sd.api.GetConversationHistory(params)
			if err != nil {
				return nil, err
			}

			allMessages = append(allMessages, hist.Messages...)
			if dumpFiles {
				// place files in download queue
				chunk := sd.getFilesFromChunk(hist.Messages)
				for i := range chunk {
					filesC <- &chunk[i]
				}
			}

			log.Printf("request #%d, fetched: %d, total: %d\n",
				i, len(hist.Messages), len(allMessages))

			if !hist.HasMore {
				break LOOP
			}

			params.Cursor = hist.ResponseMetaData.NextCursor
		}
	}

	sort.Slice(allMessages, func(i, j int) bool {
		return allMessages[i].Timestamp < allMessages[j].Timestamp
	})

	if dumpFiles {
		close(filesC)
		<-done
	}

	return &Messages{Messages: allMessages, ChannelID: channelID, SD: sd}, nil
}

// UpdateUserMap updates user[id]->*User mapping from the current Users slice.
func (sd *SlackDumper) UpdateUserMap() error {
	if sd.Users.Len() == 0 {
		return fmt.Errorf("no users loaded")
	}
	sd.UserForID = sd.Users.MakeUserIDIndex()
	return nil
}
