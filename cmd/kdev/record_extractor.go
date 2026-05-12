package main

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/martinlehoux/kagamigo/kcore"
)

type RecordExtractorFactory interface {
	CreateExtractor(relativePath string) (RecordExtractor, error)
}

type RecordExtractor interface {
	Extract(keyword string, line int) (Record, error)
}

var ErrNotTracked = fmt.Errorf("file is not tracked by git")

func CreateRecordExtractorFactory(algo, repoPath string) (RecordExtractorFactory, error) {
	switch algo {
	case "git":
		return NewGitRecordExtractorFactory(repoPath)
	case "go-git":
		return NewGoGitRecordExtractorFactory(repoPath)
	case "stat":
		return &StatRecordExtractorFactory{repoPath: repoPath}, nil
	default:
		return nil, fmt.Errorf("unsupported algorithm: %s", algo)
	}
}

type GitRecordExtractorFactory struct {
	repoPath     string
	trackedFiles map[string]bool
}

func NewGitRecordExtractorFactory(repoPath string) (*GitRecordExtractorFactory, error) {
	cmd := exec.Command("git", "ls-files")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-files failed: %w", err)
	}
	tracked := map[string]bool{}
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line != "" {
			tracked[line] = true
		}
	}
	return &GitRecordExtractorFactory{repoPath: repoPath, trackedFiles: tracked}, nil
}

func (f *GitRecordExtractorFactory) CreateExtractor(relativePath string) (RecordExtractor, error) {
	if !f.trackedFiles[relativePath] {
		return nil, ErrNotTracked
	}
	cmd := exec.Command("git", "blame", "--porcelain", relativePath) // #nosec G204
	cmd.Dir = f.repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git blame failed: %w", err)
	}
	lineDates, err := parseBlameOutput(output)
	if err != nil {
		return nil, err
	}
	return &GitRecordExtractor{lineDates: lineDates}, nil
}

func parseBlameOutput(output []byte) (map[int]time.Time, error) {
	// Porcelain format emits commit metadata only on first occurrence of a SHA.
	// Track timestamp per SHA, then map final line number → timestamp.
	commitTimes := map[string]time.Time{}
	lineCommit := map[int]string{}
	var currentSHA string
	var currentLine int
	for _, blameLine := range strings.Split(string(output), "\n") {
		// Commit header: "<40-char-sha> <orig> <final> [<count>]"
		if len(blameLine) > 40 && blameLine[40] == ' ' {
			parts := strings.Fields(blameLine)
			if len(parts) >= 3 {
				currentSHA = parts[0]
				n, err := strconv.Atoi(parts[2])
				if err == nil {
					currentLine = n
					lineCommit[currentLine] = currentSHA
				}
			}
		} else if after, ok := strings.CutPrefix(blameLine, "author-time "); ok {
			timestamp, err := strconv.ParseInt(after, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("error parsing timestamp: %w", err)
			}
			commitTimes[currentSHA] = time.Unix(timestamp, 0)
		}
	}
	lineDates := map[int]time.Time{}
	for line, sha := range lineCommit {
		lineDates[line] = commitTimes[sha]
	}
	return lineDates, nil
}

type GoGitRecordExtractorFactory struct {
	head *object.Commit
}

func NewGoGitRecordExtractorFactory(repoPath string) (*GoGitRecordExtractorFactory, error) {
	repo, err := git.PlainOpen(path.Join(repoPath, ".git"))
	kcore.Expect(err, "Error opening repository")
	ref, err := repo.Head()
	kcore.Expect(err, "Error getting HEAD")
	head, err := repo.CommitObject(ref.Hash())
	kcore.Expect(err, "Error getting commit object")
	return &GoGitRecordExtractorFactory{head: head}, nil
}

func (f *GoGitRecordExtractorFactory) CreateExtractor(relativePath string) (RecordExtractor, error) {
	blame, err := git.Blame(f.head, relativePath)
	if err == object.ErrFileNotFound {
		return nil, err
	}
	return &GoGitRecordExtractor{blame: blame}, nil
}

type StatRecordExtractorFactory struct {
	repoPath string
}

func (f *StatRecordExtractorFactory) CreateExtractor(relativePath string) (RecordExtractor, error) {
	fileInfo, err := os.Stat(path.Join(f.repoPath, relativePath))
	kcore.Expect(err, "Error getting file info")

	return &StatRecordExtractor{
		fileInfo: fileInfo,
	}, nil
}

type StatRecordExtractor struct {
	fileInfo fs.FileInfo
}

func (re *StatRecordExtractor) Extract(keyword string, line int) (Record, error) {
	return Record{
		keyword: keyword,
		line:    line,
		date:    re.fileInfo.ModTime(),
	}, nil
}

type GoGitRecordExtractor struct {
	blame *git.BlameResult
}

func NewGoGitRecordExtractor(head *object.Commit, relativePath string) *GoGitRecordExtractor {
	blame, err := git.Blame(head, relativePath)
	if err == object.ErrFileNotFound {
		return nil
	}
	return &GoGitRecordExtractor{
		blame: blame,
	}
}

func (re *GoGitRecordExtractor) Extract(keyword string, line int) (Record, error) {
	record := Record{
		keyword: keyword,
		line:    line,
		date:    re.blame.Lines[line-1].Date,
	}

	return record, nil
}

type GitRecordExtractor struct {
	lineDates map[int]time.Time
}

func (re *GitRecordExtractor) Extract(keyword string, line int) (Record, error) {
	date, ok := re.lineDates[line]
	if !ok {
		return Record{}, fmt.Errorf("no blame data for line %d", line)
	}
	return Record{keyword: keyword, line: line, date: date}, nil
}
