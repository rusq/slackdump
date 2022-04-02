package fixtures

const (
	SimpleMessageJSON = `    {
        "client_msg_id": "c6cdfb3a-59d6-4198-9800-cc74bcdc0b7d",
        "type": "message",
        "text": "Test message with Html chars &lt; &gt;",
        "user": "UHSD97ZA5",
        "ts": "1645095505.023899",
        "team": "THY5HTZ8U",
        "blocks": [
            {
                "type": "rich_text",
                "block_id": "SkX",
                "elements": [
                    {
                        "type": "rich_text_section",
                        "elements": [
                            {
                                "type": "text",
                                "text": "Test message with Html chars < >"
                            }
                        ]
                    }
                ]
            }
        ]
    }
	`
	ThreadMessageJSON = `    {
	"client_msg_id": "676e1cbb-15fe-45e9-b7f2-32a8764fe560",
	"type": "message",
	"user": "UHSD97ZA5",
	"text": "This ~is a~  Rich Text message test.",
	"ts": "1577694990.000400",
	"thread_ts": "1577694990.000400",
	"last_read": "1648633700.407619",
	"subscribed": true,
	"reply_count": 3,
	"latest_reply": "1648633700.407619",
	"team": "THY5HTZ8U",
	"replace_original": false,
	"delete_original": false,
	"blocks": [
	  {
		"type": "rich_text",
		"block_id": "r+rDn",
		"elements": [
		  {
			"type": "rich_text_section",
			"elements": [
			  {
				"type": "text",
				"text": "This "
			  },
			  {
				"type": "text",
				"text": "is a ",
				"style": {
				  "strike": true
				}
			  },
			  {
				"type": "text",
				"text": " Rich Text message test."
			  }
			]
		  }
		]
	  }
	],
	"slackdump_thread_replies": [
	  {
		"client_msg_id": "5c905e3c-6a6f-40ee-9b0f-c53dc8e74885",
		"type": "message",
		"user": "UHSD97ZA5",
		"text": "Test thread reply",
		"ts": "1638784588.000100",
		"thread_ts": "1577694990.000400",
		"parent_user_id": "UHSD97ZA5",
		"team": "THY5HTZ8U",
		"replace_original": false,
		"delete_original": false,
		"blocks": [
		  {
			"type": "rich_text",
			"block_id": "iIw",
			"elements": [
			  {
				"type": "rich_text_section",
				"elements": [
				  {
					"type": "text",
					"text": "Test thread reply"
				  }
				]
			  }
			]
		  }
		]
	  },
	  {
		"client_msg_id": "bd1ce8e1-7646-48a3-abd7-ec19c094e6f9",
		"type": "message",
		"user": "UHSD97ZA5",
		"text": "test image",
		"ts": "1638784627.000300",
		"thread_ts": "1577694990.000400",
		"parent_user_id": "UHSD97ZA5",
		"files": [
		  {
			"id": "F02PM6A1AUA",
			"created": 1638784624,
			"timestamp": 1638784624,
			"name": "Chevy.jpg",
			"title": "Chevy.jpg",
			"mimetype": "image/jpeg",
			"image_exif_rotation": 0,
			"filetype": "jpg",
			"pretty_type": "JPEG",
			"user": "UHSD97ZA5",
			"mode": "hosted",
			"editable": false,
			"is_external": false,
			"external_type": "",
			"size": 359002,
			"url": "",
			"url_download": "",
			"url_private": "https://files.slack.com/files-pri/THY5HTZ8U-F02PM6A1AUA/chevy.jpg",
			"url_private_download": "https://files.slack.com/files-pri/THY5HTZ8U-F02PM6A1AUA/download/chevy.jpg",
			"original_h": 1080,
			"original_w": 1920,
			"thumb_64": "https://files.slack.com/files-tmb/THY5HTZ8U-F02PM6A1AUA-d1fdb12d3e/chevy_64.jpg",
			"thumb_80": "https://files.slack.com/files-tmb/THY5HTZ8U-F02PM6A1AUA-d1fdb12d3e/chevy_80.jpg",
			"thumb_160": "https://files.slack.com/files-tmb/THY5HTZ8U-F02PM6A1AUA-d1fdb12d3e/chevy_160.jpg",
			"thumb_360": "https://files.slack.com/files-tmb/THY5HTZ8U-F02PM6A1AUA-d1fdb12d3e/chevy_360.jpg",
			"thumb_360_gif": "",
			"thumb_360_w": 360,
			"thumb_360_h": 203,
			"thumb_480": "https://files.slack.com/files-tmb/THY5HTZ8U-F02PM6A1AUA-d1fdb12d3e/chevy_480.jpg",
			"thumb_480_w": 480,
			"thumb_480_h": 270,
			"thumb_720": "https://files.slack.com/files-tmb/THY5HTZ8U-F02PM6A1AUA-d1fdb12d3e/chevy_720.jpg",
			"thumb_720_w": 720,
			"thumb_720_h": 405,
			"thumb_960": "https://files.slack.com/files-tmb/THY5HTZ8U-F02PM6A1AUA-d1fdb12d3e/chevy_960.jpg",
			"thumb_960_w": 960,
			"thumb_960_h": 540,
			"thumb_1024": "https://files.slack.com/files-tmb/THY5HTZ8U-F02PM6A1AUA-d1fdb12d3e/chevy_1024.jpg",
			"thumb_1024_w": 1024,
			"thumb_1024_h": 576,
			"permalink": "https://ora600.slack.com/files/UHSD97ZA5/F02PM6A1AUA/chevy.jpg",
			"permalink_public": "https://slack-files.com/THY5HTZ8U-F02PM6A1AUA-ea648a3dee",
			"edit_link": "",
			"preview": "",
			"preview_highlight": "",
			"lines": 0,
			"lines_more": 0,
			"is_public": true,
			"public_url_shared": false,
			"channels": null,
			"groups": null,
			"ims": null,
			"initial_comment": {},
			"comments_count": 0,
			"num_stars": 0,
			"is_starred": false,
			"shares": {
			  "public": null,
			  "private": null
			}
		  }
		],
		"replace_original": false,
		"delete_original": false,
		"blocks": [
		  {
			"type": "rich_text",
			"block_id": "sh6oC",
			"elements": [
			  {
				"type": "rich_text_section",
				"elements": [
				  {
					"type": "text",
					"text": "test image"
				  }
				]
			  }
			]
		  }
		]
	  },
	  {
		"client_msg_id": "5dcb1332-8bf7-4ce6-b884-ae6d0f4aac35",
		"type": "message",
		"user": "UHSD97ZA5",
		"text": "Hello from 2022",
		"ts": "1648633700.407619",
		"thread_ts": "1577694990.000400",
		"parent_user_id": "UHSD97ZA5",
		"team": "THY5HTZ8U",
		"replace_original": false,
		"delete_original": false,
		"blocks": [
		  {
			"type": "rich_text",
			"block_id": "H3x",
			"elements": [
			  {
				"type": "rich_text_section",
				"elements": [
				  {
					"type": "text",
					"text": "Hello from 2022"
				  }
				]
			  }
			]
		  }
		]
	  }
	]
  }
`

	BotMessageThreadParentJSON = `{
        "bot_id": "B035AAUBQ3S",
        "type": "message",
        "text": "This content can't be displayed.",
        "user": "U034HM0P7RB",
        "ts": "1648085300.726649",
        "team": "THY5HTZ8U",
        "bot_profile": {
            "id": "B035AAUBQ3S",
            "app_id": "A035A9SCZDE",
            "name": "Data Test Communiqués",
            "icons": {
                "image_36": "https:\/\/a.slack-edge.com\/80588\/img\/plugins\/app\/bot_36.png",
                "image_48": "https:\/\/a.slack-edge.com\/80588\/img\/plugins\/app\/bot_48.png",
                "image_72": "https:\/\/a.slack-edge.com\/80588\/img\/plugins\/app\/service_72.png"
            },
            "deleted": false,
            "updated": 1645745563,
            "team_id": "THY5HTZ8U"
        },
        "blocks": [
            {
                "type": "header",
                "block_id": "C5wf8",
                "text": {
                    "type": "plain_text",
                    "text": "All Checks: :large_green_square: PASS",
                    "emoji": true
                }
            },
            {
                "type": "divider",
                "block_id": "D8R"
            },
            {
                "type": "section",
                "block_id": "MH4e",
                "fields": [
                    {
                        "type": "mrkdwn",
                        "text": "*AAAA*: PASSED",
                        "verbatim": false
                    },
                    {
                        "type": "mrkdwn",
                        "text": "*BBBB*: FAILED",
                        "verbatim": false
                    },
                    {
                        "type": "mrkdwn",
                        "text": "*CCCC*: ERROR",
                        "verbatim": false
                    },
                    {
                        "type": "mrkdwn",
                        "text": "*CCCC*: UNKNOWN",
                        "verbatim": false
                    }
                ]
            },
            {
                "type": "context",
                "block_id": "xx",
                "elements": [
                    {
                        "type": "plain_text",
                        "text": "Started  at: 2022-03-24 01:28:19.7703933 +0000 UTC m=+0.001169901",
                        "emoji": true
                    },
                    {
                        "type": "plain_text",
                        "text": "Finished at: 2022-03-24 01:28:19.7704092 +0000 UTC m=+0.001185901",
                        "emoji": true
                    }
                ]
            }
        ],
        "thread_ts": "1648085300.726649",
        "reply_count": 1,
        "reply_users_count": 1,
        "latest_reply": "1648085301.269949",
        "reply_users": [
            "U034HM0P7RB"
        ],
        "replies": [
            {
                "user": "U034HM0P7RB",
                "ts": "1648085301.269949"
            }
        ],
        "is_locked": false,
        "subscribed": false
    }
`
	BotMessageThreadChildJSON = `{
        "type": "message",
        "text": "",
        "files": [
            {
                "id": "F0394BFKYL8",
                "created": 1648085301,
                "timestamp": 1648085301,
                "name": "report-20220324012819.txt",
                "title": "report-20220324012819.txt",
                "mimetype": "text\/plain",
                "filetype": "text",
                "pretty_type": "Plain Text",
                "user": "U034HM0P7RB",
                "editable": true,
                "size": 16,
                "mode": "snippet",
                "is_external": false,
                "external_type": "",
                "is_public": true,
                "public_url_shared": false,
                "display_as_bot": false,
                "username": "",
                "url_private": "https:\/\/files.slack.com\/files-pri\/THY5HTZ8U-F0394BFKYL8\/report-20220324012819.txt?t=xoxe-610187951300-3333679776258-3327029718998-172d87a599d073b787115e53b775061b",
                "url_private_download": "https:\/\/files.slack.com\/files-pri\/THY5HTZ8U-F0394BFKYL8\/download\/report-20220324012819.txt?t=xoxe-610187951300-3333679776258-3327029718998-172d87a599d073b787115e53b775061b",
                "permalink": "https:\/\/ora600.slack.com\/files\/U034HM0P7RB\/F0394BFKYL8\/report-20220324012819.txt",
                "permalink_public": "https:\/\/slack-files.com\/THY5HTZ8U-F0394BFKYL8-7ce1343fe2",
                "edit_link": "https:\/\/ora600.slack.com\/files\/U034HM0P7RB\/F0394BFKYL8\/report-20220324012819.txt\/edit",
                "is_starred": false,
                "has_rich_preview": false
            }
        ],
        "upload": true,
        "user": "U034HM0P7RB",
        "display_as_bot": false,
        "ts": "1648085301.269949",
        "thread_ts": "1648085300.726649",
        "parent_user_id": "U034HM0P7RB"
    }`
)
