package slackdump

import (
	"context"
	"errors"
	"io"
	"sync"

	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v3/logger"
)

const enterpriseIsRestricted = "enterprise_is_restricted"

type fallbackMethod int

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

type fallbackClient struct {
	cl        []clienter
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
	fc.mu.Lock()
	defer fc.mu.Unlock()
	if fc.methodPtr == nil {
		fc.methodPtr = make(map[fallbackMethod]int)
	}
	if _, ok := fc.methodPtr[m]; !ok {
		fc.methodPtr[m] = 0
	}
	ptr := fc.methodPtr[m]
	if ptr >= len(fc.cl) {
		return errNoMoreFallback
	}
	fc.methodPtr[m]++
	return nil
}

func (fc *fallbackClient) getClient(m fallbackMethod) (clienter, error) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	ptr := fc.methodPtr[m]
	if ptr >= len(fc.cl) {
		return nil, errNoMoreFallback
	}
	return fc.cl[ptr], nil
}

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
	const this = fbAuthTestContext
	cl, err := fc.getClient(this)
	if err != nil {
		return nil, err
	}
	resp, err := cl.AuthTestContext(ctx)
	if err != nil {
		if isFallbackError(err) {
			err = fc.fallback(this)
			if err != nil {
				return nil, err
			}
			return fc.AuthTestContext(ctx)
		}
	}
	return resp, err
}

func (fc *fallbackClient) GetConversationHistoryContext(ctx context.Context, params *slack.GetConversationHistoryParameters) (*slack.GetConversationHistoryResponse, error) {
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
			return fc.GetConversationHistoryContext(ctx, params)
		}
	}
	return resp, err
}

func (fc *fallbackClient) GetConversationRepliesContext(ctx context.Context, params *slack.GetConversationRepliesParameters) (msgs []slack.Message, hasMore bool, nextCursor string, err error) {
	const this = fbGetConversationRepliesContext
	cl, err := fc.getClient(this)
	if err != nil {
		return nil, false, "", err
	}
	msgs, hasMore, nextCursor, err = cl.GetConversationRepliesContext(ctx, params)
	if err != nil {
		if isFallbackError(err) {
			err = fc.fallback(this)
			if err != nil {
				return nil, false, "", err
			}
			return fc.GetConversationRepliesContext(ctx, params)
		}
	}
	return msgs, hasMore, nextCursor, err
}

func (fc *fallbackClient) GetConversationsContext(ctx context.Context, params *slack.GetConversationsParameters) (channels []slack.Channel, nextCursor string, err error) {
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
			return fc.GetConversationsContext(ctx, params)
		}
	}
	return channels, nextCursor, err
}

func (fc *fallbackClient) GetStarredContext(ctx context.Context, params slack.StarsParameters) ([]slack.StarredItem, *slack.Paging, error) {
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
			return fc.GetStarredContext(ctx, params)
		}
	}
	return items, paging, err

}

// GetUserPaginated always calls the first client in the list.
func (fc *fallbackClient) GetUsersPaginated(options ...slack.GetUsersOption) slack.UserPagination {
	const this = fbGetUsersPaginated
	return fc.cl[0].GetUsersPaginated(options...)
}

func (fc *fallbackClient) ListBookmarks(channelID string) ([]slack.Bookmark, error) {
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
			return fc.ListBookmarks(channelID)
		}
	}
	return bm, err
}

func (fc *fallbackClient) GetConversationInfoContext(ctx context.Context, input *slack.GetConversationInfoInput) (*slack.Channel, error) {
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
			return fc.GetConversationInfoContext(ctx, input)
		}
	}
	return c, err
}

func (fc *fallbackClient) GetUsersInConversationContext(ctx context.Context, params *slack.GetUsersInConversationParameters) ([]string, string, error) {
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
			return fc.GetUsersInConversationContext(ctx, params)
		}
	}
	return userIDs, next, err
}

func (fc *fallbackClient) GetFile(downloadURL string, writer io.Writer) error {
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
			return fc.GetFile(downloadURL, writer)
		}
	}
	return err
}

func (fc *fallbackClient) GetUsersContext(ctx context.Context, options ...slack.GetUsersOption) ([]slack.User, error) {
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
			return fc.GetUsersContext(ctx, options...)
		}
	}
	return users, err
}

func (fc *fallbackClient) GetEmojiContext(ctx context.Context) (map[string]string, error) {
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
			return fc.GetEmojiContext(ctx)
		}
	}
	return emojis, err
}

func isFallbackError(err error) bool {
	var serr slack.SlackErrorResponse
	return errors.As(err, &serr) && serr.Err == enterpriseIsRestricted
}
