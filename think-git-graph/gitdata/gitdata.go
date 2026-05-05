package gitdata

import (
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type Commit struct {
	Hash      string   `json:"hash"`
	ShortHash string   `json:"shortHash"`
	Message   string   `json:"message"`
	Author    string   `json:"author"`
	Date      string   `json:"date"`
	Parents   []string `json:"parents"`
	Branches  []string `json:"branches"`
}

type Branch struct {
	Name     string `json:"name"`
	HeadHash string `json:"headHash"`
	Color    string `json:"color"`
}

type FileChange struct {
	Path      string `json:"path"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
}

type CommitDetail struct {
	Hash    string       `json:"hash"`
	Message string       `json:"message"`
	Author  string       `json:"author"`
	Date    string       `json:"date"`
	Files   []FileChange `json:"files"`
}

type Runner interface {
	Run(args ...string) (string, error)
}

type RealRunner struct{}

func (r *RealRunner) Run(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

type GitData struct {
	runner Runner
}

func New(runner Runner) *GitData {
	return &GitData{runner: runner}
}

func NewReal() *GitData {
	return &GitData{runner: &RealRunner{}}
}

func (g *GitData) HasUncommitted() (bool, error) {
	out, err := g.runner.Run("status", "--porcelain")
	if err != nil {
		return false, err
	}
	return len(strings.TrimSpace(out)) > 0, nil
}

func (g *GitData) GetCurrentBranch() (string, error) {
	out, err := g.runner.Run("branch", "--show-current")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func (g *GitData) GetCommits(limit, offset int) ([]Commit, int, error) {
	totalOut, err := g.runner.Run("rev-list", "--count", "HEAD")
	if err != nil {
		return nil, 0, err
	}
	total, err := strconv.Atoi(strings.TrimSpace(totalOut))
	if err != nil {
		return nil, 0, err
	}

	out, err := g.runner.Run("log",
		"--topo-order",
		"--format=%H%x00%h%x00%s%x00%an%x00%aI%x00%P",
		"--max-count="+strconv.Itoa(limit),
		"--skip="+strconv.Itoa(offset),
	)
	if err != nil {
		return nil, 0, err
	}

	commits := parseCommits(out)
	if commits == nil {
		commits = []Commit{}
	}

	allBranches, err := g.GetBranches()
	if err == nil {
		branchCommitMap := g.buildBranchCommitMap(allBranches)
		for i := range commits {
			commits[i].Branches = branchCommitMap[commits[i].Hash]
		}
	}

	return commits, total, nil
}

func (g *GitData) GetBranches() ([]Branch, error) {
	out, err := g.runner.Run("for-each-ref",
		"--format=%(refname:short)%x00%(objectname:short)%x00%(objectname)",
		"refs/heads/",
	)
	if err != nil {
		return nil, err
	}

	return parseBranches(out), nil
}

func (g *GitData) GetCommitDetail(hash string) (*CommitDetail, error) {
	out, err := g.runner.Run("log", "-1",
		"--format=%H%x00%s%x00%an%x00%aI",
		hash,
	)
	if err != nil {
		return nil, err
	}

	detail := parseCommitDetail(out, hash)
	if detail == nil {
		return nil, nil
	}

	filesOut, err := g.runner.Run("diff-tree", "--no-commit-id", "--numstat", "-r", hash)
	if err == nil && filesOut != "" {
		detail.Files = parseFileChanges(filesOut)
	}

	// For merge commits, diff-tree shows nothing; try show --stat
	if len(detail.Files) == 0 {
		filesOut, err = g.runner.Run("show", "--stat=1000", "--format=", hash)
		if err == nil && filesOut != "" {
			detail.Files = parseShowStat(filesOut)
		}
	}

	if detail.Files == nil {
		detail.Files = []FileChange{}
	}

	return detail, nil
}

func (g *GitData) buildBranchCommitMap(branches []Branch) map[string][]string {
	result := make(map[string][]string)
	for _, b := range branches {
		out, err := g.runner.Run("log", "--format=%H", b.Name)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
			hash := strings.TrimSpace(line)
			if hash == "" {
				continue
			}
			result[hash] = append(result[hash], b.Name)
		}
	}
	return result
}

func parseCommits(input string) []Commit {
	var commits []Commit
	entries := strings.Split(strings.TrimSpace(input), "\n")
	for _, entry := range entries {
		if entry == "" {
			continue
		}
		parts := strings.Split(entry, "\x00")
		if len(parts) < 6 {
			continue
		}
		var parents []string
		if parts[5] != "" {
			parents = strings.Split(parts[5], " ")
		}
		commits = append(commits, Commit{
			Hash:      parts[0],
			ShortHash: parts[1],
			Message:   parts[2],
			Author:    parts[3],
			Date:      formatDate(parts[4]),
			Parents:   parents,
		})
	}
	return commits
}

var branchColors = []string{
	"#ef4444", "#f97316", "#eab308", "#22c55e", "#06b6d4",
	"#3b82f6", "#8b5cf6", "#ec4899", "#f43f5e", "#84cc16",
	"#14b8a6", "#6366f1", "#a855f7", "#d946ef", "#0ea5e9",
}

func parseBranches(input string) []Branch {
	var branches []Branch
	lines := strings.Split(strings.TrimSpace(input), "\n")
	for i, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\x00")
		if len(parts) < 2 {
			continue
		}
		headHash := parts[1]
		if len(parts) >= 3 {
			headHash = parts[2]
		}
		colorIdx := i % len(branchColors)
		branches = append(branches, Branch{
			Name:     parts[0],
			HeadHash: headHash,
			Color:    branchColors[colorIdx],
		})
	}
	return branches
}

func parseCommitDetail(input string, hash string) *CommitDetail {
	lines := strings.Split(strings.TrimSpace(input), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return nil
	}
	parts := strings.Split(lines[0], "\x00")
	if len(parts) < 4 {
		return nil
	}
	return &CommitDetail{
		Hash:    parts[0],
		Message: parts[1],
		Author:  parts[2],
		Date:    formatDate(parts[3]),
	}
}

func parseFileChanges(input string) []FileChange {
	var files []FileChange
	lines := strings.Split(strings.TrimSpace(input), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		adds, _ := strconv.Atoi(fields[0])
		dels, _ := strconv.Atoi(fields[1])
		// handle binary files (-, -, path)
		if fields[0] == "-" {
			adds = 0
		}
		if fields[1] == "-" {
			dels = 0
		}
		// path is everything after the first two fields
		path := strings.Join(fields[2:], " ")
		files = append(files, FileChange{Path: path, Additions: adds, Deletions: dels})
	}
	return files
}

func parseShowStat(input string) []FileChange {
	var files []FileChange
	lines := strings.Split(strings.TrimSpace(input), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// lines look like: "path/to/file | 10 ++++---"
		if !strings.Contains(line, "|") {
			continue
		}
		parts := strings.SplitN(line, "|", 2)
		if len(parts) < 2 {
			continue
		}
		path := strings.TrimSpace(parts[0])
		changes := strings.TrimSpace(parts[1])
		adds, dels := parseChangeCounts(changes)
		// binary file marker
		if strings.Contains(changes, "Bin") {
			adds, dels = 0, 0
		}
		files = append(files, FileChange{Path: path, Additions: adds, Deletions: dels})
	}
	return files
}

func parseChangeCounts(changes string) (int, int) {
	fields := strings.Fields(changes)
	if len(fields) == 0 {
		return 0, 0
	}
	countStr := strings.TrimSpace(fields[0])
	adds, dels := 0, 0

	if countStr == "Bin" {
		return 0, 0
	}

	// e.g. "10", or "10 ++++---"
	if strings.Contains(countStr, "+") || strings.Contains(countStr, "-") {
		digits := ""
		for _, ch := range countStr {
			if ch >= '0' && ch <= '9' {
				digits += string(ch)
			}
		}
		n, _ := strconv.Atoi(digits)
		adds = strings.Count(countStr, "+")
		dels = strings.Count(countStr, "-")
		if adds == 0 && dels == 0 {
			adds, dels = n, 0
		}
	} else {
		// Could be a plain number
		n, err := strconv.Atoi(countStr)
		if err == nil {
			adds = n
		}
	}
	return adds, dels
}

func formatDate(isoDate string) string {
	t, err := time.Parse(time.RFC3339, isoDate)
	if err != nil {
		return isoDate
	}
	return t.Format("2006-01-02 15:04:05")
}
