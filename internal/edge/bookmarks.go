package edge

import "encoding/json"

/*
channel: GM27XUQT0
include_folders: true
include_legacy_workflows: true
*/
type bookmarksListForm struct {
	BaseRequest
	Channel                string `json:"channel"`
	IncludeFolders         bool   `json:"include_folders"`
	IncludeLegacyWorkflows bool   `json:"include_legacy_workflows"`
}

type bookmarksListResponse struct {
	BaseResponse
	Bookmarks []json.RawMessage `json:"bookmarks"`
}

//"bookmarks.list"
