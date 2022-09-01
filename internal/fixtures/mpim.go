package fixtures

const (
	MpIM = `
  {
    "id": "C01AB34E90U",
    "created": 1615855270,
    "is_open": false,
    "is_group": false,
    "is_shared": false,
    "is_im": false,
    "is_ext_shared": false,
    "is_org_shared": false,
    "is_pending_ext_shared": false,
    "is_private": true,
    "is_mpim": true,
    "unlinked": 0,
    "name_normalized": "mpdm-yippi--ka--yay--motherfucker-1",
    "num_members": 4,
    "priority": 0,
    "user": "",
    "name": "mpdm-yippi--ka--yay--motherfucker-1",
    "creator": "LOL1",
    "is_archived": false,
    "members": [
      "LOL1",
      "DELD",
      "LOL3",
      "LOL4"
    ],
    "topic": {
      "value": "",
      "creator": "",
      "last_set": 0
    },
    "purpose": {
      "value": "Group messaging with: @yippi @ka @yay @motherfucker",
      "creator": "LOL1",
      "last_set": 1615855270
    },
    "is_channel": true,
    "is_general": false,
    "is_member": true,
    "locale": ""
  }
`

	MpIMNoMembers = `
  {
    "id": "G01M34GA2BX",
    "created": 1612757627,
    "is_open": false,
    "last_read": "1614837294.003700",
    "is_group": true,
    "is_shared": false,
    "is_im": false,
    "is_ext_shared": false,
    "is_org_shared": false,
    "is_pending_ext_shared": false,
    "is_private": true,
    "is_mpim": true,
    "unlinked": 0,
    "name_normalized": "mpdm-yippi--yay--motherfucker-1",
    "num_members": 0,
    "priority": 0,
    "user": "",
    "name": "mpdm-yippi--yay--motherfucker-1",
    "creator": "LOL1",
    "is_archived": false,
    "members": [],
    "topic": {
      "value": "",
      "creator": "",
      "last_set": 0
    },
    "purpose": {
      "value": "Group messaging with: @yippi @yay @motherfucker",
      "creator": "LOL1",
      "last_set": 1612757627
    },
    "is_channel": false,
    "is_general": false,
    "is_member": true,
    "locale": ""
  }
`

	MpIMnoMembersFixed = `  {
    "id": "G01M34GA2BX",
    "created": 1612757627,
    "is_open": false,
    "last_read": "1614837294.003700",
    "is_group": true,
    "is_shared": false,
    "is_im": false,
    "is_ext_shared": false,
    "is_org_shared": false,
    "is_pending_ext_shared": false,
    "is_private": true,
    "is_mpim": true,
    "unlinked": 0,
    "name_normalized": "mpdm-yippi--yay--motherfucker-1",
    "num_members": 0,
    "priority": 0,
    "user": "",
    "name": "mpdm-yippi--yay--motherfucker-1",
    "creator": "LOL1",
    "is_archived": false,
    "members": [
		"LOL1",
		"LOL3",
		"LOL4"
	],
    "topic": {
      "value": "",
      "creator": "",
      "last_set": 0
    },
    "purpose": {
      "value": "Group messaging with: @yippi @yay @motherfucker",
      "creator": "LOL1",
      "last_set": 1612757627
    },
    "is_channel": false,
    "is_general": false,
    "is_member": true,
    "locale": ""
  }`
)
