package proctest

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"runtime/trace"
	"strconv"

	"github.com/rusq/slackdump/v2/internal/event"
	"github.com/slack-go/slack"
)

type Server struct {
	*httptest.Server
	p *event.Player
}

func NewServer(rs io.ReadSeeker) *Server {
	p, err := event.NewPlayer(rs)
	if err != nil {
		panic(err)
	}
	return &Server{
		Server: httptest.NewServer(router(p)),
		p:      p,
	}
}

func (s *Server) Close() {
	s.Server.Close()
}

type GetConversationRepliesResponse struct {
	slack.SlackResponse
	HasMore          bool             `json:"has_more"`
	ResponseMetaData responseMetaData `json:"response_metadata"`
	Messages         []slack.Message  `json:"messages"`
}

func router(p *event.Player) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/conversations.history", func(w http.ResponseWriter, r *http.Request) {
		_, task := trace.NewTask(r.Context(), "conversation.history")
		defer task.End()

		channel := r.FormValue("channel")
		if channel == "" {
			http.NotFound(w, r)
			return
		}
		log.Printf("channel: %s", channel)

		msg, err := p.Messages(channel)
		if err != nil {
			if errors.Is(err, event.ErrNotFound) {
				http.NotFound(w, r)
				return
			}
			if errors.Is(err, io.EOF) {
				if err := json.NewEncoder(w).Encode(slack.GetConversationHistoryResponse{
					HasMore: false,
					SlackResponse: slack.SlackResponse{
						Ok: true,
					},
				}); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		resp := slack.GetConversationHistoryResponse{
			HasMore:          p.HasMoreMessages(channel),
			Messages:         msg,
			ResponseMetaData: responseMetaData{NextCursor: strconv.FormatInt(p.Offset(), 10)},
			SlackResponse: slack.SlackResponse{
				Ok: true,
			},
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
	mux.HandleFunc("/api/conversations.replies", func(w http.ResponseWriter, r *http.Request) {
		_, task := trace.NewTask(r.Context(), "conversation.replies")
		defer task.End()

		timestamp := r.FormValue("ts")
		channel := r.FormValue("channel")
		log.Printf("channel: %s, ts: %s", channel, timestamp)

		if timestamp == "" {
			http.Error(w, "ts is required", http.StatusBadRequest)
			return
		}

		var slackResp = slack.SlackResponse{
			Ok: true,
		}
		msg, err := p.Thread(channel, timestamp)
		if err != nil {
			slackResp.Ok = false
			if errors.Is(err, io.EOF) {
				slackResp.Error = fmt.Sprintf("thread_not_found[%s:%s]", channel, timestamp)
			} else {
				slackResp.Error = err.Error()
			}
		}
		resp := GetConversationRepliesResponse{
			HasMore:          p.HasMoreThreads(channel, timestamp),
			Messages:         msg,
			ResponseMetaData: responseMetaData{strconv.FormatInt(p.Offset(), 10)}, // adding offset for the ease of debugging.
			SlackResponse:    slackResp,
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
	return mux
}

type responseMetaData struct {
	NextCursor string `json:"next_cursor"`
}
