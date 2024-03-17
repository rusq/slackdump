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
	fxtrMsgNestedLists = `{
		"client_msg_id": "229a6d45-a202-4c1f-86bc-24bded55cc0a",
		"type": "message",
		"user": "UHSD97ZA5",
		"text": "Enumerated:\n1. First (1)\n2. Second(2)\n    a. Nested (2.a)\n    b. Nested (2.b)\n        ▪︎ Nexted bullet point\n            • Another nested bullet\n                a. Nested enumeration",
		"ts": "1710144832.176569",
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
			"block_id": "hqvjh",
			"elements": [
			  {
				"type": "rich_text_section",
				"elements": [
				  {
					"type": "text",
					"text": "Enumerated:\n"
				  }
				]
			  },
			  {
				"type": "rich_text_list",
				"elements": [
				  {
					"type": "rich_text_section",
					"elements": [
					  {
						"type": "text",
						"text": "First (1)"
					  }
					]
				  },
				  {
					"type": "rich_text_section",
					"elements": [
					  {
						"type": "text",
						"text": "Second(2)"
					  }
					]
				  }
				],
				"style": "ordered",
				"indent": 0
			  },
			  {
				"type": "rich_text_list",
				"elements": [
				  {
					"type": "rich_text_section",
					"elements": [
					  {
						"type": "text",
						"text": "Nested (2.a)"
					  }
					]
				  },
				  {
					"type": "rich_text_section",
					"elements": [
					  {
						"type": "text",
						"text": "Nested (2.b)"
					  }
					]
				  }
				],
				"style": "ordered",
				"indent": 1
			  },
			  {
				"type": "rich_text_list",
				"elements": [
				  {
					"type": "rich_text_section",
					"elements": [
					  {
						"type": "text",
						"text": "Nexted bullet point"
					  }
					]
				  }
				],
				"style": "bullet",
				"indent": 2
			  },
			  {
				"type": "rich_text_list",
				"elements": [
				  {
					"type": "rich_text_section",
					"elements": [
					  {
						"type": "text",
						"text": "Another nested bullet"
					  }
					]
				  }
				],
				"style": "bullet",
				"indent": 3
			  },
			  {
				"type": "rich_text_list",
				"elements": [
				  {
					"type": "rich_text_section",
					"elements": [
					  {
						"type": "text",
						"text": "Nested enumeration"
					  }
					]
				  }
				],
				"style": "ordered",
				"indent": 4
			  }
			]
		  }
		]
	  }`
)

// Attachments
const (
	fxtrAttYoutube = `{
		"fallback": "YouTube Video: Microsoft Account Takeover: Combination of Subdomain Takeovers and Open Redirection Vulnerabilities",
		"id": 1,
		"author_name": "VULLNERABILITY",
		"author_link": "https://www.youtube.com/channel/UClWkD38yogV4fRktm6Kb_2w",
		"title": "Microsoft Account Takeover: Combination of Subdomain Takeovers and Open Redirection Vulnerabilities",
		"title_link": "https://youtu.be/Jg3mkLm2K2g",
		"thumb_url": "https://i.ytimg.com/vi/Jg3mkLm2K2g/hqdefault.jpg",
		"service_name": "YouTube",
		"service_icon": "https://a.slack-edge.com/80588/img/unfurl_icons/youtube.png",
		"from_url": "https://youtu.be/Jg3mkLm2K2g",
		"original_url": "https://youtu.be/Jg3mkLm2K2g",
		"blocks": null
	  }`
	fxtrAttTwitter = `{
		"fallback": "\u003chttps://twitter.com/edwardodell|@edwardodell\u003e: NEVER LEAVE, NEVER PAY",
		"id": 1,
		"author_name": "Edward Odell",
		"author_subname": "@edwardodell",
		"author_link": "https://twitter.com/edwardodell/status/1591044196705927168",
		"author_icon": "https://pbs.twimg.com/profile_images/1590674641458167808/Z4vACFd0_normal.jpg",
		"text": "NEVER LEAVE, NEVER PAY",
		"image_url": "https://pbs.twimg.com/media/FhSGQsVXwAUpyU5.jpg",
		"service_name": "twitter",
		"from_url": "https://twitter.com/edwardodell/status/1591044196705927168?s=46\u0026amp;t=w-i11UUFTIWOtvEWpF2hpQ",
		"original_url": "https://twitter.com/edwardodell/status/1591044196705927168?s=46\u0026amp;t=w-i11UUFTIWOtvEWpF2hpQ",
		"blocks": null,
		"footer": "Twitter",
		"footer_icon": "https://a.slack-edge.com/80588/img/services/twitter_pixel_snapped_32.png",
		"ts": 1668169471
	  }`
	fxtrAttTwitterX = `{
			"fallback": "X (formerly Twitter): Elon Musk Junior :flag-ke: (@ElonMursq) on X",
			"id": 1,
			"title": "Elon Musk Junior :flag-ke: (@ElonMursq) on X",
			"title_link": "https://twitter.com/ElonMursq",
			"text": "Please help me reconnect with my dad @ElonMusk.\n\nMy BTC wallet address: 1FM6odFCta6gKwo2ib9jJ2JCEJgQixLoc2",
			"thumb_url": "https://pbs.twimg.com/profile_images/1767278440892289024/WXKK1Oa-_200x200.jpg",
			"service_name": "X (formerly Twitter)",
			"service_icon": "http://abs.twimg.com/favicons/twitter.3.ico",
			"from_url": "https://twitter.com/ElonMursq",
			"original_url": "https://twitter.com/ElonMursq",
			"blocks": null
		  }`
	fxtrAttNzHerald = `{
		"fallback": "NZ Herald: NZ Herald: Latest NZ news, plus World, Sport, Business and more - NZ Herald",
		"id": 3,
		"title": "NZ Herald: Latest NZ news, plus World, Sport, Business and more - NZ Herald",
		"title_link": "https://www.nzherald.co.nz/",
		"text": "Get the latest breaking news, analysis and opinion from NZ and around the world, including politics, business, sport, entertainment, travel and more, with NZ Herald",
		"thumb_url": "https://www.nzherald.co.nz/pf/resources/images/fallback-promo-image.png?d=744",
		"service_name": "NZ Herald",
		"service_icon": "https://www.nzherald.co.nz/pf/resources/images/favicons/apple-touch-icon-57x57-precomposed.png?d=744",
		"from_url": "https://www.nzherald.co.nz/",
		"original_url": "https://www.nzherald.co.nz/",
		"blocks": null
	  }`
	fxtrAttImage = `{
		"fallback": "1200x1200px image",
		"id": 2,
		"image_url": "https://pbs.twimg.com/media/FhTXW4bWAA4jqXk.jpg",
		"from_url": "https://twitter.com/rafaelshimunov/status/1591133819918114816?s=46\u0026amp;t=w-i11UUFTIWOtvEWpF2hpQ",
		"blocks": null
	  }`
	fxtrAttBBC = `{
		"fallback": "BBC News: Elon Musk: Judge blocks 'unfathomable' $56bn Tesla pay deal",
		"id": 1,
		"title": "Elon Musk: Judge blocks 'unfathomable' $56bn Tesla pay deal",
		"title_link": "https://www.bbc.co.uk/news/business-68150306",
		"text": "The lawsuit was filed by a shareholder who argued that it was an inappropriate overpayment.",
		"image_url": "https://ichef.bbci.co.uk/news/1024/branded_news/F474/production/_132508526_gettyimages-1963458442.jpg",
		"service_name": "BBC News",
		"service_icon": "https://www.bbc.co.uk/favicon.ico",
		"from_url": "https://www.bbc.co.uk/news/business-68150306",
		"original_url": "https://www.bbc.co.uk/news/business-68150306",
		"blocks": null
	  }`
)
