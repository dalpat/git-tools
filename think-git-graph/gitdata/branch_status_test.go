package gitdata

import (
	"errors"
	"strings"
	"testing"
	"time"
)

type MockRunner struct {
	responses map[string]string
	errors    map[string]error
}

func NewMockRunner() *MockRunner {
	return &MockRunner{
		responses: make(map[string]string),
		errors:    make(map[string]error),
	}
}

func (m *MockRunner) Run(args ...string) (string, error) {
	key := strings.Join(args, " ")
	if err, ok := m.errors[key]; ok {
		return "", err
	}
	if resp, ok := m.responses[key]; ok {
		return resp, nil
	}
	return "", errors.New("unexpected command: " + key)
}

func (m *MockRunner) SetResponse(args []string, response string) {
	key := strings.Join(args, " ")
	m.responses[key] = response
}

func (m *MockRunner) SetError(args []string, err error) {
	key := strings.Join(args, " ")
	m.errors[key] = err
}

func TestGetAheadBehind(t *testing.T) {
	tests := []struct {
		name          string
		branch        string
		base          string
		setupMock     func(*MockRunner)
		wantAhead     int
		wantBehind    int
		wantErr       bool
	}{
		{
			name:   "branch ahead of main",
			branch: "feature-branch",
			base:   "main",
			setupMock: func(m *MockRunner) {
				m.SetResponse([]string{"rev-parse", "--verify", "main"}, "abc123\n")
				m.SetResponse([]string{"rev-list", "--count", "main..feature-branch"}, "5\n")
				m.SetResponse([]string{"rev-list", "--count", "feature-branch..main"}, "0\n")
			},
			wantAhead:  5,
			wantBehind: 0,
			wantErr:    false,
		},
		{
			name:   "branch behind main",
			branch: "feature-branch",
			base:   "main",
			setupMock: func(m *MockRunner) {
				m.SetResponse([]string{"rev-parse", "--verify", "main"}, "abc123\n")
				m.SetResponse([]string{"rev-list", "--count", "main..feature-branch"}, "0\n")
				m.SetResponse([]string{"rev-list", "--count", "feature-branch..main"}, "3\n")
			},
			wantAhead:  0,
			wantBehind: 3,
			wantErr:    false,
		},
		{
			name:   "branch ahead and behind main",
			branch: "feature-branch",
			base:   "main",
			setupMock: func(m *MockRunner) {
				m.SetResponse([]string{"rev-parse", "--verify", "main"}, "abc123\n")
				m.SetResponse([]string{"rev-list", "--count", "main..feature-branch"}, "2\n")
				m.SetResponse([]string{"rev-list", "--count", "feature-branch..main"}, "4\n")
			},
			wantAhead:  2,
			wantBehind: 4,
			wantErr:    false,
		},
		{
			name:   "base branch does not exist",
			branch: "feature-branch",
			base:   "nonexistent",
			setupMock: func(m *MockRunner) {
				m.SetError([]string{"rev-parse", "--verify", "nonexistent"}, errors.New("exit status 128"))
			},
			wantAhead:  0,
			wantBehind: 0,
			wantErr:    true,
		},
		{
			name:   "up to date with main",
			branch: "feature-branch",
			base:   "main",
			setupMock: func(m *MockRunner) {
				m.SetResponse([]string{"rev-parse", "--verify", "main"}, "abc123\n")
				m.SetResponse([]string{"rev-list", "--count", "main..feature-branch"}, "0\n")
				m.SetResponse([]string{"rev-list", "--count", "feature-branch..main"}, "0\n")
			},
			wantAhead:  0,
			wantBehind: 0,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockRunner()
			tt.setupMock(mock)
			gd := New(mock)

			ahead, behind, err := gd.getAheadBehind(tt.branch, tt.base)

			if (err != nil) != tt.wantErr {
				t.Errorf("getAheadBehind() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if ahead != tt.wantAhead {
				t.Errorf("getAheadBehind() ahead = %v, want %v", ahead, tt.wantAhead)
			}
			if behind != tt.wantBehind {
				t.Errorf("getAheadBehind() behind = %v, want %v", behind, tt.wantBehind)
			}
		})
	}
}

func TestGetRemoteTrackingBranch(t *testing.T) {
	tests := []struct {
		name           string
		branchName     string
		setupMock      func(*MockRunner)
		wantRemote     string
		wantErr        bool
	}{
		{
			name:       "branch with remote tracking",
			branchName: "feature-branch",
			setupMock: func(m *MockRunner) {
				m.SetResponse([]string{"config", "--get", "branch.feature-branch.remote"}, "origin\n")
				m.SetResponse([]string{"config", "--get", "branch.feature-branch.merge"}, "refs/heads/feature-branch\n")
			},
			wantRemote: "origin/feature-branch",
			wantErr:    false,
		},
		{
			name:       "branch with different merge ref",
			branchName: "local-branch",
			setupMock: func(m *MockRunner) {
				m.SetResponse([]string{"config", "--get", "branch.local-branch.remote"}, "upstream\n")
				m.SetResponse([]string{"config", "--get", "branch.local-branch.merge"}, "refs/heads/main\n")
			},
			wantRemote: "upstream/main",
			wantErr:    false,
		},
		{
			name:       "branch without remote tracking",
			branchName: "no-remote",
			setupMock: func(m *MockRunner) {
				m.SetError([]string{"config", "--get", "branch.no-remote.remote"}, errors.New("exit status 1"))
			},
			wantRemote: "",
			wantErr:    true,
		},
		{
			name:       "branch with remote but no merge ref",
			branchName: "partial-remote",
			setupMock: func(m *MockRunner) {
				m.SetResponse([]string{"config", "--get", "branch.partial-remote.remote"}, "origin\n")
				m.SetError([]string{"config", "--get", "branch.partial-remote.merge"}, errors.New("exit status 1"))
			},
			wantRemote: "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockRunner()
			tt.setupMock(mock)
			gd := New(mock)

			remote, err := gd.getRemoteTrackingBranch(tt.branchName)

			if (err != nil) != tt.wantErr {
				t.Errorf("getRemoteTrackingBranch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if remote != tt.wantRemote {
				t.Errorf("getRemoteTrackingBranch() = %v, want %v", remote, tt.wantRemote)
			}
		})
	}
}

func TestIsBranchMergedToMain(t *testing.T) {
	tests := []struct {
		name       string
		branchName string
		setupMock  func(*MockRunner)
		wantMerged bool
		wantErr    bool
	}{
		{
			name:       "branch is merged to main",
			branchName: "merged-branch",
			setupMock: func(m *MockRunner) {
				m.SetResponse([]string{"rev-parse", "--verify", "main"}, "abc123\n")
				m.SetResponse([]string{"branch", "--merged", "main"}, "  main\n  merged-branch\n  another-branch\n")
			},
			wantMerged: true,
			wantErr:    false,
		},
		{
			name:       "branch is not merged to main",
			branchName: "unmerged-branch",
			setupMock: func(m *MockRunner) {
				m.SetResponse([]string{"rev-parse", "--verify", "main"}, "abc123\n")
				m.SetResponse([]string{"branch", "--merged", "main"}, "  main\n  merged-branch\n")
			},
			wantMerged: false,
			wantErr:    false,
		},
		{
			name:       "current branch marked with asterisk",
			branchName: "current-branch",
			setupMock: func(m *MockRunner) {
				m.SetResponse([]string{"rev-parse", "--verify", "main"}, "abc123\n")
				m.SetResponse([]string{"branch", "--merged", "main"}, "* current-branch\n  main\n")
			},
			wantMerged: true,
			wantErr:    false,
		},
		{
			name:       "uses origin/main when main does not exist",
			branchName: "feature-branch",
			setupMock: func(m *MockRunner) {
				m.SetError([]string{"rev-parse", "--verify", "main"}, errors.New("exit status 128"))
				m.SetResponse([]string{"rev-parse", "--verify", "origin/main"}, "def456\n")
				m.SetResponse([]string{"branch", "--merged", "origin/main"}, "  main\n  feature-branch\n")
			},
			wantMerged: true,
			wantErr:    false,
		},
		{
			name:       "error when neither main nor origin/main exists",
			branchName: "feature-branch",
			setupMock: func(m *MockRunner) {
				m.SetError([]string{"rev-parse", "--verify", "main"}, errors.New("exit status 128"))
				m.SetError([]string{"rev-parse", "--verify", "origin/main"}, errors.New("exit status 128"))
			},
			wantMerged: false,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockRunner()
			tt.setupMock(mock)
			gd := New(mock)

			merged, err := gd.isBranchMergedToMain(tt.branchName)

			if (err != nil) != tt.wantErr {
				t.Errorf("isBranchMergedToMain() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if merged != tt.wantMerged {
				t.Errorf("isBranchMergedToMain() = %v, want %v", merged, tt.wantMerged)
			}
		})
	}
}

func TestGetBranchAge(t *testing.T) {
	now := time.Now()
	oneHourAgo := now.Add(-1 * time.Hour).Format(time.RFC3339)
	oneDayAgo := now.Add(-24 * time.Hour).Format(time.RFC3339)
	oneMonthAgo := now.Add(-30 * 24 * time.Hour).Format(time.RFC3339)
	oneYearAgo := now.Add(-365 * 24 * time.Hour).Format(time.RFC3339)

	tests := []struct {
		name       string
		branchName string
		setupMock  func(*MockRunner)
		wantAge    string
		wantErr    bool
	}{
		{
			name:       "branch with recent commit",
			branchName: "feature-branch",
			setupMock: func(m *MockRunner) {
				m.SetResponse([]string{"log", "-1", "--format=%aI", "feature-branch"}, oneHourAgo+"\n")
			},
			wantAge: "1h",
			wantErr: false,
		},
		{
			name:       "branch with day old commit",
			branchName: "feature-branch",
			setupMock: func(m *MockRunner) {
				m.SetResponse([]string{"log", "-1", "--format=%aI", "feature-branch"}, oneDayAgo+"\n")
			},
			wantAge: "1d",
			wantErr: false,
		},
		{
			name:       "branch with month old commit",
			branchName: "old-branch",
			setupMock: func(m *MockRunner) {
				m.SetResponse([]string{"log", "-1", "--format=%aI", "old-branch"}, oneMonthAgo+"\n")
			},
			wantAge: "30d",
			wantErr: false,
		},
		{
			name:       "branch with year old commit",
			branchName: "ancient-branch",
			setupMock: func(m *MockRunner) {
				m.SetResponse([]string{"log", "-1", "--format=%aI", "ancient-branch"}, oneYearAgo+"\n")
			},
			wantAge: "12mo",
			wantErr: false,
		},
		{
			name:       "error getting branch age",
			branchName: "nonexistent",
			setupMock: func(m *MockRunner) {
				m.SetError([]string{"log", "-1", "--format=%aI", "nonexistent"}, errors.New("exit status 128"))
			},
			wantAge: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockRunner()
			tt.setupMock(mock)
			gd := New(mock)

			_, age, err := gd.getBranchAge(tt.branchName)

			if (err != nil) != tt.wantErr {
				t.Errorf("getBranchAge() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if age != tt.wantAge {
				t.Errorf("getBranchAge() age = %v, want %v", age, tt.wantAge)
			}
		})
	}
}

func TestFormatAge(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{"just now", 0, "now"},
		{"30 seconds", 30 * time.Second, "now"},
		{"5 minutes", 5 * time.Minute, "5m"},
		{"1 hour", 1 * time.Hour, "1h"},
		{"5 hours", 5 * time.Hour, "5h"},
		{"1 day", 24 * time.Hour, "1d"},
		{"5 days", 5 * 24 * time.Hour, "5d"},
		{"30 days", 30 * 24 * time.Hour, "30d"},
		{"60 days", 60 * 24 * time.Hour, "2mo"},
		{"365 days", 365 * 24 * time.Hour, "12mo"},
		{"730 days", 730 * 24 * time.Hour, "2y"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatAge(tt.duration)
			if got != tt.want {
				t.Errorf("formatAge() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetBranchStatus(t *testing.T) {
	now := time.Now()
	commitTime := now.Add(-2 * time.Hour).Format(time.RFC3339)

	mock := NewMockRunner()
	// Setup for ahead/behind vs main
	mock.SetResponse([]string{"rev-parse", "--verify", "main"}, "abc123\n")
	mock.SetResponse([]string{"rev-list", "--count", "main..feature-branch"}, "3\n")
	mock.SetResponse([]string{"rev-list", "--count", "feature-branch..main"}, "1\n")

	// Setup for remote tracking
	mock.SetResponse([]string{"config", "--get", "branch.feature-branch.remote"}, "origin\n")
	mock.SetResponse([]string{"config", "--get", "branch.feature-branch.merge"}, "refs/heads/feature-branch\n")
	mock.SetResponse([]string{"rev-parse", "--verify", "origin/feature-branch"}, "abc123\n")
	mock.SetResponse([]string{"rev-list", "--count", "origin/feature-branch..feature-branch"}, "2\n")
	mock.SetResponse([]string{"rev-list", "--count", "feature-branch..origin/feature-branch"}, "0\n")

	// Setup for merge status
	mock.SetResponse([]string{"branch", "--merged", "main"}, "  main\n  other-branch\n")

	// Setup for branch age
	mock.SetResponse([]string{"log", "-1", "--format=%aI", "feature-branch"}, commitTime+"\n")

	gd := New(mock)
	status, err := gd.GetBranchStatus("feature-branch")

	if err != nil {
		t.Errorf("GetBranchStatus() unexpected error = %v", err)
		return
	}

	if status.Name != "feature-branch" {
		t.Errorf("GetBranchStatus() name = %v, want feature-branch", status.Name)
	}
	if status.AheadOfMain != 3 {
		t.Errorf("GetBranchStatus() aheadOfMain = %v, want 3", status.AheadOfMain)
	}
	if status.BehindMain != 1 {
		t.Errorf("GetBranchStatus() behindMain = %v, want 1", status.BehindMain)
	}
	if status.AheadOfRemote != 2 {
		t.Errorf("GetBranchStatus() aheadOfRemote = %v, want 2", status.AheadOfRemote)
	}
	if status.BehindRemote != 0 {
		t.Errorf("GetBranchStatus() behindRemote = %v, want 0", status.BehindRemote)
	}
	if status.IsMergedToMain {
		t.Error("GetBranchStatus() isMergedToMain = true, want false")
	}
	if status.Age != "2h" {
		t.Errorf("GetBranchStatus() age = %v, want 2h", status.Age)
	}
	if status.RemoteTracking != "origin/feature-branch" {
		t.Errorf("GetBranchStatus() remoteTracking = %v, want origin/feature-branch", status.RemoteTracking)
	}
}

func TestGetBranchStatus_FallbackToOriginMain(t *testing.T) {
	now := time.Now()
	commitTime := now.Add(-2 * time.Hour).Format(time.RFC3339)

	mock := NewMockRunner()
	// Main doesn't exist, but origin/main does
	mock.SetError([]string{"rev-parse", "--verify", "main"}, errors.New("exit status 128"))
	mock.SetResponse([]string{"rev-parse", "--verify", "origin/main"}, "def456\n")
	mock.SetResponse([]string{"rev-list", "--count", "origin/main..feature-branch"}, "5\n")
	mock.SetResponse([]string{"rev-list", "--count", "feature-branch..origin/main"}, "2\n")

	// No remote tracking
	mock.SetError([]string{"config", "--get", "branch.feature-branch.remote"}, errors.New("exit status 1"))

	// Merge status with origin/main
	mock.SetResponse([]string{"branch", "--merged", "origin/main"}, "  main\n  feature-branch\n")

	// Branch age
	mock.SetResponse([]string{"log", "-1", "--format=%aI", "feature-branch"}, commitTime+"\n")

	gd := New(mock)
	status, err := gd.GetBranchStatus("feature-branch")

	if err != nil {
		t.Errorf("GetBranchStatus() unexpected error = %v", err)
		return
	}

	if status.AheadOfMain != 5 {
		t.Errorf("GetBranchStatus() aheadOfMain = %v, want 5", status.AheadOfMain)
	}
	if status.BehindMain != 2 {
		t.Errorf("GetBranchStatus() behindMain = %v, want 2", status.BehindMain)
	}
	if !status.IsMergedToMain {
		t.Error("GetBranchStatus() isMergedToMain = false, want true")
	}
}
