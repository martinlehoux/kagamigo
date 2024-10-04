package main

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/martinlehoux/kagamigo/kcore"
)

type RecordExtractor interface {
	Extract(keyword string, path string, line int) Record
}

type StatRecordExtractor struct {
	fileInfo fs.FileInfo
}

func NewStatRecordExtractor(absolutePath string) *StatRecordExtractor {
	fileInfo, err := os.Stat(absolutePath)
	kcore.Expect(err, "Error getting file info")

	return &StatRecordExtractor{
		fileInfo: fileInfo,
	}
}

func (re *StatRecordExtractor) Extract(keyword string, path string, line int) Record {
	return Record{
		keyword: keyword,
		path:    path,
		line:    line,
		date:    re.fileInfo.ModTime(),
	}
}

type GoGitRecordExtractor struct {
	blame *git.BlameResult
}

func NewGoGitRecordExtractor(head *object.Commit, path string) *GoGitRecordExtractor {
	blame, err := git.Blame(head, path)
	if err == object.ErrFileNotFound {
		return nil
	}
	return &GoGitRecordExtractor{
		blame: blame,
	}
}

func (re *GoGitRecordExtractor) Extract(keyword string, path string, line int) Record {
	return Record{
		keyword: keyword,
		path:    path,
		line:    line,
		date:    re.blame.Lines[line-1].Date,
	}
}

type GitRecordExtractor struct {
	repoPath string
}

func (re *GitRecordExtractor) Extract(keyword string, path string, line int) Record {
	record := Record{
		keyword: keyword,
		path:    path,
		line:    line,
	}
	gitArgs := []string{"blame", "-L", fmt.Sprintf("%d,%d", line, line), "--porcelain", path}
	cmd := exec.Command("git", gitArgs...) // #nosec G204
	cmd.Dir = re.repoPath
	output, err := cmd.Output()

	if err != nil {
		panic(kcore.Wrap(err, fmt.Sprintf("Error running: git %s", strings.Join(gitArgs, " "))))
	}

	for _, blameLine := range strings.Split(string(output), "\n") {
		if strings.HasPrefix(blameLine, "author-time ") {
			timestamp, err := strconv.ParseInt(strings.TrimPrefix(blameLine, "author-time "), 10, 64)
			kcore.Expect(err, "Error parsing timestamp")
			record.date = time.Unix(timestamp, 0)
			return record
		}
	}
	kcore.Assert(false, "No date found in git blame output")
	return Record{}
}
