package slackdump

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime/trace"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/pkg/errors"
	"github.com/rusq/dlog"
	"github.com/slack-go/slack"
)

// Users keeps slice of users
type Users []slack.User

// GetUsers retrieves all users either from cache or from the API.
func (sd *SlackDumper) GetUsers(ctx context.Context) (Users, error) {
	// TODO: validate that the cache is from the same workspace, it can be done by team ID.
	ctx, task := trace.NewTask(ctx, "GetUsers")
	defer task.End()

	users, err := sd.loadUserCache(sd.options.userCacheFilename)
	if err != nil {
		if os.IsNotExist(err) {
			dlog.Println("  caching users for the first time")
		} else {
			dlog.Printf("  failed to load cache, it will be recreated: %s", err)
		}
		users, err = sd.fetchUsers(ctx)
		if err != nil {
			return nil, err
		}
		if err := sd.saveUserCache(sd.options.userCacheFilename, users); err != nil {
			dlog.Printf("error saving user cache to %q: %s, but nevermind, let's continue", sd.options.userCacheFilename, err)
		}
	}

	sd.Users = users
	sd.UserIndex = sd.Users.IndexByID()

	return users, err
}

// fetchUsers fetches users from the API.
func (sd *SlackDumper) fetchUsers(ctx context.Context) (Users, error) {
	users, err := sd.client.GetUsers()
	if err != nil {
		return nil, err
	}
	// BUG: as of 201902 there's a bug in slack module, the invalid_auth error
	// is not propagated properly, so we'll check for number of users.  There
	// should be at least one (slackbot).
	if len(users) == 0 {
		return nil, fmt.Errorf("couldn't fetch users")
	}
	return users, nil
}

// loadUsers tries to load the users from the file
func (sd *SlackDumper) loadUserCache(filename string) (Users, error) {
	if err := sd.validateUserCache(filename); err != nil {
		return nil, err
	}
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	var uu Users
	if err := dec.Decode(&uu); err != nil {
		return nil, err
	}
	return uu, nil
}

func (*SlackDumper) saveUserCache(filename string, uu Users) error {
	f, err := os.Create("users.json")
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(uu); err != nil {
		return err
	}
	return nil
}

func (sd *SlackDumper) validateUserCache(filename string) error {
	if filename == "" {
		return errors.New("no user cache filename")
	}
	fi, err := os.Stat(filename)
	if err != nil {
		return err
	}
	if fi.Size() == 0 {
		return errors.New("empty user cache")
	}
	if time.Since(fi.ModTime()) > sd.options.maxUserCacheAge {
		return errors.New("user cache expired")
	}
	return nil
}

// ToText outputs Users us to io.Writer w in Text format
func (us Users) ToText(sd *SlackDumper, w io.Writer) (err error) {
	const strFormat = "%s\t%s\t%s\t%s\n"
	writer := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	defer writer.Flush()

	var names = make([]string, 0, len(us))
	var usermap = make(map[string]*slack.User, len(us))
	for i := range us {
		names = append(names, us[i].Name)
		usermap[us[i].Name] = &us[i]
	}
	sort.Strings(names)

	// header
	fmt.Fprintf(writer, strFormat, "Name", "ID", "Bot?", "Deleted?")
	fmt.Fprintf(writer, strFormat, "", "", "", "")

	for _, name := range names {
		var deleted, bot string
		if usermap[name].Deleted {
			deleted = "deleted"
		}
		if usermap[name].IsBot {
			bot = "bot"
		}

		fmt.Fprintf(writer, strFormat,
			name, usermap[name].ID, bot, deleted)
	}
	return
}

// IndexByID returns the userID map to relevant *slack.User
func (us Users) IndexByID() map[string]*slack.User {
	var usermap = make(map[string]*slack.User, len(us))

	for i := range us {
		usermap[(us)[i].ID] = &us[i]
	}

	return usermap
}

// Len returns the user count
func (us Users) Len() int {
	return len(us)
}

// IsDeletedUser checks if the user is deleted and returns appropriate value
func (sd *SlackDumper) IsDeletedUser(id string) bool {
	thisUser, ok := sd.UserIndex[id]
	if !ok {
		return false
	}
	return thisUser.Deleted
}
