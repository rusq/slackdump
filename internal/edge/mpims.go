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

// mpim.* API

type mpimListForm struct {
	BaseRequest
	GetLatest bool   `json:"get_latest"`
	Cursor    string `json:"cursor,omitempty"`
	WebClientFields
}

type mpimListResponse struct {
	baseResponse
	MPIMs []UserBootChannel `json:"groups,omitempty"`
}

func (cl *Client) MPIMList(ctx context.Context) ([]UserBootChannel, error) {
	ctx, task := trace.NewTask(ctx, "MPIMList")
	defer task.End()

	form := mpimListForm{
		BaseRequest: BaseRequest{Token: cl.token},
		GetLatest:   true,
		WebClientFields: WebClientFields{
			XReason:  "external-connections-browser-conversation-counts",
			XMode:    "online",
			XSonic:   true,
			XAppName: "client",
		},
		Cursor: "",
	}
	lim := tier2boost.limiter()
	var MPIMs []UserBootChannel
	for {
		resp, err := cl.PostForm(ctx, "mpim.list", values(form, true))
		if err != nil {
			return nil, err
		}
		r := mpimListResponse{}
		if err := cl.ParseResponse(&r, resp); err != nil {
			return nil, err
		}
		if err := r.validate("mpim.list"); err != nil {
			return nil, err
		}
		MPIMs = append(MPIMs, r.MPIMs...)
		if r.ResponseMetadata.NextCursor == "" {
			break
		}
		form.Cursor = r.ResponseMetadata.NextCursor
		if err := lim.Wait(ctx); err != nil {
			return nil, err
		}
	}
	return MPIMs, nil
}
