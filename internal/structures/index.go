package structures

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"reflect"
	"strings"

	"github.com/rusq/fsadapter"

	"github.com/rusq/slack"
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
			idx.DMs = append(idx.DMs, convertToDM(currentUserID, ch))
		case ch.IsMpIM:
			if ch.NumMembers == 0 {
				ch.NumMembers = len(ch.Members)
			}
			idx.MPIMs = append(idx.MPIMs, ch)
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
		tag := field.Tag.Get(filenameTag)
		if tag == "" {
			continue
		}
		filename, option, found := strings.Cut(tag, tagSep)
		switch filename {
		case "-":
			continue
		case "":
			return fmt.Errorf("missing filename for: %s", field.Name)
		default:
		}
		if found && (option == omitemptyTagOpt && val.Field(i).IsZero()) {
			continue
		}
		if err := marshalFileFSA(fs, filename, val.Field(i).Interface()); err != nil {
			return err
		}
	}
	return nil
}

const (
	filenameTag     = "filename"
	omitemptyTagOpt = "omitempty"
	tagSep          = ","
)

// Unmarshal reads the index from the filesystem in a set of files specified in
// `filename` tags of the structure.
func (idx *ExportIndex) Unmarshal(fsys fs.FS) error {
	var newIdx ExportIndex

	st := reflect.TypeOf(*idx)
	val := reflect.ValueOf(&newIdx).Elem()
	for i := 0; i < st.NumField(); i++ {
		field := st.Field(i)
		tg := field.Tag.Get(filenameTag)
		if tg == "" {
			continue
		}
		filename, _, _ := strings.Cut(tg, tagSep)
		if err := unmarshalFileFS(fsys, filename, val.Field(i).Addr().Interface()); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return err
		}
	}
	*idx = newIdx
	return nil
}

// Restore restores the index to the original channels slice (minus the lost
// data from DMs).
func (idx *ExportIndex) Restore() []slack.Channel {
	var chans = make([]slack.Channel, 0, len(idx.Channels)+len(idx.Groups)+len(idx.MPIMs)+len(idx.DMs))

	chans = append(chans, idx.Channels...)
	chans = append(chans, idx.Groups...)
	chans = append(chans, idx.MPIMs...)
	chans = append(chans, dmsToChannels(idx.DMs)...)

	return chans
}

func dmsToChannels(DMs []DM) []slack.Channel {
	me := mostFrequentMember(DMs)
	var chans = make([]slack.Channel, 0, len(DMs))
	for _, dm := range DMs {
		chans = append(chans, slack.Channel{
			GroupConversation: slack.GroupConversation{
				Conversation: slack.Conversation{
					ID:      dm.ID,
					Created: slack.JSONTime(dm.Created),
					IsIM:    true,
					User:    except(me, dm.Members),
				},
				Members: dm.Members,
			},
		})
	}
	return chans
}

func marshalFileFSA(fs fsadapter.FS, filename string, data any) error {
	f, err := fs.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

// unmarshalFileFS unmarshals the file with filename from the fsys into data.
func unmarshalFileFS(fsys fs.FS, filename string, data any) error {
	f, err := fsys.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	return dec.Decode(data)
}

// except returns the first element of the slice that is not s, or zero value
// if not found.
func except[S ~[]T, T comparable](s T, ss S) T {
	var zero T
	for _, t := range ss {
		if t != s {
			return t
		}
	}
	return zero
}

// mostFrequentMember attempts to identify the current user in the index.  It uses the DMs of
// the index. If DMs are empty, or it's unable to identify the user, it
// returns an empty string.  The user, who appears in "Members" slices the
// most, is considered the current user.
func mostFrequentMember(dms []DM) string {
	var counts = make(map[string]int)
	for _, dm := range dms {
		for _, m := range dm.Members {
			counts[m]++
		}
	}
	var (
		max int
		id  string
	)
	for k, v := range counts {
		if v > max {
			max = v
			id = k
		}
	}
	return id
}

func convertToDM(me string, ch slack.Channel) DM {
	d := DM{
		ID:      ch.ID,
		Created: int64(ch.Created),
	}
	switch len(ch.Members) {
	case 0:
		d.Members = []string{ch.User, me}
	case 1:
		d.Members = []string{ch.User, me}
	default:
		d.Members = ch.Members
	}
	return d
}
