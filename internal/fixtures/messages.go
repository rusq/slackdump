package fixtures

import _ "embed"

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

	ThreadMessage1JSON = `    {
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

	ThreadedExportedMessage1JSON = `{
			"client_msg_id": "676e1cbb-15fe-45e9-b7f2-32a8764fe560",
			"type": "message",
			"text": "This ~is a~  Rich Text message test.",
			"user": "UHSD97ZA5",
			"ts": "1577694990.000400",
			"team": "THY5HTZ8U",
			"user_team": "THY5HTZ8U",
			"source_team": "THY5HTZ8U",
			"user_profile": {
				"image_72": "https:\/\/secure.gravatar.com\/avatar\/41eca2328d6510133f47ffceae7b912a.jpg?s=72&d=https%3A%2F%2Fa.slack-edge.com%2Fdf10d%2Fimg%2Favatars%2Fava_0022-72.png",
				"first_name": "Jane",
				"real_name": "Jane Doe",
				"display_name": "",
				"team": "THY5HTZ8U",
				"name": "janed",
				"is_restricted": false,
				"is_ultra_restricted": false
			},
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
			"thread_ts": "1577694990.000400",
			"reply_count": 3,
			"reply_users_count": 1,
			"latest_reply": "1648633700.407619",
			"reply_users": [
				"UHSD97ZA5"
			],
			"replies": [
				{
					"user": "UHSD97ZA5",
					"ts": "1638784588.000100"
				},
				{
					"user": "UHSD97ZA5",
					"ts": "1638784627.000300"
				},
				{
					"user": "UHSD97ZA5",
					"ts": "1648633700.407619"
				}
			],
			"is_locked": false,
			"subscribed": true,
			"last_read": "1648633700.407619"
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

	TestChannelEveryoneMessagesNativeExport = `[
		{
			"type": "message",
			"subtype": "channel_join",
			"ts": "1555493778.000200",
			"user": "UHSD97ZA5",
			"text": "<@UHSD97ZA5> has joined the channel"
		},
    {
        "type": "message",
        "subtype": "channel_join",
        "ts": "1563609394.000200",
        "user": "ULLLZ6SAH",
        "text": "<@ULLLZ6SAH> has joined the channel"
    },
    {
        "client_msg_id": "6A910F66-48E3-450B-973C-7C0F3AFBE282",
        "type": "message",
        "text": "Fjdj",
        "user": "UHSD97ZA5",
        "ts": "1563609673.000400",
        "blocks": [
            {
                "type": "rich_text",
                "block_id": "klx4M",
                "elements": [
                    {
                        "type": "rich_text_section",
                        "elements": [
                            {
                                "type": "text",
                                "text": "Fjdj"
                            }
                        ]
                    }
                ]
            }
        ],
        "team": "THY5HTZ8U",
        "user_team": "THY5HTZ8U",
        "source_team": "THY5HTZ8U",
        "user_profile": {
            "avatar_hash": "g1eca2328d65",
            "image_72": "https:\/\/secure.gravatar.com\/avatar\/41eca2328d6510133f47ffceae7b912a.jpg?s=72&d=https%3A%2F%2Fa.slack-edge.com%2Fdf10d%2Fimg%2Favatars%2Fava_0022-72.png",
            "first_name": "Rustam",
            "real_name": "Rustam Gilyazov",
            "display_name": "",
            "team": "THY5HTZ8U",
            "name": "gilyazov",
            "is_restricted": false,
            "is_ultra_restricted": false
        }
    },
		{
			"type": "message",
			"text": "Test message with a file.",
			"files": [
				{
					"id": "F02PM6A1AUA",
					"mode": "hidden_by_limit"
				}
			],
			"upload": false,
			"user": "UHSD97ZA5",
			"display_as_bot": false,
			"ts": "1658222446.866019",
			"blocks": [
				{
					"type": "rich_text",
					"block_id": "6Ak",
					"elements": [
						{
							"type": "rich_text_section",
							"elements": [
								{
									"type": "text",
									"text": "Test message with a file."
								}
							]
						}
					]
				}
			],
			"client_msg_id": "c5f20b81-059d-486c-a1a5-6144eb59e15c"
		},
		{
			"type": "message",
			"text": "",
			"files": [
				{
					"id": "F03Q1KNGE3C",
					"mode": "hidden_by_limit"
				}
			],
			"upload": false,
			"user": "UHSD97ZA5",
			"display_as_bot": false,
			"ts": "1658222457.282419",
			"client_msg_id": "9de354dd-fe35-43d4-9992-1cfb4e475f8e"
		},
		{
			"type": "message",
			"text": "",
			"files": [
				{
					"id": "F03Q1KPCADQ",
					"mode": "hidden_by_limit"
				}
			],
			"upload": false,
			"user": "UHSD97ZA5",
			"display_as_bot": false,
			"ts": "1658222467.643489",
			"client_msg_id": "67a1d92c-ffea-420f-a597-f4f43f121d83"
		},
		{
			"type": "message",
			"text": "",
			"files": [
				{
					"id": "F03PYM54C2H",
					"mode": "hidden_by_limit"
				}
			],
			"upload": false,
			"user": "UHSD97ZA5",
			"display_as_bot": false,
			"ts": "1658222478.048689",
			"client_msg_id": "4539ec8b-4dfb-4e48-ab8e-eaf94a5ef9dd"
		},
		{
			"client_msg_id": "44b5c5ae-c637-4673-82d2-2c8c2fc00cbe",
			"type": "message",
			"text": "test message",
			"user": "UHSD97ZA5",
			"ts": "1658222505.879599",
			"blocks": [
				{
					"type": "rich_text",
					"block_id": "AUyAY",
					"elements": [
						{
							"type": "rich_text_section",
							"elements": [
								{
									"type": "text",
									"text": "test message"
								}
							]
						}
					]
				}
			],
			"team": "THY5HTZ8U",
			"user_team": "THY5HTZ8U",
			"source_team": "THY5HTZ8U",
			"user_profile": {
				"avatar_hash": "g1eca2328d65",
				"image_72": "https:\/\/secure.gravatar.com\/avatar\/41eca2328d6510133f47ffceae7b912a.jpg?s=72&d=https%3A%2F%2Fa.slack-edge.com%2Fdf10d%2Fimg%2Favatars%2Fava_0022-72.png",
				"first_name": "Rustam",
				"real_name": "Rustam Gilyazov",
				"display_name": "",
				"team": "THY5HTZ8U",
				"name": "gilyazov",
				"is_restricted": false,
				"is_ultra_restricted": false
			}
		},
		{
			"client_msg_id": "784c7c78-e7a7-46fa-95ff-948de0893754",
			"type": "message",
			"text": "test message 2",
			"user": "UHSD97ZA5",
			"ts": "1658222508.798439",
			"blocks": [
				{
					"type": "rich_text",
					"block_id": "j24",
					"elements": [
						{
							"type": "rich_text_section",
							"elements": [
								{
									"type": "text",
									"text": "test message 2"
								}
							]
						}
					]
				}
			],
			"team": "THY5HTZ8U",
			"user_team": "THY5HTZ8U",
			"source_team": "THY5HTZ8U",
			"user_profile": {
				"avatar_hash": "g1eca2328d65",
				"image_72": "https:\/\/secure.gravatar.com\/avatar\/41eca2328d6510133f47ffceae7b912a.jpg?s=72&d=https%3A%2F%2Fa.slack-edge.com%2Fdf10d%2Fimg%2Favatars%2Fava_0022-72.png",
				"first_name": "Rustam",
				"real_name": "Rustam Gilyazov",
				"display_name": "",
				"team": "THY5HTZ8U",
				"name": "gilyazov",
				"is_restricted": false,
				"is_ultra_restricted": false
			}
		}
	]
`
	AppMessageJSON = `{
	"type": "message",
	"ts": "1586042786.000100",
	"attachments": [
	  {
		"color": "4183c4",
		"fallback": "[xxx:v2] \u003chttps://bitbucket.org/xxx/xxx/commits/4e762b01256229a840784529a44381445ccb10b1|1 new commit\u003e by John Smoth",
		"id": 1,
		"pretext": "[xxx:v2] 1 new commit by John Smoth:",
		"text": "\u003chttps://wakatime.com/projects/project/commits|10 hrs 57 mins\u003e - \u003chttps://bitbucket.org/xxx/xxx/commits/4e762b01256229a840784529a44381445ccb10b1|4e762b0\u003e - iometer -\u0026gt; transport - John Smoth",
		"blocks": null
	  }
	],
	"subtype": "bot_message",
	"bot_id": "B011D8MSHK7",
	"username": "WakaTime",
	"replace_original": false,
	"delete_original": false,
	"metadata": {
	  "event_type": "",
	  "event_payload": null
	},
	"blocks": null
  }`
)

//go:embed assets/messages/bot_message.json
var BotMessageJSON string
