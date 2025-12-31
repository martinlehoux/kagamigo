package main

import (
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/martinlehoux/kagamigo/kcore"
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
	// Match both { ... } style and @ki18n.Tr(...) style, supporting both Tr and Str functions
	reg := regexp.MustCompile(`(?:\{ ?((?:login\.|ki18n\.)?(?:Tr|Str)\([^}]*\) ?)\}|@ki18n\.Tr\(([^)]+)\))`)
	matches := reg.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		var trCall string
		if match[1] != "" {
			// { ... } style match
			trCall = match[1]
		} else {
			// @ki18n.Tr(...) style match
			trCall = "ki18n.Tr(" + match[2] + ")"
		}
		call, isTrCall := parseTrCall(trCall)
		if isTrCall {
			keyLiteral := call.Args[1].(*ast.BasicLit).Value
			key, err := strconv.Unquote(keyLiteral)
			if err != nil {
				slog.Warn("error unquoting key", "key", keyLiteral)
				key = strings.Trim(keyLiteral, `"`)
			}
			extractedKeys[key] = len(call.Args) - 2
		}
	}
	return extractedKeys
}

func parseTrCall(trCall string) (*ast.CallExpr, bool) {
	fakePackage := fmt.Sprintf("package main\nfunc main() {\n	%s\n}", trCall)
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", fakePackage, 0)
	kcore.Expect(err, "error parsing file")
	main := f.Decls[0].(*ast.FuncDecl)
	call := main.Body.List[0].(*ast.ExprStmt).X.(*ast.CallExpr)
	switch call.Fun.(type) {
	case *ast.SelectorExpr:
		selector := call.Fun.(*ast.SelectorExpr)
		return call, selector.Sel.Name == "Tr" || selector.Sel.Name == "Str"

	case *ast.Ident:
		ident := call.Fun.(*ast.Ident)
		return call, ident.Name == "Tr" || ident.Name == "Str"
	}
	return call, false
}

func extractAllKeys() map[string]int {
	extractedKeys := make(map[string]int, 0)

	err := filepath.Walk(".", func(path string, info fs.FileInfo, err error) error {
		if !info.IsDir() && filepath.Ext(path) == ".templ" {
			content, err := os.ReadFile(path) // #nosec G304
			kcore.Expect(err, "error reading file")
			maps.Copy(extractedKeys, extractKeys(string(content)))
		}
		return nil
	})
	kcore.Expect(err, "error walking templates directory")

	return extractedKeys
}

func getOrCreateLocale(lang string, logger *slog.Logger) map[string]string {
	currentLocales := make(map[string]string, 0)
	locales, err := os.ReadFile(filepath.Join("locales", lang, "index.yml")) // #nosec G304
	if errors.Is(err, os.ErrNotExist) {
		logger.Info("no locales file found, creating")
		err = os.MkdirAll(filepath.Join("locales", lang), 0o700)
		kcore.Expect(err, "error creating directory")
		err = os.WriteFile(filepath.Join("locales", lang, "index.yml"), []byte{}, 0o600)
		kcore.Expect(err, "error writing file")
		locales, err = os.ReadFile(filepath.Join("locales", lang, "index.yml")) // #nosec G304
		kcore.Expect(err, "error reading file")
	} else {
		kcore.Expect(err, "error reading file")
	}
	kcore.Expect(yaml.Unmarshal(locales, &currentLocales), "error unmarshalling yaml")
	return currentLocales
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
		currentLocales := getOrCreateLocale(lang, logger)
		newLocales := make(map[string]string, 0)

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
		var completion string
		if len(extractedKeys) > 0 {
			completion = fmt.Sprintf("%d%%", correctLocales*100/len(extractedKeys))
		} else {
			completion = "?%"
		}
		logger.Info("finished checking locales", slog.Int("count", len(newLocales)), slog.Int("correct", correctLocales), slog.String("completion", completion))

		if *write {
			content, err := yaml.Marshal(newLocales)
			kcore.Expect(err, "error marshalling yaml")
			kcore.Expect(os.WriteFile(filepath.Join("locales", lang, "index.yml"), content, 0o600), "error writing file")
		}
	}
}
