package slackdump

// In this file: user related code.

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime/trace"
	"time"

	"errors"

	"github.com/slack-go/slack"

	"github.com/rusq/slackdump/v2/internal/network"
	"github.com/rusq/slackdump/v2/types"
)

// GetUsers retrieves all users either from cache or from the API.
func (sd *Session) GetUsers(ctx context.Context) (types.Users, error) {
	// TODO: validate that the cache is from the same workspace, it can be done by team ID.
	ctx, task := trace.NewTask(ctx, "GetUsers")
	defer task.End()

	if sd.options.NoUserCache {
		return types.Users{}, nil
	}

	users, err := sd.loadUserCache(sd.options.UserCacheFilename, sd.wspInfo.TeamID, sd.options.MaxUserCacheAge)
	if err != nil {
		if os.IsNotExist(err) {
			sd.l().Println("  caching users for the first time")
		} else {
			sd.l().Printf("  %s: it will be recreated.", err)
		}
		users, err = sd.fetchUsers(ctx)
		if err != nil {
			return nil, err
		}
		if err := sd.saveUserCache(sd.options.UserCacheFilename, sd.wspInfo.TeamID, users); err != nil {
			trace.Logf(ctx, "error", "saving user cache to %q, error: %s", sd.options.UserCacheFilename, err)
			sd.l().Printf("error saving user cache to %q: %s, but nevermind, let's continue", sd.options.UserCacheFilename, err)
		}
	}

	return users, err
}

// fetchUsers fetches users from the API.
func (sd *Session) fetchUsers(ctx context.Context) (types.Users, error) {
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
func (sd *Session) loadUserCache(filename string, suffix string, maxAge time.Duration) (types.Users, error) {
	filename = sd.makeCacheFilename(filename, suffix)

	if err := checkCacheFile(filename, maxAge); err != nil {
		return nil, err
	}

	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", filename, err)
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	var uu types.Users
	if err := dec.Decode(&uu); err != nil {
		return nil, fmt.Errorf("failed to decode users from %s: %w", filename, err)
	}
	return uu, nil
}

func (sd *Session) saveUserCache(filename string, suffix string, uu types.Users) error {
	filename = sd.makeCacheFilename(filename, suffix)

	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filename, err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(uu); err != nil {
		return fmt.Errorf("failed to encode data for %s: %w", filename, err)
	}
	return nil
}

// makeCacheFilename converts filename.ext to filename-suffix.ext.
func (sd *Session) makeCacheFilename(filename, suffix string) string {
	ne := filenameSplit(filename)
	return filepath.Join(sd.cacheDir, filenameJoin(nameExt{ne[0] + "-" + suffix, ne[1]}))
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
