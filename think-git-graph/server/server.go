package server

import (
	"context"
	"encoding/json"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/dalpat/git-tools/think-git-graph/gitdata"
)

type StatusResponse struct {
	CurrentBranch  string            `json:"currentBranch"`
	HasUncommitted bool              `json:"hasUncommitted"`
	BranchStatus   *gitdata.BranchStatus `json:"branchStatus,omitempty"`
}

type GraphResponse struct {
	Commits  []gitdata.Commit `json:"commits"`
	Branches []gitdata.Branch `json:"branches"`
	Total    int              `json:"total"`
}

type CommitDetailResponse struct {
	Hash    string              `json:"hash"`
	Message string              `json:"message"`
	Author  string              `json:"author"`
	Date    string              `json:"date"`
	Files   []gitdata.FileChange `json:"files"`
}

// Server represents the HTTP server
type Server struct {
	gd            *gitdata.GitData
	httpSrv       *http.Server
	listener      net.Listener
	handler       http.Handler
	refreshCh     chan struct{}
	clients       map[chan struct{}]bool
	clientsMu     sync.RWMutex
	refreshMu     sync.Mutex
}

// New creates a new Server instance
func New(gd *gitdata.GitData, assets embed.FS) (*Server, error) {
	mux := http.NewServeMux()

	s := &Server{
		gd:        gd,
		refreshCh: make(chan struct{}, 10),
		clients:   make(map[chan struct{}]bool),
	}

	mux.HandleFunc("GET /status", s.handleStatus)
	mux.HandleFunc("GET /graph", s.handleGraph)
	mux.HandleFunc("GET /commit/{hash}", s.handleCommitDetail)
	mux.HandleFunc("GET /events", s.handleEvents)

	staticFS, err := fs.Sub(assets, "static")
	if err != nil {
		return nil, err
	}
	mux.Handle("/", http.FileServer(http.FS(staticFS)))

	s.handler = mux
	return s, nil
}

// Start starts the HTTP server
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

// Stop stops the HTTP server
func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.httpSrv.Shutdown(ctx)
}

// NotifyRefresh implements the gitdata.RefreshNotifier interface
// It broadcasts a refresh event to all connected SSE clients
func (s *Server) NotifyRefresh() {
	s.refreshMu.Lock()
	defer s.refreshMu.Unlock()

	// Notify local channel
	select {
	case s.refreshCh <- struct{}{}:
	default:
	}

	// Notify all SSE clients
	s.clientsMu.RLock()
	clients := make([]chan struct{}, 0, len(s.clients))
	for client := range s.clients {
		clients = append(clients, client)
	}
	s.clientsMu.RUnlock()

	for _, client := range clients {
		select {
		case client <- struct{}{}:
		default:
		}
	}

	log.Printf("Server: Refresh notification broadcast to %d clients", len(clients))
}

// handleEvents serves Server-Sent Events for real-time refresh notifications
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create a channel for this client
	clientCh := make(chan struct{}, 1)

	// Register client
	s.clientsMu.Lock()
	s.clients[clientCh] = true
	s.clientsMu.Unlock()

	// Unregister client when done
	defer func() {
		s.clientsMu.Lock()
		delete(s.clients, clientCh)
		s.clientsMu.Unlock()
		close(clientCh)
	}()

	// Flush the headers
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	// Send initial connection event
	fmt.Fprintf(w, "event: connected\ndata: {}\n\n")
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	// Listen for refresh events or client disconnect
	for {
		select {
		case <-clientCh:
			// Send refresh event
			fmt.Fprintf(w, "event: refresh\ndata: {}\n\n")
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		case <-r.Context().Done():
			// Client disconnected
			return
		}
	}
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

	// Get branch status for current branch
	if branch != "" {
		branchStatus, err := s.gd.GetBranchStatus(branch)
		if err == nil {
			resp.BranchStatus = branchStatus
		}
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

	branches, _ := s.gd.GetBranches()
	if branches == nil {
		branches = []gitdata.Branch{}
	}

	resp := GraphResponse{
		Commits:  commits,
		Branches: branches,
		Total:    total,
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
