package server

import (
	"context"
	"encoding/json"
	"embed"
	"io/fs"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/dalpat/git-tools/think-git-graph/gitdata"
)

type StatusResponse struct {
	CurrentBranch  string `json:"currentBranch"`
	HasUncommitted bool   `json:"hasUncommitted"`
}

type GraphResponse struct {
	Commits []gitdata.Commit `json:"commits"`
	Total   int              `json:"total"`
}

type CommitDetailResponse struct {
	Hash    string              `json:"hash"`
	Message string              `json:"message"`
	Author  string              `json:"author"`
	Date    string              `json:"date"`
	Files   []gitdata.FileChange `json:"files"`
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
	mux.HandleFunc("GET /graph", s.handleGraph)
	mux.HandleFunc("GET /commit/{hash}", s.handleCommitDetail)

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

func (s *Server) handleGraph(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 200
	if limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 && v <= 500 {
			limit = v
		}
	}

	offset := 0
	if offsetStr != "" {
		if v, err := strconv.Atoi(offsetStr); err == nil && v >= 0 {
			offset = v
		}
	}

	commits, total, err := s.gd.GetCommits(limit, offset)
	if err != nil {
		http.Error(w, `{"error":"failed to get commits"}`, http.StatusInternalServerError)
		return
	}

	resp := GraphResponse{
		Commits: commits,
		Total:   total,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleCommitDetail(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("hash")
	if hash == "" {
		http.Error(w, `{"error":"missing hash"}`, http.StatusBadRequest)
		return
	}

	detail, err := s.gd.GetCommitDetail(hash)
	if err != nil {
		http.Error(w, `{"error":"commit not found"}`, http.StatusNotFound)
		return
	}
	if detail == nil {
		http.Error(w, `{"error":"commit not found"}`, http.StatusNotFound)
		return
	}

	resp := CommitDetailResponse{
		Hash:    detail.Hash,
		Message: detail.Message,
		Author:  detail.Author,
		Date:    detail.Date,
		Files:   detail.Files,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
