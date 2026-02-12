// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package edge

import (
	"context"
	"runtime/trace"
)

// im.* API

type imListForm struct {
	BaseRequest
	GetLatest    bool   `json:"get_latest"`
	GetReadState bool   `json:"get_read_state"`
	Cursor       string `json:"cursor,omitempty"`
	WebClientFields
}

type imListResponse struct {
	baseResponse
	IMs []IM `json:"ims,omitempty"`
}

func (cl *Client) IMList(ctx context.Context) ([]IM, error) {
	ctx, task := trace.NewTask(ctx, "IMList")
	defer task.End()

	form := imListForm{
		BaseRequest:  BaseRequest{Token: cl.token},
		GetLatest:    true,
		GetReadState: true,
		WebClientFields: WebClientFields{
			XReason:  "guided-search-people-empty-state",
			XMode:    "online",
			XSonic:   true,
			XAppName: "client",
		},
		Cursor: "",
	}
	lim := tier2boost.limiter()
	var IMs []IM
	for {
		resp, err := cl.PostForm(ctx, "im.list", values(form, true))
		if err != nil {
			return nil, err
		}
		r := imListResponse{}
		if err := cl.ParseResponse(&r, resp); err != nil {
			return nil, err
		}
		IMs = append(IMs, r.IMs...)
		if r.ResponseMetadata.NextCursor == "" {
			break
		}
		form.Cursor = r.ResponseMetadata.NextCursor
		if err := lim.Wait(ctx); err != nil {
			return nil, err
		}
	}
	return IMs, nil
}
