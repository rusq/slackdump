package chunktest

import (
	"net/http"
	"net/http/httptest"
	"sync"

	"github.com/rusq/slackdump/v2/internal/chunk"
)

// DirServer is a test server that serves files from a chunk.Directory.
type DirServer struct {
	*httptest.Server
	cd     *chunk.Directory
	userID string

	mu   sync.Mutex
	ptrs map[string]map[chunk.GroupID]int
}

func NewDirServer(dir string, currentUserID string) *DirServer {
	cd, err := chunk.OpenDir(dir)
	if err != nil {
		panic(err)
	}
	ds := &DirServer{
		cd:     cd,
		userID: currentUserID,
		ptrs:   make(map[string]map[chunk.GroupID]int),
	}
	ds.init()
	return ds
}

func (s *DirServer) init() {
	s.Server = httptest.NewServer(s.dirRouter())
}

func (s *DirServer) dirRouter() *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/api/auth.test", authHandler{s.userID})

	mux.Handle("/api/conversations.info", s.chunkWrapper(handleConversationsInfo))
	mux.Handle("/api/conversations.history", s.chunkWrapper(handleConversationsHistory))
	mux.Handle("/api/conversations.replies", s.chunkWrapper(handleConversationsReplies))
	mux.Handle("/api/conversations.list", s.chunkWrapper(handleConversationsList))
	mux.Handle("/api/users.list", s.chunkfileWrapper("users", handleUsersList))
	return mux
}

func (s *DirServer) chunkWrapper(fn func(p *chunk.Player) http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		channel := r.FormValue("channel")
		rs, err := s.cd.Open(channel)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		p := chunk.NewPlayerFromFile(rs)

		s.mu.Lock()
		if state, ok := s.ptrs[channel]; ok {
			p.SetState(state)
		} else {
			s.ptrs[channel] = make(map[chunk.GroupID]int)
		}
		s.mu.Unlock()
		fn(p)(w, r)
		s.mu.Lock()
		s.ptrs[channel] = p.State()
		s.mu.Unlock()
	})
}

func (s *DirServer) chunkfileWrapper(name string, fn func(p *chunk.Player) http.HandlerFunc) http.Handler {
	rs, err := s.cd.Open(name)
	if err != nil {
		panic(err)
	}
	p := chunk.NewPlayerFromFile(rs)
	s.mu.Lock()
	if state, ok := s.ptrs["$"+name]; ok {
		p.SetState(state)
	} else {
		s.ptrs["$"+name] = make(map[chunk.GroupID]int)
	}
	s.mu.Unlock()

	return fn(p)
}
