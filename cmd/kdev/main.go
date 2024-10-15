package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime/pprof"
	"slices"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/martinlehoux/kagamigo/kcore"
	"github.com/samber/lo"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/viper"
	"golang.org/x/exp/slog"
)

type Record struct {
	keyword string
	path    string
	line    int
	date    time.Time
}

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
var sortBy = flag.String("sort", "date", "Sort by date or random")
var maxRecords = flag.Int("max", 25, "Maximum number of records to display")
var repoPath = flag.String("repo", ".", "Path to the repository")
var afterS = flag.String("after", "2022-09-01", "Only show records after this date")
var algo = flag.String("algo", "git", "Record extraction algorithm (git, go-git, stat)")

var excludes map[string]bool = map[string]bool{
	".git":       true,
	".kdev.yaml": true,
}

func main() {
	var err error
	flag.Parse()
	after, err := time.Parse(time.DateOnly, *afterS)
	kcore.Expect(err, "Error parsing date")
	if !strings.HasSuffix(*repoPath, "/") {
		*repoPath = *repoPath + "/"
	}

	records := []Record{}
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		kcore.Expect(err, "Error creating CPU profile")
		kcore.Expect(pprof.StartCPUProfile(f), "Error starting CPU profile")
		defer pprof.StopCPUProfile()
	}
	slog.Info("Scanning repository", "repo", *repoPath, "sort", *sortBy, "max", *maxRecords, "after", *afterS, "algo", *algo)
	viper.SetConfigType("yaml")
	configFile, err := os.Open(path.Join(*repoPath, ".kdev.yaml"))
	kcore.Expect(err, "Error opening config file")
	defer configFile.Close()
	kcore.Expect(viper.ReadConfig(configFile), "Error reading config file")
	for _, exclude := range viper.GetStringSlice("excludes") {
		excludes[exclude] = true
	}
	keywords := viper.GetStringSlice("keywords")
	repo, err := git.PlainOpen(path.Join(*repoPath, ".git"))
	kcore.Expect(err, "Error opening repository")
	ref, err := repo.Head()
	kcore.Expect(err, "Error getting HEAD")
	head, err := repo.CommitObject(ref.Hash())
	kcore.Expect(err, "Error getting commit object")
	kcore.Expect(walkRepo(keywords, head, &records), "Error walking directory")

	records = lo.Filter(records, func(record Record, index int) bool { return record.date.After(after) })
	if *sortBy == "random" {
		records = lo.Shuffle(records)
	} else {
		slices.SortFunc(records, func(a, b Record) int { return -int(a.date.Sub(b.date).Nanoseconds()) })
	}
	fmt.Println("")
	for _, record := range records[:min(*maxRecords, len(records))] {
		fmt.Printf("%s\t %s:%d\n", record.date.Format("2006-01-02"), record.path, record.line)
	}
}

func walkRepo(keywords []string, head *object.Commit, records *[]Record) error {
	progress := progressbar.Default(-1, "Scanning")
	return filepath.WalkDir(*repoPath, func(path string, d fs.DirEntry, err error) error {
		relativePath := strings.TrimPrefix(path, *repoPath)
		progress.Describe(relativePath)
		if err != nil {
			return err
		}
		if excludes[d.Name()] {
			if d.IsDir() {
				return filepath.SkipDir
			} else {
				return nil
			}
		}
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		if !d.IsDir() {
			kcore.Expect(progress.Add(1), "Error incrementing progress")
			return processFile(*repoPath, relativePath, keywords, records, head)
		}
		return nil
	})
}

type MatchingLine struct {
	line    int
	keyword string
}

func processFile(repoPath string, relativePath string, keywords []string, records *[]Record, head *object.Commit) error {
	absolutePath := path.Join(repoPath, relativePath)
	content, err := os.ReadFile(absolutePath) // #nosec G304
	if err != nil {
		return kcore.Wrap(err, "Error reading file")
	}

	lines := strings.Split(string(content), "\n")
	matchingLines := lo.FilterMap(lines, func(line string, i int) (MatchingLine, bool) {
		for _, keyword := range keywords {
			if strings.Contains(line, keyword) {
				return MatchingLine{i + 1, keyword}, true
			}
		}
		return MatchingLine{}, false
	})
	if len(matchingLines) == 0 {
		return nil
	}

	var recordExtractor RecordExtractor
	switch *algo {
	case "git":
		recordExtractor = &GitRecordExtractor{repoPath: repoPath}
	case "go-git":
		recordExtractor = NewGoGitRecordExtractor(head, relativePath)
	case "stat":
		recordExtractor = NewStatRecordExtractor(absolutePath)
	default:
		kcore.Assert(false, "wrong algo value (git, go-git, stat)")
	}
	if recordExtractor == nil {
		return nil
	}
	for _, line := range matchingLines {
		*records = append(*records, recordExtractor.Extract(line.keyword, relativePath, line.line))
	}

	return nil
}
