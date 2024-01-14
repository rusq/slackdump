package chunktest

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime/trace"
	"strconv"

	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/slack-go/slack"
)

func router(p *chunk.Player, userID string) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/api/auth.test", authHandler{userID})

	mux.HandleFunc("/api/conversations.info", handleConversationsInfo(p))
	mux.HandleFunc("/api/conversations.members", handleConversationsMembers(p))
	mux.HandleFunc("/api/conversations.history", handleConversationsHistory(p))
	mux.HandleFunc("/api/conversations.replies", handleConversationsReplies(p))
	mux.HandleFunc("/api/conversations.list", handleConversationsList(p))
	mux.HandleFunc("/api/users.list", handleUsersList(p))
	return mux
}

type GetConversationRepliesResponse struct {
	slack.SlackResponse
	HasMore          bool             `json:"has_more"`
	ResponseMetaData responseMetaData `json:"response_metadata"`
	Messages         []slack.Message  `json:"messages"`
}

type responseMetaData struct {
	NextCursor string `json:"next_cursor"`
}

func handleConversationsHistory(p *chunk.Player) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, task := trace.NewTask(r.Context(), "conversation.history")
		defer task.End()
		channel := r.FormValue("channel")
		if channel == "" {
			http.NotFound(w, r)
			return
		}
		sresp := slack.SlackResponse{
			Ok: true,
		}

		msg, err := p.Messages(channel)
		if err != nil {
			if errors.Is(err, chunk.ErrNotFound) {
				sresp.Ok = false
				sresp.Error = fmt.Sprintf("channel_not_found[%s]", channel)
			} else if errors.Is(err, io.EOF) {
				if err := json.NewEncoder(w).Encode(slack.GetConversationHistoryResponse{
					HasMore: false,
					SlackResponse: slack.SlackResponse{
						Ok: true,
					},
				}); err != nil {
					lg.Printf("unexpected error: %s", err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
				return
			} else {
				lg.Printf("error processing messages: %s", err)
				sresp.Ok = false
				sresp.Error = fmt.Sprintf("channel: %q: error: %s", channel, err)
			}
		}
		hasmore := p.HasMoreMessages(channel)
		lg.Printf("serving channel: %s messages: %d, hasmore: %v", channel, len(msg), hasmore)
		resp := slack.GetConversationHistoryResponse{
			HasMore:          hasmore,
			Messages:         msg,
			ResponseMetaData: responseMetaData{NextCursor: strconv.FormatInt(p.Offset(), 10)},
			SlackResponse:    sresp,
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			lg.Printf("error encoding channel.history response: %s", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func handleConversationsReplies(p *chunk.Player) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, task := trace.NewTask(r.Context(), "conversation.replies")
		defer task.End()

		timestamp := r.FormValue("ts")
		channel := r.FormValue("channel")
		lg.Printf("serving channel: %s, ts: %s", channel, timestamp)

		if timestamp == "" {
			http.Error(w, "ts is required", http.StatusBadRequest)
			return
		}

		slackResp := slack.SlackResponse{
			Ok: true,
		}
		msg, err := p.Thread(channel, timestamp)
		if err != nil {
			slackResp.Ok = false
			if errors.Is(err, io.EOF) {
				slackResp.Error = fmt.Sprintf("thread_not_found[%s:%s]", channel, timestamp)
			} else {
				slackResp.Error = fmt.Sprintf("thread: [%s:%s]: error: %s", channel, timestamp, err.Error())
			}
			lg.Printf("error processing thread %s:%s: %s", channel, timestamp, err)
		}
		resp := GetConversationRepliesResponse{
			HasMore:          p.HasMoreThreads(channel, timestamp),
			Messages:         msg,
			ResponseMetaData: responseMetaData{strconv.FormatInt(p.Offset(), 10)}, // adding offset for the ease of debugging.
			SlackResponse:    slackResp,
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			lg.Printf("error encoding conversation.replies response: %s", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

type channelResponseFull struct {
	Channel      slack.Channel `json:"channel"`
	Purpose      string        `json:"purpose"`
	Topic        string        `json:"topic"`
	NotInChannel bool          `json:"not_in_channel"`
	slack.History
	slack.SlackResponse
	Metadata slack.ResponseMetadata `json:"response_metadata"`
}

func handleConversationsInfo(p *chunk.Player) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, task := trace.NewTask(r.Context(), "conversations.info")
		defer task.End()

		channel := r.FormValue("channel")
		if channel == "" {
			http.Error(w, "channel is required", http.StatusBadRequest)
			return
		}
		resp := channelResponseFull{
			SlackResponse: slack.SlackResponse{
				Ok: true,
			},
		}

		lg.Printf("channel: %s", channel)
		ci, err := p.ChannelInfo(channel)
		if err != nil {
			if errors.Is(err, chunk.ErrNotFound) {
				resp.Ok = false
				resp.Error = fmt.Sprintf("conversationInfo: not found: (%q) %v", channel, err)
			} else {
				resp.Ok = false
				resp.Error = fmt.Sprintf("conversationInfo: error: %s", err)
			}
		} else {
			resp.Channel = *ci
		}

		if err := json.NewEncoder(w).Encode(resp); err != nil {
			lg.Printf("error encoding channel.info response: %s", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

type channelResponse struct {
	Channels []slack.Channel `json:"channels"`
	slack.SlackResponse
	Metadata slack.ResponseMetadata `json:"response_metadata"`
}

func handleConversationsList(p *chunk.Player) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, task := trace.NewTask(r.Context(), "conversation.list")
		defer task.End()

		cr := channelResponse{
			Channels: []slack.Channel{},
			SlackResponse: slack.SlackResponse{
				Ok: true,
			},
			Metadata: slack.ResponseMetadata{
				Cursor: "next",
			},
		}
		c, err := p.Channels()
		if err != nil {
			cr.Ok = false
			if errors.Is(err, io.EOF) {
				cr.Metadata.Cursor = ""
			} else {
				cr.Error = err.Error()
			}
		}
		cr.Channels = c
		if err := json.NewEncoder(w).Encode(cr); err != nil {
			lg.Printf("error encoding channel.list response: %s", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

type userResponseFull struct {
	Users   []slack.User `json:"users,omitempty"`
	User    slack.User   `json:"user,omitempty"`
	Members []slack.User `json:"members"`
	slack.SlackResponse
	slack.UserPresence
	Metadata slack.ResponseMetadata `json:"response_metadata"`
}

func handleUsersList(p *chunk.Player) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, task := trace.NewTask(r.Context(), "users.list")
		defer task.End()

		sr := slack.SlackResponse{
			Ok: true,
		}
		u, err := p.Users()
		if err != nil {
			if errors.Is(err, io.EOF) {
				sr.Ok = false
				sr.Error = "pagination complete"
			} else if errors.Is(err, chunk.ErrNotFound) {
				sr.Ok = false
				sr.Error = "user chunks not found"
			} else {
				lg.Printf("error processing users.list: %s", err)
				sr.Ok = false
				sr.Error = err.Error()
			}
		}
		resp := userResponseFull{
			SlackResponse: sr,
			Members:       u,
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			lg.Printf("error encoding users.list response: %s", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

type authHandler struct {
	userID string
}

type authTestResponseFull struct {
	slack.SlackResponse
	slack.AuthTestResponse
}

func (ah authHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_, task := trace.NewTask(r.Context(), "auth.test")
	defer task.End()

	resp := authTestResponseFull{
		SlackResponse: slack.SlackResponse{
			Ok: true,
		},
		AuthTestResponse: slack.AuthTestResponse{
			Team:   "test",
			User:   "Charlie Brown",
			UserID: ah.userID,
		},
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		lg.Printf("error encoding auth.test response: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func handleAuthTest(p *chunk.Player) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		atr := authTestResponseFull{
			SlackResponse: slack.SlackResponse{
				Ok: true,
			},
		}
		wi, err := p.WorkspaceInfo()
		if err != nil {
			atr.SlackResponse.Ok = false
			atr.SlackResponse.Error = err.Error()
		} else {
			atr.AuthTestResponse = *wi
		}
		if err := json.NewEncoder(w).Encode(atr); err != nil {
			lg.Printf("error encoding auth.test response: %s", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

type conversationsMembersResp struct {
	Members []string `json:"members"`
	slack.ResponseMetadata
	slack.SlackResponse
}

func handleConversationsMembers(p *chunk.Player) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, task := trace.NewTask(r.Context(), "conversations.members")
		defer task.End()

		channel := r.FormValue("channel")
		if channel == "" {
			http.Error(w, "channel is required", http.StatusBadRequest)
			return
		}
		lg.Printf("conversations.members: channel: %s", channel)

		resp := conversationsMembersResp{
			SlackResponse: slack.SlackResponse{
				Ok: true,
			},
		}

		uu, err := p.ChannelUsers(channel)
		if err != nil {
			resp.Ok = false
			if errors.Is(err, chunk.ErrNotFound) {
				resp.Error = fmt.Sprintf("conversation.members: channel: %s, not_found", channel)
			} else {
				resp.Error = fmt.Sprintf("conversation.members: channel: %s, unexpected error: %s", channel, err)
			}
		} else {
			resp.Members = uu
		}
		if p.HasMoreChannelUsers(channel) {
			resp.Cursor = "go_ahead"
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			lg.Printf("error encoding channel.info response: %s", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
