package structures

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/rusq/fsadapter"

	"github.com/slack-go/slack"
)

// ExportIndex is the ExportIndex of the export archive.  filename tags are used to
// serialize the structure to JSON files.
type ExportIndex struct {
	Channels []slack.Channel `filename:"channels.json"`
	Groups   []slack.Channel `filename:"groups.json,omitempty"`
	MPIMs    []slack.Channel `filename:"mpims.json,omitempty"`
	DMs      []DM            `filename:"dms.json,omitempty"`
	Users    []slack.User    `filename:"users.json"`
}

// DM respresents a direct Message entry in dms.json.
// Structure is based on this post:
//
//	https://github.com/RocketChat/Rocket.Chat/issues/13905#issuecomment-477500022
type DM struct {
	ID      string   `json:"id"`
	Created int64    `json:"created"`
	Members []string `json:"members"`
}

var (
	ErrNoChannel = errors.New("empty channel data base")
	ErrNoUsers   = errors.New("empty users data base")
	ErrNoIdent   = errors.New("empty user identity")
)

// MakeExportIndex creates a channels and users index for export archive, splitting
// channels in group/mpims/dms/public channels.  currentUserID should contain
// the current user ID.
func MakeExportIndex(channels []slack.Channel, users []slack.User, currentUserID string) (*ExportIndex, error) {
	if len(channels) == 0 {
		return nil, ErrNoChannel
	}
	if len(users) == 0 {
		return nil, ErrNoUsers
	}
	if currentUserID == "" {
		return nil, ErrNoIdent
	}

	var idx = ExportIndex{
		Users:    users,
		Channels: make([]slack.Channel, 0, len(channels)),
		Groups:   []slack.Channel{},
		MPIMs:    []slack.Channel{},
		DMs:      []DM{},
	}

	for _, ch := range channels {
		switch {
		case ch.IsIM:
			idx.DMs = append(idx.DMs, DM{
				ID:      ch.ID,
				Created: int64(ch.Created),
				Members: []string{ch.User, currentUserID},
			})
		case ch.IsMpIM:
			fixed, err := FixMpIMmembers(&ch, users)
			if err != nil {
				return nil, err
			}
			idx.MPIMs = append(idx.MPIMs, *fixed)
		case ch.IsGroup:
			idx.Groups = append(idx.Groups, ch)
		default:
			idx.Channels = append(idx.Channels, ch)
		}
	}
	return &idx, nil
}

// Marshal writes the index to the filesystem in a set of files specified in
// `filename` tags of the structure.
func (idx *ExportIndex) Marshal(fs fsadapter.FS) error {
	if fs == nil {
		return errors.New("marshal: no fs")
	}
	st := reflect.TypeOf(*idx)
	val := reflect.ValueOf(*idx)
	for i := 0; i < st.NumField(); i++ {
		field := st.Field(i)
		tg := field.Tag.Get("filename")
		if tg == "" {
			continue
		}
		filename, option, found := strings.Cut(tg, ",")
		switch filename {
		case "-":
			continue
		case "":
			return fmt.Errorf("empty filename for: %s", field.Name)
		default:
		}
		if found && (option == "omitempty" && val.Field(i).IsZero()) {
			continue
		}
		if err := serializeToFS(fs, filename, val.Field(i).Interface()); err != nil {
			return err
		}
	}
	return nil
}

func serializeToFS(fs fsadapter.FS, filename string, data interface{}) error {
	f, err := fs.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}
