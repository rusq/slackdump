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
	"fmt"
	"strings"
)

// quip.* API

type quipLookupForm struct {
	BaseRequest
	FileIDs string `json:"file_ids"`
	WebClientFields
}

type quipLookupResponse struct {
	baseResponse
	Lookup map[string]string `json:"lookup"`
}

// QuipLookupThreadIDs maps Slack file IDs to Quip/OYP document IDs used
// in canvas load-data requests. Returns a map of fileID → OYP ID.
func (cl *Client) QuipLookupThreadIDs(ctx context.Context, fileID ...string) (map[string]string, error) {
	if len(fileID) == 0 {
		return map[string]string{}, nil
	}
	const ep = "quip.lookupThreadIds"
	form := quipLookupForm{
		BaseRequest:     BaseRequest{Token: cl.token},
		FileIDs:         strings.Join(fileID, ","),
		WebClientFields: webclientReason("fetch-quip-ids"),
	}
	resp, err := cl.Post(ctx, ep, form)
	if err != nil {
		return nil, err
	}
	var r quipLookupResponse
	if err := cl.ParseResponse(&r, resp); err != nil {
		return nil, fmt.Errorf("%s: %w", ep, err)
	}
	if err := r.validate(ep); err != nil {
		return nil, err
	}
	return r.Lookup, nil
}
