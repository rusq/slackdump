package fixtures

import (
	"embed"
	_ "embed"
)

const TestConversationExportJSON = `{
	"2019-04-17": [
	  {
		"type": "message",
		"user": "UHSD97ZA5",
		"text": "\u003c@UHSD97ZA5\u003e has joined the channel",
		"ts": "1555493779.000200",
		"subtype": "channel_join",
		"replace_original": false,
		"delete_original": false,
		"blocks": null,
		"user_profile": {
		  "image_72": "https://secure.gravatar.com/avatar/41eca2328d6510133f47ffceae7b912a.jpg?s=72\u0026d=https%3A%2F%2Fa.slack-edge.com%2Fdf10d%2Fimg%2Favatars%2Fava_0022-72.png",
		  "first_name": "Jane",
		  "real_name": "Jane Doe",
		  "team": "THY5HTZ8U",
		  "name": "janed"
		}
	  }
	],
	"2019-07-20": [
	  {
		"type": "message",
		"user": "ULLLZ6SAH",
		"text": "\u003c@ULLLZ6SAH\u003e has joined the channel",
		"ts": "1563609394.000200",
		"subtype": "channel_join",
		"replace_original": false,
		"delete_original": false,
		"blocks": null,
		"user_profile": {
		  "image_72": "https://secure.gravatar.com/avatar/95a51d9723dbea1d8af04d31e35e1d37.jpg?s=72\u0026d=https%3A%2F%2Fa.slack-edge.com%2Fdf10d%2Fimg%2Favatars%2Fava_0020-72.png",
		  "real_name": "John",
		  "team": "THY5HTZ8U",
		  "name": "johnd"
		}
	  }
	],
	"2019-07-25": [
	  {
		"type": "message",
		"user": "UHSD97ZA5",
		"text": "hello",
		"ts": "1564023445.000100",
		"bot_id": "BKQPUHWF2",
		"bot_profile": {
		  "app_id": "AKDB9CUKC",
		  "deleted": true,
		  "icons": {
			"image_36": "https://a.slack-edge.com/80588/img/plugins/app/bot_36.png",
			"image_48": "https://a.slack-edge.com/80588/img/plugins/app/bot_48.png",
			"image_72": "https://a.slack-edge.com/80588/img/plugins/app/service_72.png"
		  },
		  "id": "BKQPUHWF2",
		  "name": "Redash Monitor",
		  "team_id": "THY5HTZ8U",
		  "updated": 1592611052
		},
		"team": "THY5HTZ8U",
		"replace_original": false,
		"delete_original": false,
		"blocks": null,
		"user_team": "THY5HTZ8U",
		"source_team": "THY5HTZ8U",
		"user_profile": {
		  "image_72": "https://secure.gravatar.com/avatar/41eca2328d6510133f47ffceae7b912a.jpg?s=72\u0026d=https%3A%2F%2Fa.slack-edge.com%2Fdf10d%2Fimg%2Favatars%2Fava_0022-72.png",
		  "first_name": "Jane",
		  "real_name": "Jane Doe",
		  "team": "THY5HTZ8U",
		  "name": "janed"
		}
	  },
	  {
		"type": "message",
		"user": "UHSD97ZA5",
		"text": "hello",
		"ts": "1564023560.000200",
		"bot_id": "BKQPUHWF2",
		"bot_profile": {
		  "app_id": "AKDB9CUKC",
		  "deleted": true,
		  "icons": {
			"image_36": "https://a.slack-edge.com/80588/img/plugins/app/bot_36.png",
			"image_48": "https://a.slack-edge.com/80588/img/plugins/app/bot_48.png",
			"image_72": "https://a.slack-edge.com/80588/img/plugins/app/service_72.png"
		  },
		  "id": "BKQPUHWF2",
		  "name": "Redash Monitor",
		  "team_id": "THY5HTZ8U",
		  "updated": 1592611052
		},
		"team": "THY5HTZ8U",
		"replace_original": false,
		"delete_original": false,
		"blocks": null,
		"user_team": "THY5HTZ8U",
		"source_team": "THY5HTZ8U",
		"user_profile": {
		  "image_72": "https://secure.gravatar.com/avatar/41eca2328d6510133f47ffceae7b912a.jpg?s=72\u0026d=https%3A%2F%2Fa.slack-edge.com%2Fdf10d%2Fimg%2Favatars%2Fava_0022-72.png",
		  "first_name": "Jane",
		  "real_name": "Jane Doe",
		  "team": "THY5HTZ8U",
		  "name": "janed"
		}
	  }
	],
	"2019-12-30": [
	  {
		"client_msg_id": "676e1cbb-15fe-45e9-b7f2-32a8764fe560",
		"type": "message",
		"user": "UHSD97ZA5",
		"text": "This ~is a~  Rich Text message test.",
		"ts": "1577694990.000400",
		"thread_ts": "1577694990.000400",
		"last_read": "1648633700.407619",
		"subscribed": true,
		"reply_count": 3,
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
		"user_team": "THY5HTZ8U",
		"source_team": "THY5HTZ8U",
		"user_profile": {
		  "image_72": "https://secure.gravatar.com/avatar/41eca2328d6510133f47ffceae7b912a.jpg?s=72\u0026d=https%3A%2F%2Fa.slack-edge.com%2Fdf10d%2Fimg%2Favatars%2Fava_0022-72.png",
		  "first_name": "Jane",
		  "real_name": "Jane Doe",
		  "team": "THY5HTZ8U",
		  "name": "janed"
		},
		"reply_users_count": 1,
		"reply_users": [
		  "UHSD97ZA5"
		]
	  }
	],
	"2021-12-06": [
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
		],
		"user_team": "THY5HTZ8U",
		"source_team": "THY5HTZ8U",
		"user_profile": {
		  "image_72": "https://secure.gravatar.com/avatar/41eca2328d6510133f47ffceae7b912a.jpg?s=72\u0026d=https%3A%2F%2Fa.slack-edge.com%2Fdf10d%2Fimg%2Favatars%2Fava_0022-72.png",
		  "first_name": "Jane",
		  "real_name": "Jane Doe",
		  "team": "THY5HTZ8U",
		  "name": "janed"
		}
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
		],
		"user_profile": {
		  "image_72": "https://secure.gravatar.com/avatar/41eca2328d6510133f47ffceae7b912a.jpg?s=72\u0026d=https%3A%2F%2Fa.slack-edge.com%2Fdf10d%2Fimg%2Favatars%2Fava_0022-72.png",
		  "first_name": "Jane",
		  "real_name": "Jane Doe",
		  "team": "THY5HTZ8U",
		  "name": "janed"
		}
	  }
	],
	"2022-02-17": [
	  {
		"client_msg_id": "c6cdfb3a-59d6-4198-9800-cc74bcdc0b7d",
		"type": "message",
		"user": "UHSD97ZA5",
		"text": "Test message with Html chars \u0026lt; \u0026gt;",
		"ts": "1645095505.023899",
		"team": "THY5HTZ8U",
		"replace_original": false,
		"delete_original": false,
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
					"text": "Test message with Html chars \u003c \u003e"
				  }
				]
			  }
			]
		  }
		],
		"user_team": "THY5HTZ8U",
		"source_team": "THY5HTZ8U",
		"user_profile": {
		  "image_72": "https://secure.gravatar.com/avatar/41eca2328d6510133f47ffceae7b912a.jpg?s=72\u0026d=https%3A%2F%2Fa.slack-edge.com%2Fdf10d%2Fimg%2Favatars%2Fava_0022-72.png",
		  "first_name": "Jane",
		  "real_name": "Jane Doe",
		  "team": "THY5HTZ8U",
		  "name": "janed"
		}
	  }
	],
	"2022-03-30": [
	  {
		"client_msg_id": "2aeb2443-971e-454b-a61b-c18583ae783f",
		"type": "message",
		"user": "UHSD97ZA5",
		"text": "30-Mar-2022",
		"ts": "1648633633.716099",
		"team": "THY5HTZ8U",
		"replace_original": false,
		"delete_original": false,
		"blocks": [
		  {
			"type": "rich_text",
			"block_id": "F1/cq",
			"elements": [
			  {
				"type": "rich_text_section",
				"elements": [
				  {
					"type": "text",
					"text": "30-Mar-2022"
				  }
				]
			  }
			]
		  }
		],
		"user_team": "THY5HTZ8U",
		"source_team": "THY5HTZ8U",
		"user_profile": {
		  "image_72": "https://secure.gravatar.com/avatar/41eca2328d6510133f47ffceae7b912a.jpg?s=72\u0026d=https%3A%2F%2Fa.slack-edge.com%2Fdf10d%2Fimg%2Favatars%2Fava_0022-72.png",
		  "first_name": "Jane",
		  "real_name": "Jane Doe",
		  "team": "THY5HTZ8U",
		  "name": "janed"
		}
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
		],
		"user_team": "THY5HTZ8U",
		"source_team": "THY5HTZ8U",
		"user_profile": {
		  "image_72": "https://secure.gravatar.com/avatar/41eca2328d6510133f47ffceae7b912a.jpg?s=72\u0026d=https%3A%2F%2Fa.slack-edge.com%2Fdf10d%2Fimg%2Favatars%2Fava_0022-72.png",
		  "first_name": "Jane",
		  "real_name": "Jane Doe",
		  "team": "THY5HTZ8U",
		  "name": "janed"
		}
	  }
	]
  }`

// Export fixtures.
var (
	//go:embed assets/export/dms.json
	TestExpDMsJSON []byte

	//go:embed assets/export/mpims.json
	TestExpMPIMsJSON []byte

	//go:embed assets/export/groups.json
	TestExpGroupsJSON []byte

	//go:embed assets/export/channels.json
	TestExpChannelsJSON []byte

	//go:embed assets/export/users.json
	TestExpUsersJSON []byte

	//go:embed assets/export/ref-channels.json
	TestExpReferenceChannelsJSON []byte
)

//go:embed assets/export/*.json
var TestExportFS embed.FS
