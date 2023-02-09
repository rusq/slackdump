package proctest

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime/trace"

	"github.com/rusq/slackdump/v2/internal/processors"
	"github.com/slack-go/slack"
)

type Server struct {
	*httptest.Server
	p *processors.Player
}

func NewServer(rs io.ReadSeeker) *Server {
	p, err := processors.NewPlayer(rs)
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
	HasMore          bool `json:"has_more"`
	ResponseMetaData struct {
		NextCursor string `json:"next_cursor"`
	} `json:"response_metadata"`
	Messages []slack.Message `json:"messages"`
}

func router(p *processors.Player) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/conversations.history", func(w http.ResponseWriter, r *http.Request) {
		_, task := trace.NewTask(r.Context(), "conversation.history")
		defer task.End()

		channel := r.FormValue("channel")
		if channel == "" {
			http.NotFound(w, r)
			return
		}

		msg, err := p.Messages(channel)
		if err != nil {
			if errors.Is(err, processors.ErrNotFound) {
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
			HasMore:  p.HasMoreMessages(channel),
			Messages: msg,
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

		if timestamp == "" {
			http.Error(w, "ts is required", http.StatusBadRequest)
			return
		}
		msg, err := p.Thread(channel, timestamp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		resp := GetConversationRepliesResponse{
			HasMore:  p.HasMoreThreads(channel, timestamp),
			Messages: msg,
			SlackResponse: slack.SlackResponse{
				Ok: true,
			},
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
	return mux
}
