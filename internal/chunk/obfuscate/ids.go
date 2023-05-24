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
