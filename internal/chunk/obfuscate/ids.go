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

package obfuscate

import (
	"encoding/hex"
	"strings"
)

const (
	userPrefix = "UO"
	chanPrefix = "CO"
	filePrefix = "FO"
	teamPrefix = "TO"
	appPrefix  = "AO"
	botPrefix  = "BO"
	entPrefix  = "EO"
)

// ID obfuscates an ID.
func (o obfuscator) ID(prefix string, id string) string {
	if id == "" {
		return ""
	}
	h := o.hasher()
	if _, err := h.Write([]byte(o.salt + id)); err != nil {
		panic(err)
	}
	return prefix + strings.ToUpper(hex.EncodeToString(h.Sum(nil)))[:len(id)-1]
}

func (o obfuscator) UserID(u string) string        { return o.ID(userPrefix, u) }
func (o obfuscator) ChannelID(c string) string     { return o.ID(chanPrefix, c) }
func (o obfuscator) FileID(f string) string        { return o.ID(filePrefix, f) }
func (o obfuscator) TeamID(g string) string        { return o.ID(teamPrefix, g) }
func (o *obfuscator) BotID(b string) string        { return o.ID(botPrefix, b) }
func (o *obfuscator) AppID(a string) string        { return o.ID(appPrefix, a) }
func (o *obfuscator) EnterpriseID(e string) string { return o.ID(entPrefix, e) }
