package slackdump

// In this file: user related code.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/trace"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/pkg/errors"
	"github.com/rusq/dlog"
	"github.com/slack-go/slack"

	"github.com/rusq/slackdump/v2/internal/network"
)

// Users is a slice of users.
type Users []slack.User

// GetUsers retrieves all users either from cache or from the API.
func (sd *SlackDumper) GetUsers(ctx context.Context) (Users, error) {
	// TODO: validate that the cache is from the same workspace, it can be done by team ID.
	ctx, task := trace.NewTask(ctx, "GetUsers")
	defer task.End()

	if sd.options.NoUserCache {
		return Users{}, nil
	}

	users, err := sd.loadUserCache(sd.options.UserCacheFilename, sd.teamID, sd.options.MaxUserCacheAge)
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
		if err := sd.saveUserCache(sd.options.UserCacheFilename, sd.teamID, users); err != nil {
			trace.Logf(ctx, "error", "saving user cache to %q, error: %s", sd.options.UserCacheFilename, err)
			dlog.Printf("error saving user cache to %q: %s, but nevermind, let's continue", sd.options.UserCacheFilename, err)
		}
	}

	return users, err
}

// fetchUsers fetches users from the API.
func (sd *SlackDumper) fetchUsers(ctx context.Context) (Users, error) {
	var (
		users []slack.User
	)
	if err := withRetry(ctx, network.NewLimiter(network.Tier2, sd.options.Tier2Burst, int(sd.options.Tier2Boost)), sd.options.Tier2Retries, func() error {
		var err error
		users, err = sd.client.GetUsersContext(ctx)
		return err
	}); err != nil {
		trace.Logf(ctx, "error", "GetUsers error=%s", err)
		return nil, err
	}
	// BUG: as of 201902 there's a bug in slack module, the invalid_auth error
	// is not propagated properly, so we'll check for number of users.  There
	// should be at least one (slackbot).
	if len(users) == 0 {
		return nil, errors.New("couldn't fetch users")
	}
	return users, nil
}

// loadUsers tries to load the users from the file
func (sd *SlackDumper) loadUserCache(filename string, suffix string, maxAge time.Duration) (Users, error) {
	filename = makeCacheFilename(filename, suffix)

	if err := checkCacheFile(filename, maxAge); err != nil {
		return nil, err
	}

	f, err := os.Open(filename)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	var uu Users
	if err := dec.Decode(&uu); err != nil {
		return nil, errors.WithStack(err)
	}
	return uu, nil
}

func (sd *SlackDumper) saveUserCache(filename string, suffix string, uu Users) error {
	filename = makeCacheFilename(filename, suffix)

	f, err := os.Create(filename)
	if err != nil {
		return errors.WithStack(err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(uu); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// ToText outputs Users us to io.Writer w in Text format
func (us Users) ToText(w io.Writer, _ *SlackDumper) error {
	const strFormat = "%s\t%s\t%s\t%s\t%s\n"
	writer := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	defer writer.Flush()

	// header
	if _, err := fmt.Fprintf(writer, strFormat, "Name", "ID", "Bot?", "Deleted?", "Restricted?"); err != nil {
		return errors.WithStack(err)
	}
	if _, err := fmt.Fprintf(writer, strFormat, "", "", "", "", ""); err != nil {
		return errors.WithStack(err)
	}

	var (
		names   = make([]string, 0, len(us))
		usermap = make(map[string]*slack.User, len(us))
	)
	for i := range us {
		names = append(names, us[i].Name)
		usermap[us[i].Name] = &us[i]
	}
	sort.Strings(names)

	// data
	for _, name := range names {
		var (
			deleted    string
			bot        string
			restricted string
		)
		if usermap[name].Deleted {
			deleted = "deleted"
		}
		if usermap[name].IsBot {
			bot = "bot"
		}
		if usermap[name].IsRestricted {
			restricted = "restricted"
		}

		_, err := fmt.Fprintf(writer, strFormat,
			name, usermap[name].ID, bot, deleted, restricted,
		)
		if err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

// IndexByID returns the userID map to relevant *slack.User
func (us Users) IndexByID() map[string]*slack.User {
	var usermap = make(map[string]*slack.User, len(us))

	for i := range us {
		usermap[(us)[i].ID] = &us[i]
	}

	return usermap
}

// IsUserDeleted checks if the user is deleted and returns appropriate value. It
// will assume user is not deleted, if it's not present in the user index.
func (sd *SlackDumper) IsUserDeleted(id string) bool {
	thisUser, ok := sd.UserIndex[id]
	if !ok {
		return false
	}
	return thisUser.Deleted
}

// makeCacheFilename converts filename.ext to filename-suffix.ext.
func makeCacheFilename(filename, suffix string) string {
	ne := filenameSplit(filename)
	return filenameJoin(nameExt{ne[0] + "-" + suffix, ne[1]})
}

type nameExt [2]string

// filenameSplit splits the "path/to/filename.ext" into nameExt{"path/to/filename", ".ext"}
func filenameSplit(filename string) nameExt {
	ext := filepath.Ext(filename)
	name := filename[:len(filename)-len(ext)]
	return nameExt{name, ext}
}

// filenameJoin combines nameExt{"path/to/filename", ".ext"} to "path/to/filename.ext".
func filenameJoin(split nameExt) string {
	return split[0] + split[1]
}
