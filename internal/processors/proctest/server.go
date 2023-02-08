package proctest

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/slack-go/slack"
)

type Server struct {
	*httptest.Server
	p *Player
}

func NewServer(rs io.ReadSeeker) *Server {
	p, err := NewPlayer(rs)
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

func router(p *Player) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/conversations.history", func(w http.ResponseWriter, r *http.Request) {
		msg, err := p.Messages()
		if err != nil {
			if err == io.EOF {
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
			HasMore:  p.HasMoreMessages(),
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
		timestamp := r.FormValue("ts")
		if timestamp == "" {
			http.Error(w, "ts is required", http.StatusBadRequest)
			return
		}
		msg, err := p.Thread(timestamp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		resp := GetConversationRepliesResponse{
			HasMore:  p.HasMoreThreads(timestamp),
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

type GetConversationRepliesResponse struct {
	slack.SlackResponse
	HasMore          bool `json:"has_more"`
	ResponseMetaData struct {
		NextCursor string `json:"next_cursor"`
	} `json:"response_metadata"`
	Messages []slack.Message `json:"messages"`
}
