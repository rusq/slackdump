package chunktest

import (
	"net/http"
	"net/http/httptest"
	"os"
	"sync"

	"github.com/rusq/slackdump/v2/internal/chunk"
)

// DirServer is a test server that serves files from a chunk.Directory.
type DirServer struct {
	baseServer
	cd *chunk.Directory

	mu   sync.Mutex
	ptrs map[string]*chunk.Player
}

func NewDirServer(dir string) *DirServer {
	cd, err := chunk.OpenDir(dir)
	if err != nil {
		panic(err)
	}
	ds := &DirServer{
		cd:   cd,
		ptrs: make(map[string]*chunk.Player),
	}
	ds.init()
	return ds
}

func (s *DirServer) init() {
	s.Server = httptest.NewServer(s.dirRouter())
}

func (s *DirServer) Close() {
	s.Server.Close()
	for _, p := range s.ptrs {
		p.Close()
	}
}

func (s *DirServer) dirRouter() *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/api/conversations.info", s.chunkWrapper(handleConversationsInfo))
	mux.Handle("/api/conversations.history", s.chunkWrapper(handleConversationsHistory))
	mux.Handle("/api/conversations.replies", s.chunkWrapper(handleConversationsReplies))

	mux.Handle("/api/conversations.list", s.chunkfileWrapper(chunk.FChannels, handleConversationsList))
	mux.Handle("/api/users.list", s.chunkfileWrapper(chunk.FUsers, handleUsersList))
	mux.Handle("/api/auth.test", s.chunkfileWrapper(chunk.FWorkspace, handleAuthTest))

	return mux
}

func (s *DirServer) chunkWrapper(fn func(p *chunk.Player) http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		channel := r.FormValue("channel")
		if channel == "" {
			http.Error(w, "no_channel", http.StatusBadRequest)
			return
		}
		s.mu.Lock()
		p, ok := s.ptrs[channel]
		s.mu.Unlock()
		if !ok {
			cf, err := s.cd.Open(channel)
			if err != nil {
				if os.IsNotExist(err) {
					http.NotFound(w, r)
					return
				}
				lg.Printf("error while opening chunk file for %s: %s", channel, err)
				http.Error(w, http.StatusText(http.StatusUnprocessableEntity), http.StatusUnprocessableEntity)
				return
			}
			p = chunk.NewPlayerFromFile(cf)
			s.mu.Lock()
			s.ptrs[channel] = p
			s.mu.Unlock()
		}
		fn(p)(w, r)
	})
}

func (s *DirServer) chunkfileWrapper(name string, fn func(p *chunk.Player) http.HandlerFunc) http.Handler {
	rs, err := s.cd.Open(name)
	if err != nil {
		panic(err)
	}
	return fn(chunk.NewPlayerFromFile(rs))
}
