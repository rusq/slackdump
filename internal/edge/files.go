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

	"github.com/rusq/slack"
)

// files.* API

type filesListForm struct {
	BaseRequest
	Channel string `json:"channel"`
	Count   int    `json:"count"`
	Page    int    `json:"page"`
	WebClientFields
}

type filesListResponse struct {
	baseResponse
	Files []slack.File `json:"files"`
	Pagination
}

func (cl *Client) FilesList(ctx context.Context, channel string, count int) ([]slack.File, error) {
	form := filesListForm{
		BaseRequest:     BaseRequest{Token: cl.token},
		Channel:         channel,
		Count:           count,
		WebClientFields: webclientReason("about-modal/sharedFiles"),
	}
	lim := tier3.limiter()
	var ff []slack.File
	for {
		resp, err := cl.Post(ctx, "files.list", form)
		if err != nil {
			return nil, err
		}
		r := filesListResponse{}
		if err := cl.ParseResponse(&r, resp); err != nil {
			return nil, err
		}
		ff = append(ff, r.Files...)
		if form.Page == int(r.PageCount) || r.PageCount == 0 {
			break
		}
		form.Page++
		if err := lim.Wait(ctx); err != nil {
			return nil, err
		}
	}
	return ff, nil
}
