package main

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/pprof"
	"slices"
	"strconv"
	"strings"
	"sync"
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
	excludes       map[string]bool = map[string]bool{}
	ignorePatterns []*regexp.Regexp
	repoPath       string
	after          time.Time
	before         time.Time
	sortBy         string
	maxRecords     int
	maxFileSize    int64
	algo           string
	cpuProfile     string
	keywords       []string
	logLevel       string
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
	pflag.String("logLevel", "info", "Log level (debug, info, warn, error)")
	pflag.Int64("maxFileSize", 1024*1024, "Skip files larger than this size in bytes")
	pflag.StringSlice("ignore", []string{}, "Regex patterns to ignore files by relative path")
	pflag.StringSlice("keywords", []string{}, "Keywords to search for")
	pflag.Parse()

	viper.SetConfigType("yaml")
	viper.SetConfigName(".kdev")
	viper.AddConfigPath(".")
	viper.AddConfigPath(repoPath)
	kcore.Expect(viper.BindPFlags(pflag.CommandLine), "Error binding flags")
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			kcore.Expect(err, "Error reading config file")
		}
	}

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
	maxFileSize = viper.GetInt64("maxFileSize")
	for _, pattern := range viper.GetStringSlice("ignore") {
		re, err := regexp.Compile(pattern)
		kcore.Expect(err, "Invalid ignore pattern: "+pattern)
		ignorePatterns = append(ignorePatterns, re)
	}
	logLevel = viper.GetString("logLevel")
	level := slog.LevelInfo
	kcore.Expect(level.UnmarshalText([]byte(logLevel)), "Invalid log level")
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})))
}

func main() {
	var err error
	initConfig()
	kcore.Assert(len(keywords) > 0, "No keywords configured: set 'keywords' in .kdev.yaml or pass --keywords")

	records := []Record{}
	if cpuProfile != "" {
		f, err := os.Create(cpuProfile) // #nosec G304 CLI arg
		kcore.Expect(err, "Error creating CPU profile")
		kcore.Expect(pprof.StartCPUProfile(f), "Error starting CPU profile")
		defer pprof.StopCPUProfile()
	}
	logArgs := []any{"repo", repoPath, "sort", sortBy, "max", maxRecords, "algo", algo}
	if !after.IsZero() {
		logArgs = append(logArgs, "after", after.Format(time.DateOnly))
	}
	if !before.IsZero() {
		logArgs = append(logArgs, "before", before.Format(time.DateOnly))
	}
	slog.Info("Scanning repository", logArgs...)

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

func isBinaryFile(absolutePath string) bool {
	file, err := os.Open(absolutePath) // #nosec G304 Repo walking
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
	workers := runtime.NumCPU()
	work := make(chan string, workers*2)
	results := make(chan []Record, workers*2)

	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for relativePath := range work {
				recs, err := processFile(repoPath, relativePath, keywords, factory)
				if err != nil {
					slog.Warn("Error processing file", "path", relativePath, "err", err)
					continue
				}
				if len(recs) > 0 {
					results <- recs
				}
			}
		}()
	}

	var collectorDone sync.WaitGroup
	collectorDone.Add(1)
	go func() {
		defer collectorDone.Done()
		for recs := range results {
			*records = append(*records, recs...)
		}
	}()

	progress := progressbar.NewOptions(-1, progressbar.OptionSetDescription("Scanning"), progressbar.OptionClearOnFinish(), progressbar.OptionSetWriter(os.Stderr))
	walkErr := filepath.WalkDir(repoPath, func(path string, d fs.DirEntry, err error) error {
		relativePath := strings.TrimPrefix(path, repoPath)
		progress.Describe(relativePath)
		if err != nil {
			return err
		}
		if excludes[d.Name()] {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		for _, re := range ignorePatterns {
			if re.MatchString(relativePath) {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}
		if d.IsDir() {
			return nil
		}
		if info, err := d.Info(); err == nil && info.Size() > maxFileSize {
			slog.Debug("Skipping large file", "path", relativePath, "size", info.Size())
			return nil
		}
		if isBinaryFile(path) {
			return nil
		}
		kcore.Expect(progress.Add(1), "Error incrementing progress")
		work <- relativePath
		return nil
	})

	close(work)
	wg.Wait()
	close(results)
	collectorDone.Wait()

	filesScanned := progress.State().CurrentNum
	kcore.Expect(progress.Finish(), "Error finishing progress bar")
	slog.Info("Scan complete", "workers", workers, "files_scanned", filesScanned)
	return walkErr
}

type MatchingLine struct {
	line    int
	keyword string
}

func processFile(repoPath string, relativePath string, keywords []string, factory RecordExtractorFactory) ([]Record, error) {
	absolutePath := path.Join(repoPath, relativePath)
	content, err := os.ReadFile(absolutePath) // #nosec G304
	if err != nil {
		return nil, kcore.Wrap(err, "Error reading file")
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
		return nil, nil
	}

	recordExtractor, err := factory.CreateExtractor(relativePath)
	switch err {
	case ErrNotTracked:
		return nil, nil
	case nil:
		break
	default:
		return nil, kcore.Wrap(err, "Error creating record extractor")
	}

	var records []Record
	for _, line := range matchingLines {
		record, err := recordExtractor.Extract(line.keyword, line.line)
		if err != nil {
			slog.Warn("Error extracting record", "path", relativePath, "line", line.line, "err", err)
		} else {
			record.relativePath = relativePath
			records = append(records, record)
		}
	}

	return records, nil
}
