package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/martinlehoux/kagamigo/kcore"
	"github.com/samber/lo"
)

type Record struct {
	keyword string
	path    string
	line    int
	date    time.Time
}

func main() {
	var err error
	sortBy := flag.String("sort", "date", "Sort by date or random")
	maxRecords := flag.Int("max", 25, "Maximum number of records to display")
	repoPath := flag.String("repo", ".", "Path to the repository")
	afterS := flag.String("after", "2022-09-01", "Only show records after this date")
	flag.Parse()
	after, err := time.Parse(time.DateOnly, *afterS)
	kcore.Expect(err, "Error parsing date")
	excludes := []string{".venv", ".git", ".ruff_cache", "db_dumps", ".mypy_cache"}
	keywords := []string{"# TODO"}
	records := []Record{}
	// repo, err := git.PlainOpen(root)
	kcore.Expect(err, "Error opening repository")
	// ref, err := repo.Head()
	kcore.Expect(err, "Error getting HEAD")
	// commit, err := repo.CommitObject(ref.Hash())
	kcore.Expect(err, "Error getting commit object")
	err = filepath.WalkDir(*repoPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if slices.Contains(excludes, d.Name()) {
			return filepath.SkipDir
		}
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		if !d.IsDir() {
			return processFile(path, keywords, &records, *repoPath)
		}
		return nil
	})
	kcore.Expect(err, "Error walking directory")
	records = lo.Filter(records, func(record Record, index int) bool { return record.date.After(after) })
	if *sortBy == "random" {
		records = lo.Shuffle(records)
	} else {
		slices.SortFunc(records, func(a, b Record) int { return -int(a.date.Sub(b.date).Nanoseconds()) })
	}
	for _, record := range records[:*maxRecords] {
		fmt.Printf("%s\t %s:%d\n", record.date.Format("2006-01-02"), record.path, record.line)
	}
}

func processFile(path string, keywords []string, records *[]Record, repoPath string) error {
	content, err := os.ReadFile(path) // #nosec G304
	if err != nil {
		return err
	}
	lines := strings.Split(string(content), "\n")
	for i, line := range lines {
		for _, keyword := range keywords {
			if strings.Contains(line, keyword) {
				*records = append(*records, extractRecord(keyword, path, i+1, repoPath))
			}
		}
	}
	return nil
}

func extractRecord(keyword string, path string, line int, repoPath string) Record {
	record := Record{
		keyword: keyword,
		path:    path,
		line:    line,
	}
	cmd := exec.Command("git", "blame", "-L", fmt.Sprintf("%d,%d", record.line, record.line), "--porcelain", path) // #nosec G204
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		panic(kcore.Wrap(err, "Error running git blame"))
	}
	for _, blameLine := range strings.Split(string(output), "\n") {
		if strings.HasPrefix(blameLine, "author-time ") {
			timestamp, err := strconv.ParseInt(strings.TrimPrefix(blameLine, "author-time "), 10, 64)
			kcore.Expect(err, "Error parsing timestamp")
			record.date = time.Unix(timestamp, 0)
			break
		}
	}
	return record
}
