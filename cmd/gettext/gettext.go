package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/martinlehoux/kagamigo/core"
	"golang.org/x/exp/slog"
	"gopkg.in/yaml.v3"
)

func isIdentifierRune(char rune) bool {
	return (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || char == '.'
}

type ArgsParser struct {
	argsCount            int
	currentBlock         string
	currentNesting       int
	currentStringLiteral bool
}

func (p *ArgsParser) parseStringLiteral(char rune) {
	switch char {
	case '"':
		p.currentStringLiteral = false
		p.currentBlock = ""
	default:
		p.currentBlock += string(char)
	}
}

func (p *ArgsParser) parseNonStringLiteral(char rune) {
	if isIdentifierRune(char) {
		p.currentBlock += string(char)
		return
	}
	switch char {
	case ' ':
		if p.currentBlock != "" {
			if p.currentNesting == 0 {
				p.argsCount++
			}
			p.currentBlock = ""
		}
	case '(':
		p.currentBlock = ""
		p.currentNesting++
	case ')':
		p.currentBlock = ""
		p.currentNesting--
		p.argsCount++
	case '"':
		p.currentStringLiteral = true
	}
}

func (p *ArgsParser) Parse(args string) {
	for _, char := range args {
		switch p.currentStringLiteral {
		case true:
			p.parseStringLiteral(char)
		case false:
			p.parseNonStringLiteral(char)
		}
	}
}

func (p ArgsParser) ArgsCount() int {
	if p.currentBlock != "" {
		p.argsCount++
	}
	return p.argsCount
}

func extractKeys(content string) map[string]int {
	extractedKeys := make(map[string]int, 0)
	reg := regexp.MustCompile(`\{\{\s*call \$?\.T\s*"(\w+)"([\w.\s",:()]*)}}`)
	matches := reg.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		key := match[1]
		args := strings.TrimSpace(match[2])
		parser := ArgsParser{}
		parser.Parse(args)
		extractedKeys[key] = parser.ArgsCount()
	}
	return extractedKeys
}

func extractAllKeys() map[string]int {
	extractedKeys := make(map[string]int, 0)

	err := filepath.Walk("templates", func(path string, info fs.FileInfo, err error) error {
		if !info.IsDir() && filepath.Ext(path) == ".html" {
			content, err := os.ReadFile(path) // #nosec G304
			core.Expect(err, "error reading file")
			for key, value := range extractKeys(string(content)) {
				extractedKeys[key] = value
			}
		}
		return nil
	})
	core.Expect(err, "error walking templates directory")

	return extractedKeys
}

func main() {
	write := flag.Bool("write", false, "write new locales")
	flag.Parse()
	baseLogger := slog.Default()
	langs := [...]string{"en-GB", "fr-FR"}

	extractedKeys := extractAllKeys()
	baseLogger.Info("extracted keys from templates", slog.Int("count", len(extractedKeys)))

	for _, lang := range langs {
		logger := baseLogger.With(slog.String("lang", lang))
		currentLocales := make(map[string]string, 0)
		newLocales := make(map[string]string, 0)

		locales, err := os.ReadFile(filepath.Join("locales", lang, "index.yml")) // #nosec G304
		core.Expect(err, "error reading file")
		core.Expect(yaml.Unmarshal(locales, &currentLocales), "error unmarshalling yaml")

		correctLocales := 0
		for key, translation := range currentLocales {
			currentArgsCount := strings.Count(translation, "%")
			expectedArgsCount, ok := extractedKeys[key]
			switch {
			case !ok:
				logger.Info(`found unused key`, slog.String("key", key))
			case currentArgsCount != expectedArgsCount:
				logger.Info(`found translation with incorrect number of arguments`, slog.String("key", key), slog.Int("current", currentArgsCount), slog.Int("expected", expectedArgsCount))
				newLocales[key] = ""
			case translation == "":
				newLocales[key] = ""
			default:
				newLocales[key] = currentLocales[key]
				correctLocales++
			}
		}

		for key := range extractedKeys {
			if _, ok := currentLocales[key]; !ok {
				logger.Info(`found missing key`, slog.String("key", key))
				newLocales[key] = ""
			}
		}

		logger.Info("finished checking locales", slog.Int("count", len(newLocales)), slog.Int("correct", correctLocales), slog.String("completion", fmt.Sprintf("%d%%", correctLocales*100/len(extractedKeys))))

		if *write {
			content, err := yaml.Marshal(newLocales)
			core.Expect(err, "error marshalling yaml")
			core.Expect(os.WriteFile(filepath.Join("locales", lang, "index.yml"), content, 0600), "error writing file")
		}
	}
}
