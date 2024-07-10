package fixtures

const TestChannels = `[
    {
        "id": "C03EDPTKG93",
        "created": 1651900614,
        "is_open": false,
        "is_group": false,
        "is_shared": false,
        "is_im": false,
        "is_ext_shared": false,
        "is_org_shared": false,
        "is_pending_ext_shared": false,
        "is_private": false,
        "is_mpim": false,
        "unlinked": 0,
        "name_normalized": "random",
        "num_members": 16,
        "priority": 0,
        "user": "",
        "name": "random",
        "creator": "U03EVA9J397",
        "is_archived": false,
        "members": null,
        "topic": {
            "value": "",
            "creator": "",
            "last_set": 0
        },
        "purpose": {
            "value": "This channel is for... well, everything else. It’s a place for team jokes, spur-of-the-moment ideas and funny GIFs. Go wild!",
            "creator": "U03EVA9J397",
            "last_set": 1651900614
        },
        "is_channel": true,
        "is_general": false,
        "is_member": true,
        "locale": ""
    },
    {
        "id": "C03EDPUCR53",
        "created": 1651900667,
        "is_open": false,
        "is_group": false,
        "is_shared": false,
        "is_im": false,
        "is_ext_shared": false,
        "is_org_shared": false,
        "is_pending_ext_shared": false,
        "is_private": false,
        "is_mpim": false,
        "unlinked": 0,
        "name_normalized": "slackdump",
        "num_members": 16,
        "priority": 0,
        "user": "",
        "name": "slackdump",
        "creator": "U03EVA9J397",
        "is_archived": false,
        "members": null,
        "topic": {
            "value": "",
            "creator": "",
            "last_set": 0
        },
        "purpose": {
            "value": "This *channel* is for working on a project. Hold meetings, share docs and make decisions together with your team.",
            "creator": "U03EVA9J397",
            "last_set": 1651900667
        },
        "is_channel": true,
        "is_general": false,
        "is_member": true,
        "locale": ""
    },
    {
        "id": "C03EGLZ57GS",
        "created": 1651900613,
        "is_open": false,
        "is_group": false,
        "is_shared": false,
        "is_im": false,
        "is_ext_shared": false,
        "is_org_shared": false,
        "is_pending_ext_shared": false,
        "is_private": false,
        "is_mpim": false,
        "unlinked": 0,
        "name_normalized": "general",
        "num_members": 16,
        "priority": 0,
        "user": "",
        "name": "general",
        "creator": "U03EVA9J397",
        "is_archived": false,
        "members": null,
        "topic": {
            "value": "",
            "creator": "",
            "last_set": 0
        },
        "purpose": {
            "value": "This is the one channel that will always include everyone. It’s a great spot for announcements and team-wide conversations.",
            "creator": "U03EVA9J397",
            "last_set": 1651900613
        },
        "is_channel": true,
        "is_general": true,
        "is_member": true,
        "locale": ""
    },
    {
        "id": "D04BDRSE3GQ",
        "created": 1668352023,
        "is_open": false,
        "is_group": false,
        "is_shared": false,
        "is_im": true,
        "is_ext_shared": false,
        "is_org_shared": false,
        "is_pending_ext_shared": false,
        "is_private": false,
        "is_mpim": false,
        "unlinked": 0,
        "name_normalized": "",
        "num_members": 0,
        "priority": 0,
        "user": "U04B2PYV4QH",
        "name": "",
        "creator": "",
        "is_archived": false,
        "members": null,
        "topic": {
            "value": "",
            "creator": "",
            "last_set": 0
        },
        "purpose": {
            "value": "",
            "creator": "",
            "last_set": 0
        },
        "is_channel": false,
        "is_general": false,
        "is_member": false,
        "locale": ""
    }
]
`

// TestChannelsNativeExport are from the real Slack workspace export.
const TestChannelsNativeExport = `[
    {
        "id": "CHM82GF99",
        "name": "everything",
        "created": 1555493779,
        "creator": "UHSD97ZA5",
        "is_archived": false,
        "is_general": false,
        "members": [
            "UHSD97ZA5",
            "ULLLZ6SAH"
        ],
        "topic": {
            "value": "",
            "creator": "",
            "last_set": 0
        },
        "purpose": {
            "value": "",
            "creator": "",
            "last_set": 0
        }
    },
    {
        "id": "CHY5HUESG",
        "name": "everyone",
        "created": 1555493778,
        "creator": "UHSD97ZA5",
        "is_archived": false,
        "is_general": true,
        "members": [
            "UHSD97ZA5",
            "ULLLZ6SAH"
        ],
        "topic": {
            "value": "Company-wide announcements and work-based matters",
            "creator": "UHSD97ZA5",
            "last_set": 1555493778
        },
        "purpose": {
            "value": "This channel is for workspace-wide communication and announcements. All members are in this channel.",
            "creator": "UHSD97ZA5",
            "last_set": 1555493778
        }
    },
    {
        "id": "CHYLGDP0D",
        "name": "random",
        "created": 1555493778,
        "creator": "UHSD97ZA5",
        "is_archived": false,
        "is_general": false,
        "members": [
            "UHSD97ZA5",
            "ULLLZ6SAH",
            "U034HM0P7RB"
        ],
        "topic": {
            "value": "Non-work banter and water cooler conversation",
            "creator": "UHSD97ZA5",
            "last_set": 1555493778
        },
        "purpose": {
            "value": "A place for non-work-related flimflam, faffing, hodge-podge or jibber-jabber you'd prefer to keep out of more focused work-related channels.",
            "creator": "UHSD97ZA5",
            "last_set": 1555493778
        }
    },
    {
        "id": "C011D885FP0",
        "name": "wakatime",
        "created": 1586035665,
        "creator": "UHSD97ZA5",
        "is_archived": false,
        "is_general": false,
        "members": [
            "UHSD97ZA5"
        ],
        "topic": {
            "value": "",
            "creator": "",
            "last_set": 0
        },
        "purpose": {
            "value": "Timesheet",
            "creator": "UHSD97ZA5",
            "last_set": 1586035666
        }
    },
    {
        "id": "C045TUGSSTW",
        "name": "adapt_w_3dビューア",
        "created": 1665307423,
        "creator": "UHSD97ZA5",
        "is_archived": false,
        "is_general": false,
        "members": [
            "UHSD97ZA5"
        ],
        "topic": {
            "value": "",
            "creator": "",
            "last_set": 0
        },
        "purpose": {
            "value": "Issue 44 test",
            "creator": "UHSD97ZA5",
            "last_set": 1665307423
        }
    },
    {
        "id": "C04BJATRQRL",
        "name": "slackdump",
        "created": 1668926667,
        "creator": "UHSD97ZA5",
        "is_archived": false,
        "is_general": false,
        "members": [
            "UHSD97ZA5"
        ],
        "topic": {
            "value": "",
            "creator": "",
            "last_set": 0
        },
        "purpose": {
            "value": "",
            "creator": "",
            "last_set": 0
        }
    }
    ]
`
