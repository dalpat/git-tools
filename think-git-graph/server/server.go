package server

import (
	"context"
	"encoding/json"
	"embed"
	"io/fs"
	"net"
	"net/http"
	"time"

	"github.com/dalpat/git-tools/think-git-graph/gitdata"
)

type StatusResponse struct {
	CurrentBranch string `json:"currentBranch"`
	HasUncommitted bool  `json:"hasUncommitted"`
}

type Server struct {
	gd       *gitdata.GitData
	httpSrv  *http.Server
	listener net.Listener
	handler  http.Handler
}

func New(gd *gitdata.GitData, assets embed.FS) (*Server, error) {
	mux := http.NewServeMux()

	s := &Server{
		gd: gd,
	}

	mux.HandleFunc("GET /status", s.handleStatus)

	staticFS, err := fs.Sub(assets, "static")
	if err != nil {
		return nil, err
	}
	mux.Handle("/", http.FileServer(http.FS(staticFS)))

	s.handler = mux
	return s, nil
}

func (s *Server) Start() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	s.listener = listener

	s.httpSrv = &http.Server{
		Handler: s.handler,
	}

	go s.httpSrv.Serve(listener)

	return listener.Addr().(*net.TCPAddr).Port, nil
}

func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.httpSrv.Shutdown(ctx)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	branch, err := s.gd.GetCurrentBranch()
	if err != nil {
		branch = ""
	}

	hasUncommitted, _ := s.gd.HasUncommitted()

	resp := StatusResponse{
		CurrentBranch: branch,
		HasUncommitted: hasUncommitted,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
