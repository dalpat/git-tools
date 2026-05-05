package gitdata

import (
	"fmt"
	"strings"
	"testing"
)

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

func TestGetCurrentBranch(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		want     string
		wantErr  bool
	}{
		{
			name:    "normal branch",
			output:  "main\n",
			want:    "main",
			wantErr: false,
		},
		{
			name:    "feature branch",
			output:  "feature/login\n",
			want:    "feature/login",
			wantErr: false,
		},
		{
			name:    "detached HEAD (empty output)",
			output:  "\n",
			want:    "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockRunner{
				responses: map[string]string{
					"branch --show-current": tt.output,
				},
			}
			gd := New(mock)
			got, err := gd.GetCurrentBranch()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCurrentBranch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetCurrentBranch() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetBranches(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		want    []Branch
		wantErr bool
	}{
		{
			name: "multiple branches",
			output: "main\x00abc1234\x00abc1234567890123456789012345678901234567\n" +
				"feature/login\x00def5678\x00def5678901234567890123456789012345678901\n" +
				"bugfix/typo\x00ghi9012\x00ghi9012345678901234567890123456789012345678\n",
			want: []Branch{
				{Name: "main", HeadHash: "abc1234567890123456789012345678901234567"},
				{Name: "feature/login", HeadHash: "def5678901234567890123456789012345678901"},
				{Name: "bugfix/typo", HeadHash: "ghi9012345678901234567890123456789012345678"},
			},
		},
		{
			name:   "single branch",
			output: "main\x00abc1234\x00abc1234567890123456789012345678901234567\n",
			want: []Branch{
				{Name: "main", HeadHash: "abc1234567890123456789012345678901234567"},
			},
		},
		{
			name:    "no branches",
			output:  "",
			want:    nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockRunner{
				responses: map[string]string{
					"for-each-ref --format=%(refname:short)%x00%(objectname:short)%x00%(objectname) refs/heads/": tt.output,
				},
			}
			gd := New(mock)
			got, err := gd.GetBranches()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetBranches() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != len(tt.want) {
				t.Errorf("GetBranches() len = %d, want %d", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i].Name != tt.want[i].Name {
					t.Errorf("branch[%d].Name = %q, want %q", i, got[i].Name, tt.want[i].Name)
				}
				if got[i].HeadHash != tt.want[i].HeadHash {
					t.Errorf("branch[%d].HeadHash = %q, want %q", i, got[i].HeadHash, tt.want[i].HeadHash)
				}
			}
		})
	}
}

func TestGetCommits(t *testing.T) {
	commitOutput := "abc1234567890123456789012345678901234567\x00abc1234\x00Initial commit\x00Alice\x002024-01-15T10:30:00+00:00\x00\n" +
		"def5678901234567890123456789012345678901\x00def5678\x00Add login feature\x00Bob\x002024-01-16T14:00:00+00:00\x00abc1234567890123456789012345678901234567\n" +
		"ghi9012345678901234567890123456789012345678\x00ghi9012\x00Merge pull request #42\x00Alice\x002024-01-17T09:00:00+00:00\x00def5678901234567890123456789012345678901 jkl3456789012345678901234567890123456789012\n"

	branchOutput := "main\x00abc1234\x00abc1234567890123456789012345678901234567\n" +
		"feature/login\x00def5678\x00def5678901234567890123456789012345678901\n"

	mainLog := "abc1234567890123456789012345678901234567\ndef5678901234567890123456789012345678901\nghi9012345678901234567890123456789012345678\n"
	featureLog := "def5678901234567890123456789012345678901\n"

	mock := &mockRunner{
		responses: map[string]string{
			"rev-list --count HEAD": "3",
			"log --format=%H%x00%h%x00%s%x00%an%x00%aI%x00%P --max-count=200 --skip=0": commitOutput,
			"for-each-ref --format=%(refname:short)%x00%(objectname:short)%x00%(objectname) refs/heads/": branchOutput,
			"log --format=%H main":          mainLog,
			"log --format=%H feature/login": featureLog,
		},
	}

	gd := New(mock)
	commits, total, err := gd.GetCommits(200, 0)
	if err != nil {
		t.Fatalf("GetCommits() error = %v", err)
	}

	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}

	if len(commits) != 3 {
		t.Fatalf("len(commits) = %d, want 3", len(commits))
	}

	// First commit
	if commits[0].Hash != "abc1234567890123456789012345678901234567" {
		t.Errorf("commits[0].Hash = %s", commits[0].Hash)
	}
	if commits[0].ShortHash != "abc1234" {
		t.Errorf("commits[0].ShortHash = %s", commits[0].ShortHash)
	}
	if commits[0].Message != "Initial commit" {
		t.Errorf("commits[0].Message = %s", commits[0].Message)
	}
	if commits[0].Author != "Alice" {
		t.Errorf("commits[0].Author = %s", commits[0].Author)
	}
	if commits[0].Date != "2024-01-15 10:30:00" {
		t.Errorf("commits[0].Date = %s", commits[0].Date)
	}
	if len(commits[0].Parents) != 0 {
		t.Errorf("commits[0].Parents = %v, want empty", commits[0].Parents)
	}
	if len(commits[0].Branches) < 1 || commits[0].Branches[0] != "main" {
		t.Errorf("commits[0].Branches = %v, want [main]", commits[0].Branches)
	}

	// Second commit
	if commits[1].ShortHash != "def5678" {
		t.Errorf("commits[1].ShortHash = %s", commits[1].ShortHash)
	}
	if len(commits[1].Parents) != 1 {
		t.Errorf("commits[1].Parents = %v, want 1 parent", commits[1].Parents)
	}
	// feature/login branch
	hasFeature := false
	hasMain := false
	for _, b := range commits[1].Branches {
		if b == "feature/login" {
			hasFeature = true
		}
		if b == "main" {
			hasMain = true
		}
	}
	if !hasFeature || !hasMain {
		t.Errorf("commits[1].Branches = %v, want both feature/login and main", commits[1].Branches)
	}

	// Third commit (merge)
	if commits[2].ShortHash != "ghi9012" {
		t.Errorf("commits[2].ShortHash = %s", commits[2].ShortHash)
	}
	if len(commits[2].Parents) != 2 {
		t.Errorf("commits[2].Parents = %v, want 2 parents", commits[2].Parents)
	}
}

func TestGetCommitsPagination(t *testing.T) {
	commitOutput := "commit3hash\x00com3\x00Third commit\x00Alice\x002024-03-01T10:00:00+00:00\x00commit2hash\n" +
		"commit2hash\x00com2\x00Second commit\x00Bob\x002024-02-01T10:00:00+00:00\x00commit1hash\n"

	mock := &mockRunner{
		responses: map[string]string{
			"rev-list --count HEAD": "10",
			"log --format=%H%x00%h%x00%s%x00%an%x00%aI%x00%P --max-count=2 --skip=5": commitOutput,
		},
	}

	gd := New(mock)
	commits, total, err := gd.GetCommits(2, 5)
	if err != nil {
		t.Fatalf("GetCommits() error = %v", err)
	}

	if total != 10 {
		t.Errorf("total = %d, want 10", total)
	}
	if len(commits) != 2 {
		t.Errorf("len(commits) = %d, want 2", len(commits))
	}
}

func TestGetCommitsEmptyRepo(t *testing.T) {
	mock := &mockRunner{
		responses: map[string]string{
			"rev-list --count HEAD": "ERROR: fatal: your current branch 'main' does not have any commits yet",
		},
	}

	gd := New(mock)
	_, _, err := gd.GetCommits(200, 0)
	if err == nil {
		t.Error("GetCommits() expected error for empty repo")
	}
}

func TestGetCommitDetail(t *testing.T) {
	logOutput := "ghi9012345678901234567890123456789012345678\x00Merge pull request #42\x00Alice\x002024-01-17T09:00:00+00:00\n"
	filesOutput := "10\t5\tsrc/main.go\n3\t0\tsrc/auth/login.go\n0\t15\tsrc/old/deprecated.go\n"

	mock := &mockRunner{
		responses: map[string]string{
			"log -1 --format=%H%x00%s%x00%an%x00%aI ghi9012":                       logOutput,
			"diff-tree --no-commit-id --numstat -r ghi9012":                         filesOutput,
			"show --stat=1000 --format= ghi9012":                                    "",
		},
	}

	gd := New(mock)
	detail, err := gd.GetCommitDetail("ghi9012")
	if err != nil {
		t.Fatalf("GetCommitDetail() error = %v", err)
	}

	if detail.Hash != "ghi9012345678901234567890123456789012345678" {
		t.Errorf("Hash = %s", detail.Hash)
	}
	if detail.Message != "Merge pull request #42" {
		t.Errorf("Message = %s", detail.Message)
	}
	if detail.Author != "Alice" {
		t.Errorf("Author = %s", detail.Author)
	}
	if detail.Date != "2024-01-17 09:00:00" {
		t.Errorf("Date = %s", detail.Date)
	}
	if len(detail.Files) != 3 {
		t.Fatalf("len(Files) = %d, want 3", len(detail.Files))
	}

	if detail.Files[0].Path != "src/main.go" {
		t.Errorf("Files[0].Path = %s", detail.Files[0].Path)
	}
	if detail.Files[0].Additions != 10 {
		t.Errorf("Files[0].Additions = %d, want 10", detail.Files[0].Additions)
	}
	if detail.Files[0].Deletions != 5 {
		t.Errorf("Files[0].Deletions = %d, want 5", detail.Files[0].Deletions)
	}

	if detail.Files[2].Additions != 0 {
		t.Errorf("Files[2].Additions = %d, want 0", detail.Files[2].Additions)
	}
	if detail.Files[2].Deletions != 15 {
		t.Errorf("Files[2].Deletions = %d, want 15", detail.Files[2].Deletions)
	}
}

func TestGetCommitDetailMergeCommit(t *testing.T) {
	// Merge commits: diff-tree returns empty, so we fall back to show --stat
	mock := &mockRunner{
		responses: map[string]string{
			"log -1 --format=%H%x00%s%x00%an%x00%aI mergehash":   "mergehash12345678901234567890123456789012\x00Merge branch 'feature'\x00Bob\x002024-03-01T12:00:00+00:00\n",
			"diff-tree --no-commit-id --numstat -r mergehash":     "",
			"show --stat=1000 --format= mergehash":                " src/feature.go | 50 +\n src/main.go    |  2 +-\n",
		},
	}

	gd := New(mock)
	detail, err := gd.GetCommitDetail("mergehash")
	if err != nil {
		t.Fatalf("GetCommitDetail() error = %v", err)
	}

	if detail.Hash != "mergehash12345678901234567890123456789012" {
		t.Errorf("Hash = %s", detail.Hash)
	}
	if len(detail.Files) != 2 {
		t.Fatalf("len(Files) = %d, want 2", len(detail.Files))
	}
	if detail.Files[0].Path != "src/feature.go" {
		t.Errorf("files[0].Path = %s", detail.Files[0].Path)
	}
	if detail.Files[1].Path != "src/main.go" {
		t.Errorf("files[1].Path = %s", detail.Files[1].Path)
	}
}

func TestParseShowStatDirect(t *testing.T) {
	statOutput := " src/feature.go | 50 ++++++++++++++++++++++++++++++++++++++\n" +
		" src/main.go    |  2 +-\n"

	files := parseShowStat(statOutput)
	if len(files) != 2 {
		t.Fatalf("parseShowStat() len = %d, want 2", len(files))
	}
	if files[0].Path != "src/feature.go" {
		t.Errorf("files[0].Path = %s", files[0].Path)
	}
	if files[0].Additions != 50 {
		t.Errorf("files[0].Additions = %d, want 50", files[0].Additions)
	}
	if files[0].Deletions > 0 {
		t.Errorf("files[0].Deletions = %d, want 0", files[0].Deletions)
	}
}

func TestGetCommitDetailInvalidHash(t *testing.T) {
	mock := &mockRunner{
		responses: map[string]string{
			"log -1 --format=%H%x00%s%x00%an%x00%aI deadbeef": "ERROR: fatal: ambiguous argument 'deadbeef': unknown revision",
		},
	}

	gd := New(mock)
	_, err := gd.GetCommitDetail("deadbeef")
	if err == nil {
		t.Error("GetCommitDetail() expected error for invalid hash")
	}
}

func TestParseCommitsEdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect int
	}{
		{
			name:   "empty input",
			input:  "",
			expect: 0,
		},
		{
			name:   "single newline",
			input:  "\n",
			expect: 0,
		},
		{
			name:   "malformed line",
			input:  "not enough fields\n",
			expect: 0,
		},
		{
			name:   "valid single commit",
			input:  "fullhash123\x00short\x00msg\x00author\x002024-01-01T00:00:00+00:00\x00parent\n",
			expect: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commits := parseCommits(tt.input)
			if len(commits) != tt.expect {
				t.Errorf("parseCommits() len = %d, want %d", len(commits), tt.expect)
			}
		})
	}
}

func TestParseBranchesEdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect int
	}{
		{
			name:   "empty input",
			input:  "",
			expect: 0,
		},
		{
			name:   "malformed line",
			input:  "onlyname\n",
			expect: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			branches := parseBranches(tt.input)
			if len(branches) != tt.expect {
				t.Errorf("parseBranches() len = %d, want %d", len(branches), tt.expect)
			}
		})
	}
}

func TestFormatDate(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"2024-01-15T10:30:00+00:00", "2024-01-15 10:30:00"},
		{"2024-12-31T23:59:59+05:30", "2024-12-31 23:59:59"},
		{"not-a-date", "not-a-date"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := formatDate(tt.input)
			if got != tt.want {
				t.Errorf("formatDate(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseFileChangesBinary(t *testing.T) {
	input := "-\t-\tsrc/binary.dat\n"
	files := parseFileChanges(input)
	if len(files) != 1 {
		t.Fatalf("len = %d, want 1", len(files))
	}
	if files[0].Additions != 0 {
		t.Errorf("Additions = %d, want 0", files[0].Additions)
	}
	if files[0].Deletions != 0 {
		t.Errorf("Deletions = %d, want 0", files[0].Deletions)
	}
	if files[0].Path != "src/binary.dat" {
		t.Errorf("Path = %s", files[0].Path)
	}
}

func TestParseFileChangesEdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect int
	}{
		{
			name:   "empty input",
			input:  "",
			expect: 0,
		},
		{
			name:   "whitespace only",
			input:  "  \n",
			expect: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := parseFileChanges(tt.input)
			if len(files) != tt.expect {
				t.Errorf("parseFileChanges() len = %d, want %d", len(files), tt.expect)
			}
		})
	}
}
