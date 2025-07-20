package main

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime/pprof"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/martinlehoux/kagamigo/kcore"
	"github.com/samber/lo"
	"github.com/samber/lo/mutable"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"golang.org/x/exp/slog"
)

type Record struct {
	keyword      string
	relativePath string
	line         int
	date         time.Time
}

var (
	excludes      map[string]bool = map[string]bool{}
	repoPath      string
	after         time.Time
	before        time.Time
	sortBy        string
	maxRecords    int
	algo          string
	cpuProfile    string
	keywords      []string
	binExtensions = map[string]bool{}
)

func initConfig() {
	var err error
	pflag.StringVar(&repoPath, "repo", ".", "Path to the repository")
	if !strings.HasSuffix(repoPath, "/") {
		repoPath = repoPath + "/"
	}
	pflag.String("sort", "date", "Sort by date or random")
	pflag.Int("max", 25, "Maximum number of records to display")
	pflag.String("after", "", "Only show records after this date")
	pflag.String("before", "", "Only show records before this date. ISO date, or '7 days'")
	pflag.String("algo", "git", "Record extraction algorithm (git, go-git, stat)")
	pflag.String("cpuProfile", "", "write cpu profile to file")
	pflag.StringSlice("excludes", []string{".git", ".kdev.yaml"}, "Excluded directories")
	pflag.Parse()

	viper.SetConfigType("yaml")
	viper.SetConfigName(".kdev")
	viper.AddConfigPath(".")
	viper.AddConfigPath(repoPath)
	kcore.Expect(viper.BindPFlags(pflag.CommandLine), "Error binding flags")
	kcore.Expect(viper.ReadInConfig(), "Error reading config file")

	if afterStr := viper.GetString("after"); afterStr != "" {
		after, err = time.Parse(time.DateOnly, viper.GetString("after"))
		kcore.Expect(err, "Error parsing date")
	}
	if beforeStr := viper.GetString("before"); beforeStr != "" {
		before, err = time.Parse(time.DateOnly, viper.GetString("before"))
		kcore.Assert(err != nil || before.IsZero(), "Parsing should fail")
		if err != nil {
			regexp := regexp.MustCompile(`^(\d+)d$`)
			matches := regexp.FindStringSubmatch(viper.GetString("before"))
			if len(matches) == 2 {
				days, err := strconv.Atoi(matches[1])
				kcore.Expect(err, "Error parsing days")
				d := time.Duration(days) * time.Hour * 24
				before = time.Now().Add(-d)
			}
			kcore.Assert(!before.IsZero(), "Error parsing date")
		}
	}
	sortBy = viper.GetString("sort")
	maxRecords = viper.GetInt("max")
	algo = viper.GetString("algo")
	cpuProfile = viper.GetString("cpuProfile")
	keywords = viper.GetStringSlice("keywords")
	for _, exclude := range viper.GetStringSlice("excludes") {
		excludes[exclude] = true
	}
}

func main() {
	var err error
	initConfig()

	records := []Record{}
	if cpuProfile != "" {
		f, err := os.Create(cpuProfile) // #nosec G304 CLI arg
		kcore.Expect(err, "Error creating CPU profile")
		kcore.Expect(pprof.StartCPUProfile(f), "Error starting CPU profile")
		defer pprof.StopCPUProfile()
	}
	slog.Info("Scanning repository", "repo", repoPath, "sort", sortBy, "max", maxRecords, "after", after.Format(time.DateOnly), "before", before.Format(time.DateOnly), "algo", algo)

	factory, err := CreateRecordExtractorFactory(algo, repoPath)
	kcore.Expect(err, "Error creating record extractor factory")

	kcore.Expect(walkRepo(keywords, factory, &records), "Error walking directory")

	if !after.IsZero() {
		records = lo.Filter(records, func(record Record, index int) bool { return record.date.After(after) })
	}
	if !before.IsZero() {
		records = lo.Filter(records, func(record Record, index int) bool { return record.date.Before(before) })
	}
	if sortBy == "random" {
		mutable.Shuffle(records)
	} else {
		slices.SortFunc(records, func(a, b Record) int { return -int(a.date.Sub(b.date).Nanoseconds()) })
	}
	fmt.Println("")
	for _, record := range records[:min(maxRecords, len(records))] {
		fmt.Printf("%s\t %s:%d\n", record.date.Format(time.DateOnly), record.relativePath, record.line)
	}
}

func isBinaryFile(absolutePath, relativePath string) bool {
	// First check by file extension (fast)
	ext := strings.ToLower(filepath.Ext(relativePath))
	if _, ok := binExtensions[ext]; ok {
		return true
	}

	file, err := os.Open(absolutePath)
	kcore.Expect(err, "failed to open file")
	defer file.Close()

	// Read first 512 bytes to check for binary content
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && n == 0 {
		return false
	}

	// Check for null bytes which indicate binary content
	for i := range n {
		if buffer[i] == 0 {
			return true
		}
	}

	return false
}

func walkRepo(keywords []string, factory RecordExtractorFactory, records *[]Record) error {
	progress := progressbar.Default(-1, "Scanning")
	return filepath.WalkDir(repoPath, func(path string, d fs.DirEntry, err error) error {
		relativePath := strings.TrimPrefix(path, repoPath)
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
		if d.IsDir() {
			return nil
		}
		if isBinaryFile(path, relativePath) {
			return nil
		}
		kcore.Expect(progress.Add(1), "Error incrementing progress")
		return processFile(repoPath, relativePath, keywords, records, factory)
	})
}

type MatchingLine struct {
	line    int
	keyword string
}

func processFile(repoPath string, relativePath string, keywords []string, records *[]Record, factory RecordExtractorFactory) error {
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

	recordExtractor, err := factory.CreateExtractor(relativePath)
	switch err {
	case ErrNotTracked:
		return nil
	case nil:
		break
	default:
		return kcore.Wrap(err, "Error creating record extractor")
	}

	for _, line := range matchingLines {
		record, err := recordExtractor.Extract(line.keyword, line.line)
		if err != nil {
			slog.Warn("Error extracting record", "err", err)
		} else {
			record.relativePath = relativePath
			*records = append(*records, record)
		}
	}

	return nil
}
