package fixgen

import (
	"encoding/json"
	"math/rand"
	"strconv"
	"time"

	"github.com/rusq/slackdump/v2/internal/fixtures"
	"github.com/rusq/slackdump/v2/types"
)

func init() {
	// Seed the random number generator
	rand.Seed(time.Now().UnixNano())
}

// ReSeed reseeds the random number generator
func ReSeed(n int64) {
	rand.Seed(n)
}

// randString generates a random string of length n.
func randString(n int) string {
	var letter = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	b := make([]rune, n)
	for i := range b {
		b[i] = letter[rand.Intn(len(letter))]
	}
	return string(b)
}

// GenerateTestConversation generates a test conversation with a random name.
func GenerateTestConversation(name string, startDate time.Time, endDate time.Time, numMessages int) types.Conversation {
	var messages = make([]types.Message, numMessages)
	for i := 0; i < numMessages; i++ {
		messages[i] = GenerateTestMessage(startDate, endDate)
	}

	return types.Conversation{
		Messages: messages,
		Name:     name,
		ID:       randString(9),
	}
}

func GenerateTestMessage(startDate, endDate time.Time) types.Message {
	var message types.Message
	err := json.Unmarshal([]byte(fixtures.TestMessage), &message.Message)
	if err != nil {
		panic(err)
	}
	message.Timestamp = strconv.FormatInt(rand.Int63n(endDate.Unix()-startDate.Unix())+startDate.Unix(), 10)

	return message
}
