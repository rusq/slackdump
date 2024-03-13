package renderer

import (
	"encoding/json"
	"testing"

	"github.com/rusq/slack"
)

func load(t *testing.T, s string) *slack.Message {
	t.Helper()
	var m slack.Message
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		t.Fatal(err)
	}
	return &m
}

const (
	fxtrRtseText = `
{
	"client_msg_id": "749a28ae-d333-49de-9961-d765531b447e",
	"type": "message",
	"user": "UHSD97ZA5",
	"text": "New message",
	"ts": "1710141984.243839",
	"thread_ts": "1710141984.243839",
	"last_read": "1710142198.963419",
	"subscribed": true,
	"reply_count": 5,
	"reply_users": [
		"UHSD97ZA5"
	],
	"latest_reply": "1710142198.963419",
	"team": "THY5HTZ8U",
	"replace_original": false,
	"delete_original": false,
	"metadata": {
		"event_type": "",
		"event_payload": null
	},
	"blocks": [
		{
		"type": "rich_text",
		"block_id": "P/qs1",
		"elements": [
			{
			"type": "rich_text_section",
			"elements": [
				{
				"type": "text",
				"text": "New message"
				}
			]
			}
		]
		}
	]
	}
`
)
