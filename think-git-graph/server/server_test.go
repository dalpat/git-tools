package server

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/dalpat/git-tools/think-git-graph/gitdata"
)

//go:embed static/*
var testAssets embed.FS

type mockRunner struct {
	responses map[string]string
}

func (m *mockRunner) Run(args ...string) (string, error) {
	key := strings.Join(args, " ")
	if resp, ok := m.responses[key]; ok {
		if strings.HasPrefix(resp, "ERROR:") {
			return "", fmt.Errorf("%s", strings.TrimPrefix(resp, "ERROR:"))
		}
		return resp, nil
	}
	return "", fmt.Errorf("unexpected command: %s", key)
}

func TestStatusEndpoint(t *testing.T) {
	tests := []struct {
		name             string
		branchOutput     string
		statusOutput     string
		wantBranch       string
		wantHasUncommitted bool
		wantStatus       int
	}{
		{
			name:               "normal branch, clean",
			branchOutput:       "main\n",
			statusOutput:       "",
			wantBranch:         "main",
			wantHasUncommitted: false,
			wantStatus:         200,
		},
		{
			name:               "feature branch, dirty",
			branchOutput:       "feature/login\n",
			statusOutput:       " M src/main.go\n",
			wantBranch:         "feature/login",
			wantHasUncommitted: true,
			wantStatus:         200,
		},
		{
			name:               "detached HEAD",
			branchOutput:       "\n",
			statusOutput:       "",
			wantBranch:         "",
			wantHasUncommitted: false,
			wantStatus:         200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockRunner{
				responses: map[string]string{
					"branch --show-current": tt.branchOutput,
					"status --porcelain":    tt.statusOutput,
				},
			}
			gd := gitdata.New(mock)

			s, err := New(gd, testAssets)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}

			req, _ := http.NewRequest("GET", "/status", nil)
			w := &mockResponseWriter{header: make(http.Header)}
			s.handler.ServeHTTP(w, req)

			if w.statusCode != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.statusCode, tt.wantStatus)
			}

			var resp StatusResponse
			if err := json.Unmarshal(w.body.Bytes(), &resp); err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}

			if resp.CurrentBranch != tt.wantBranch {
				t.Errorf("currentBranch = %q, want %q", resp.CurrentBranch, tt.wantBranch)
			}
			if resp.HasUncommitted != tt.wantHasUncommitted {
				t.Errorf("hasUncommitted = %v, want %v", resp.HasUncommitted, tt.wantHasUncommitted)
			}
		})
	}
}

func TestStaticFileServing(t *testing.T) {
	mock := &mockRunner{
		responses: map[string]string{},
	}
	gd := gitdata.New(mock)

	s, err := New(gd, testAssets)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	req, _ := http.NewRequest("GET", "/", nil)
	w := &mockResponseWriter{header: make(http.Header)}
	s.handler.ServeHTTP(w, req)

	if w.statusCode != 200 {
		t.Errorf("status = %d, want 200", w.statusCode)
	}
	if !strings.Contains(w.body.String(), "think-git-graph") {
		t.Errorf("response does not contain expected content")
	}
}

func TestGraphEndpoint(t *testing.T) {
	commitData := "abc1234567890123456789012345678901234567\x00abc1234\x00Initial commit\x00Alice\x002024-01-15T10:30:00+00:00\x00\n" +
		"def5678901234567890123456789012345678901\x00def5678\x00Add login feature\x00Bob\x002024-01-16T14:00:00+00:00\x00abc1234567890123456789012345678901234567\n"

	branchData := "main\x00abc1234\x00abc1234567890123456789012345678901234567\n" +
		"feature/login\x00def5678\x00def5678901234567890123456789012345678901\n"

	logFormat := "--format=%H%x00%h%x00%s%x00%an%x00%aI%x00%P"
	refsFormat := "--format=%(refname:short)%x00%(objectname:short)%x00%(objectname)"

	mock := &mockRunner{
		responses: map[string]string{
			"rev-list --count HEAD": "2",
			"log " + logFormat + " --max-count=200 --skip=0": commitData,
			"for-each-ref " + refsFormat + " refs/heads/": branchData,
			"log --format=%H main":           "abc1234567890123456789012345678901234567\ndef5678901234567890123456789012345678901\n",
			"log --format=%H feature/login":  "def5678901234567890123456789012345678901\n",
		},
	}

	gd := gitdata.New(mock)
	s, err := New(gd, testAssets)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	req, _ := http.NewRequest("GET", "/graph?limit=200&offset=0", nil)
	w := &mockResponseWriter{header: make(http.Header)}
	s.handler.ServeHTTP(w, req)

	if w.statusCode != 200 {
		t.Errorf("status = %d, want 200", w.statusCode)
	}

	var resp GraphResponse
	if err := json.Unmarshal(w.body.Bytes(), &resp); err != nil {
		t.Fatalf("json.Unmarshal() error = %v: body=%s", err, w.body.String())
	}

	if resp.Total != 2 {
		t.Errorf("total = %d, want 2", resp.Total)
	}
	if len(resp.Commits) != 2 {
		t.Fatalf("len(commits) = %d, want 2", len(resp.Commits))
	}
	if resp.Commits[0].ShortHash != "abc1234" {
		t.Errorf("commits[0].ShortHash = %s, want abc1234", resp.Commits[0].ShortHash)
	}
	if resp.Commits[1].Author != "Bob" {
		t.Errorf("commits[1].Author = %s, want Bob", resp.Commits[1].Author)
	}
}

func TestGraphEndpointPagination(t *testing.T) {
	logFormat := "--format=%H%x00%h%x00%s%x00%an%x00%aI%x00%P"
	refsFormat := "--format=%(refname:short)%x00%(objectname:short)%x00%(objectname)"

	page2Data := "ghi9012345678901234567890123456789012345678\x00ghi9012\x00Third commit\x00Charlie\x002024-03-01T10:00:00+00:00\x00\n"

	mock := &mockRunner{
		responses: map[string]string{
			"rev-list --count HEAD": "5",
			"log " + logFormat + " --max-count=2 --skip=2":     page2Data,
			"for-each-ref " + refsFormat + " refs/heads/": "",
		},
	}

	gd := gitdata.New(mock)
	s, err := New(gd, testAssets)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	req, _ := http.NewRequest("GET", "/graph?limit=2&offset=2", nil)
	w := &mockResponseWriter{header: make(http.Header)}
	s.handler.ServeHTTP(w, req)

	if w.statusCode != 200 {
		t.Errorf("status = %d, want 200", w.statusCode)
	}

	var resp GraphResponse
	if err := json.Unmarshal(w.body.Bytes(), &resp); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if resp.Total != 5 {
		t.Errorf("total = %d, want 5", resp.Total)
	}
	if len(resp.Commits) != 1 {
		t.Fatalf("len(commits) = %d, want 1", len(resp.Commits))
	}
}

func TestGraphEndpointDefaults(t *testing.T) {
	logFormat := "--format=%H%x00%h%x00%s%x00%an%x00%aI%x00%P"
	refsFormat := "--format=%(refname:short)%x00%(objectname:short)%x00%(objectname)"

	mock := &mockRunner{
		responses: map[string]string{
			"rev-list --count HEAD": "0",
			"log " + logFormat + " --max-count=200 --skip=0":     "",
			"for-each-ref " + refsFormat + " refs/heads/": "",
		},
	}

	gd := gitdata.New(mock)
	s, err := New(gd, testAssets)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	req, _ := http.NewRequest("GET", "/graph", nil)
	w := &mockResponseWriter{header: make(http.Header)}
	s.handler.ServeHTTP(w, req)

	if w.statusCode != 200 {
		t.Errorf("status = %d, want 200", w.statusCode)
	}

	var resp GraphResponse
	if err := json.Unmarshal(w.body.Bytes(), &resp); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if resp.Total != 0 {
		t.Errorf("total = %d, want 0", resp.Total)
	}
	if len(resp.Commits) != 0 {
		t.Errorf("len(commits) = %d, want 0", len(resp.Commits))
	}
}
func TestStartStop(t *testing.T) {
	mock := &mockRunner{
		responses: map[string]string{
			"branch --show-current": "main\n",
			"status --porcelain":    "",
		},
	}
	gd := gitdata.New(mock)

	s, err := New(gd, testAssets)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	port, err := s.Start()
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if port <= 0 {
		t.Errorf("port = %d, want > 0", port)
	}

	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/status", port))
	if err != nil {
		t.Fatalf("http.Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var status StatusResponse
	json.Unmarshal(body, &status)

	if err := s.Stop(); err != nil {
		t.Errorf("Stop() error = %v", err)
	}
}

type mockResponseWriter struct {
	header     http.Header
	body       bytes.Buffer
	statusCode int
}

func (m *mockResponseWriter) Header() http.Header {
	return m.header
}

func (m *mockResponseWriter) Write(b []byte) (int, error) {
	if m.statusCode == 0 {
		m.statusCode = 200
	}
	return m.body.Write(b)
}

func (m *mockResponseWriter) WriteHeader(statusCode int) {
	m.statusCode = statusCode
}
