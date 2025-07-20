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
		return &GitRecordExtractorFactory{repoPath: repoPath}, nil
	case "go-git":
		return NewGoGitRecordExtractorFactory(repoPath)
	case "stat":
		return &StatRecordExtractorFactory{repoPath: repoPath}, nil
	default:
		return nil, fmt.Errorf("unsupported algorithm: %s", algo)
	}
}

type GitRecordExtractorFactory struct {
	repoPath string
}

func (f *GitRecordExtractorFactory) CreateExtractor(relativePath string) (RecordExtractor, error) {
	cmd := exec.Command("git", "ls-files", "--error-unmatch", relativePath)
	cmd.Dir = f.repoPath
	if err := cmd.Run(); err != nil {
		return nil, ErrNotTracked
	}
	return &GitRecordExtractor{repoPath: f.repoPath, relativePath: relativePath}, nil
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
	repoPath     string
	relativePath string
}

func (re *GitRecordExtractor) Extract(keyword string, line int) (Record, error) {
	record := Record{
		keyword: keyword,
		line:    line,
	}
	gitArgs := []string{"blame", "-L", fmt.Sprintf("%d,%d", line, line), "--porcelain", re.relativePath}
	cmd := exec.Command("git", gitArgs...) // #nosec G204
	cmd.Dir = re.repoPath
	output, err := cmd.Output()
	if err != nil {
		return record, kcore.Wrap(err, fmt.Sprintf("Error running: git %s", strings.Join(gitArgs, " ")))
	}

	for _, blameLine := range strings.Split(string(output), "\n") {
		if strings.HasPrefix(blameLine, "author-time ") {
			timestamp, err := strconv.ParseInt(strings.TrimPrefix(blameLine, "author-time "), 10, 64)
			kcore.Expect(err, "Error parsing timestamp")
			record.date = time.Unix(timestamp, 0)
			return record, nil
		}
	}
	kcore.Assert(false, "No date found in git blame output")

	return Record{}, nil
}
