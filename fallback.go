package slackdump

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v3/logger"
)

const enterpriseIsRestricted = "enterprise_is_restricted"

// fallbackMethod is the type of the fallback method, it is used as the key
// in the map with current method pointers.
//
//go:generate stringer -type=fallbackMethod -trimprefix=fb
type fallbackMethod int

// method keys
const (
	fbAuthTestContext fallbackMethod = iota
	fbGetConversationHistoryContext
	fbGetConversationRepliesContext
	fbGetConversationsContext
	fbGetStarredContext
	fbGetUsersPaginated
	fbListBookmarks
	fbGetConversationInfo
	fbGetUsersInConversation
	fbGetFile
	fbGetUsersContext
	fbGetEmojiContext
)

var _ clienter = (*fallbackClient)(nil)

// fallbackClient is the clienter, that implements the fallback if the
// underlying function returns a "fallbackable" error.  It is used to chain
// multiple clients together, so that if the first client fails, the next one
// is used.
//
// There is a mutex to ensure that the fallback is thread-safe.  The pattern
// for implementing a function is - if the function is exported, it must lock
// the mutex (and unlock, after it's done).  All the logic should be
// implemented in the unexported function with the same name, but no mutex
// operations.  This ensures that the mutex is always locked and unlocked
// correctly, and no deadlocks occur, as it is locked only once per call.
// Using mutex also ensures that two parallel goroutines don't cause the
// function pointer advanced more than once.
//
// The logic is the following: if the function returns an error, and the error
// is of the type that a fallback is applicable to (see isFallbackError), the
// method pointer is advanced to the next client in the list, and the function
// is called again.  If the method pointer is already at the end of the list,
// the error is returned.
//
// The fallbackClient is used in the following way:
//
//	fc := newFallbackClient(ctx, mainClient, fallbackClient1, fallbackClient2)
//	resp, err := fc.GetConversationInfoContext(ctx, params)
//
// In the example, the GetConversationInfoContext will be called on the
// mainClient, if it fails with enterprise_is_restricted, the
// GetConversationInfoContext will be called on the fallbackClient1, and so
// on.
type fallbackClient struct {
	// cl is a slice of clients, the first one is the main client, the rest
	// are fallback clients.
	cl []clienter
	// methodPtr is a map with the current method pointers, it is used to
	// determine which client to use next for the given method.
	methodPtr map[fallbackMethod]int
	mu        sync.Mutex
	lg        logger.Interface
}

// newFallbackClient creates a new fallback client, it requires a main client.
func newFallbackClient(ctx context.Context, main clienter, cl ...clienter) *fallbackClient {
	return &fallbackClient{
		cl: append([]clienter{main}, cl...),
		methodPtr: map[fallbackMethod]int{
			fbGetConversationInfo:    0,
			fbGetUsersInConversation: 0,
		},
		lg: logger.FromContext(ctx),
	}
}

var errNoMoreFallback = errors.New("no more fallbacks")

func (fc *fallbackClient) fallback(m fallbackMethod) error {
	if fc.methodPtr == nil {
		fc.methodPtr = make(map[fallbackMethod]int)
	}
	ptr := fc.methodPtr[m] // returns 0 if empty.
	if ptr+1 >= len(fc.cl) {
		return fmt.Errorf("%w: %s", errNoMoreFallback, m)
	}
	fc.methodPtr[m]++
	fc.lg.Printf("falling back on %s, %d -> %d", m, ptr, ptr+1)
	return nil
}

func (fc *fallbackClient) getClient(m fallbackMethod) (clienter, error) {
	ptr := fc.methodPtr[m]
	if ptr >= len(fc.cl) {
		return nil, fmt.Errorf("%w: %s", errNoMoreFallback, m)
	}
	fc.lg.Debugf("current method %s[%d]", m, ptr)
	return fc.cl[ptr], nil
}

// Client returns a *slack.Client or panics if no clienter is *slack.Client
func (fc *fallbackClient) Client() *slack.Client {
	// ugly motherfucker
	switch v := fc.cl[0].(type) {
	case *fallbackClient:
		return v.cl[0].(*slack.Client)
	case *slack.Client:
		return v
	default:
		panic("unable to determine the client type")
	}
}

func (fc *fallbackClient) AuthTestContext(ctx context.Context) (*slack.AuthTestResponse, error) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	return fc.authTestContext(ctx)
}
func (fc *fallbackClient) authTestContext(ctx context.Context) (*slack.AuthTestResponse, error) {
	const this = fbAuthTestContext
	cl, err := fc.getClient(this)
	if err != nil {
		return nil, err
	}
	resp, err := cl.AuthTestContext(ctx)
	if err != nil {
		if isFallbackError(err) {
			if err := fc.fallback(this); err != nil {
				return nil, err
			}
			return fc.authTestContext(ctx)
		}
	}
	return resp, err
}

func (fc *fallbackClient) GetConversationHistoryContext(ctx context.Context, params *slack.GetConversationHistoryParameters) (*slack.GetConversationHistoryResponse, error) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	return fc.getConversationHistoryContext(ctx, params)
}
func (fc *fallbackClient) getConversationHistoryContext(ctx context.Context, params *slack.GetConversationHistoryParameters) (*slack.GetConversationHistoryResponse, error) {
	const this = fbGetConversationHistoryContext
	cl, err := fc.getClient(this)
	if err != nil {
		return nil, err
	}
	resp, err := cl.GetConversationHistoryContext(ctx, params)
	if err != nil {
		if isFallbackError(err) {
			err = fc.fallback(this)
			if err != nil {
				return nil, err
			}
			return fc.getConversationHistoryContext(ctx, params)
		}
	}
	return resp, err
}

func (fc *fallbackClient) GetConversationRepliesContext(ctx context.Context, params *slack.GetConversationRepliesParameters) (msgs []slack.Message, hasMore bool, nextCursor string, err error) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	return fc.getConversationRepliesContext(ctx, params)
}
func (fc *fallbackClient) getConversationRepliesContext(ctx context.Context, params *slack.GetConversationRepliesParameters) (msgs []slack.Message, hasMore bool, nextCursor string, err error) {
	const this = fbGetConversationRepliesContext
	cl, err := fc.getClient(this)
	if err != nil {
		return nil, false, "", err
	}
	msgs, hasMore, nextCursor, err = cl.GetConversationRepliesContext(ctx, params)
	if err != nil {
		if isFallbackError(err) {
			if err = fc.fallback(this); err != nil {
				return nil, false, "", err
			}
			return fc.getConversationRepliesContext(ctx, params)
		}
	}
	return msgs, hasMore, nextCursor, err
}

func (fc *fallbackClient) GetConversationsContext(ctx context.Context, params *slack.GetConversationsParameters) (channels []slack.Channel, nextCursor string, err error) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	return fc.getConversationsContext(ctx, params)
}
func (fc *fallbackClient) getConversationsContext(ctx context.Context, params *slack.GetConversationsParameters) (channels []slack.Channel, nextCursor string, err error) {
	const this = fbGetConversationsContext
	cl, err := fc.getClient(this)
	if err != nil {
		return nil, "", err
	}
	channels, nextCursor, err = cl.GetConversationsContext(ctx, params)
	if err != nil {
		if isFallbackError(err) {
			err = fc.fallback(this)
			if err != nil {
				return nil, "", err
			}
			return fc.getConversationsContext(ctx, params)
		}
	}
	return channels, nextCursor, err
}

func (fc *fallbackClient) GetStarredContext(ctx context.Context, params slack.StarsParameters) ([]slack.StarredItem, *slack.Paging, error) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	return fc.getStarredContext(ctx, params)
}
func (fc *fallbackClient) getStarredContext(ctx context.Context, params slack.StarsParameters) ([]slack.StarredItem, *slack.Paging, error) {
	const this = fbGetStarredContext
	cl, err := fc.getClient(this)
	if err != nil {
		return nil, nil, err
	}
	items, paging, err := cl.GetStarredContext(ctx, params)
	if err != nil {
		if isFallbackError(err) {
			err = fc.fallback(this)
			if err != nil {
				return nil, nil, err
			}
			return fc.getStarredContext(ctx, params)
		}
	}
	return items, paging, err

}

// GetUserPaginated always calls the first client in the list.
func (fc *fallbackClient) GetUsersPaginated(options ...slack.GetUsersOption) slack.UserPagination {
	return fc.cl[0].GetUsersPaginated(options...)
}

func (fc *fallbackClient) ListBookmarks(channelID string) ([]slack.Bookmark, error) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	return fc.listBookmarks(channelID)
}
func (fc *fallbackClient) listBookmarks(channelID string) ([]slack.Bookmark, error) {
	const this = fbListBookmarks
	cl, err := fc.getClient(this)
	if err != nil {
		return nil, err
	}
	bm, err := cl.ListBookmarks(channelID)
	if err != nil {
		if isFallbackError(err) {
			if err = fc.fallback(this); err != nil {
				return nil, err
			}
			return fc.listBookmarks(channelID)
		}
	}
	return bm, err
}

func (fc *fallbackClient) GetConversationInfoContext(ctx context.Context, input *slack.GetConversationInfoInput) (*slack.Channel, error) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	return fc.getConversationInfoContext(ctx, input)
}
func (fc *fallbackClient) getConversationInfoContext(ctx context.Context, input *slack.GetConversationInfoInput) (*slack.Channel, error) {
	const this = fbGetConversationInfo
	cl, err := fc.getClient(this)
	if err != nil {
		return nil, err
	}
	c, err := cl.GetConversationInfoContext(ctx, input)
	if err != nil {
		if isFallbackError(err) {
			err = fc.fallback(this)
			if err != nil {
				return nil, err
			}
			return fc.getConversationInfoContext(ctx, input)
		}
	}
	return c, err
}

func (fc *fallbackClient) GetUsersInConversationContext(ctx context.Context, params *slack.GetUsersInConversationParameters) ([]string, string, error) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	return fc.getUsersInConversationContext(ctx, params)
}
func (fc *fallbackClient) getUsersInConversationContext(ctx context.Context, params *slack.GetUsersInConversationParameters) ([]string, string, error) {
	const this = fbGetConversationInfo
	cl, err := fc.getClient(this)
	if err != nil {
		return nil, "", err
	}
	userIDs, next, err := cl.GetUsersInConversationContext(ctx, params)
	if err != nil {
		if isFallbackError(err) {
			err = fc.fallback(this)
			if err != nil {
				return nil, "", err
			}
			return fc.getUsersInConversationContext(ctx, params)
		}
	}
	return userIDs, next, err
}

func (fc *fallbackClient) GetFile(downloadURL string, writer io.Writer) error {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	return fc.getFile(downloadURL, writer)
}
func (fc *fallbackClient) getFile(downloadURL string, writer io.Writer) error {
	const this = fbGetFile
	cl, err := fc.getClient(this)
	if err != nil {
		return err
	}
	err = cl.GetFile(downloadURL, writer)
	if err != nil {
		if isFallbackError(err) {
			err = fc.fallback(this)
			if err != nil {
				return err
			}
			return fc.getFile(downloadURL, writer)
		}
	}
	return err
}

func (fc *fallbackClient) GetUsersContext(ctx context.Context, options ...slack.GetUsersOption) ([]slack.User, error) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	return fc.getUsersContext(ctx, options...)
}
func (fc *fallbackClient) getUsersContext(ctx context.Context, options ...slack.GetUsersOption) ([]slack.User, error) {
	const this = fbGetUsersContext
	cl, err := fc.getClient(this)
	if err != nil {
		return nil, err
	}
	users, err := cl.GetUsersContext(ctx, options...)
	if err != nil {
		if isFallbackError(err) {
			err = fc.fallback(this)
			if err != nil {
				return nil, err
			}
			return fc.getUsersContext(ctx, options...)
		}
	}
	return users, err
}

func (fc *fallbackClient) GetEmojiContext(ctx context.Context) (map[string]string, error) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	return fc.getEmojiContext(ctx)
}
func (fc *fallbackClient) getEmojiContext(ctx context.Context) (map[string]string, error) {
	const this = fbGetEmojiContext
	cl, err := fc.getClient(this)
	if err != nil {
		return nil, err
	}
	emojis, err := cl.GetEmojiContext(ctx)
	if err != nil {
		if isFallbackError(err) {
			err = fc.fallback(this)
			if err != nil {
				return nil, err
			}
			return fc.getEmojiContext(ctx)
		}
	}
	return emojis, err
}

func isFallbackError(err error) bool {
	logger.Default.Printf("isFallbackError type: %[1]T, error: %[1]v", err)
	var serr slack.SlackErrorResponse
	return errors.As(err, &serr) && serr.Err == enterpriseIsRestricted
}
