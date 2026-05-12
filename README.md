## Publishing a new version

- Run `task check`
- Push the code
- `git tag vx.x.x`
- `git push --tags`

## kdev

`go run github.com/martinlehoux/kagamigo/cmd/kdev@latest --repo=<repo> --keywords=TODO`

Scans a git repository for keyword occurrences and shows when each matching line was last modified (via `git blame`).

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--repo` | `.` | Path to the repository |
| `--keywords` | _(required)_ | Comma-separated keywords to search for |
| `--max` | `25` | Maximum number of results to display |
| `--sort` | `date` | Sort order: `date` or `random` |
| `--after` | | Only show results after this date (ISO format) |
| `--before` | | Only show results before this date (ISO format or `7d`) |
| `--algo` | `git` | Blame algorithm: `git`, `go-git`, or `stat` |
| `--workers` | `NumCPU` | Number of parallel workers |
| `--maxFileSize` | `1048576` | Skip files larger than this size in bytes |
| `--ignore` | | Regex patterns to skip files by relative path |
| `--excludes` | `.git,.kdev.yaml` | Directory/file names to exclude |
| `--logLevel` | `info` | Log level: `debug`, `info`, `warn`, `error` |

Settings can also be stored in `.kdev.yaml` at the repo root.

### Known limitation

`git blame` runs once per file with keyword matches (not per line). For repositories with many matching files, `--workers` controls parallelism.

## ki18n

### Setup

1. Call `ki18n.Init(localesFS)` at startup with a filesystem containing `<lang>/*.yml` translation files. Pass extra `ki18n.Locale` values to register additional languages.
2. Register `ki18n.LangMiddleware(...)` on your router, passing one or more strategies in priority order.
3. Run `go run github.com/martinlehoux/kagamigo/ki18n/cmd/gettext -write` to generate translation files.
4. Complete the generated files.
5. Add the check to CI — exits 1 if any translation key is missing from a locale file:

```sh
go run github.com/martinlehoux/kagamigo/ki18n/cmd/gettext
```

### Language detection strategies

```go
// Detect from a cookie named "lang", then fall back to Accept-Language header
ki18n.LangMiddleware(
    ki18n.CookieStrategy("lang"),
    ki18n.AcceptLanguageStrategy,
)
```

Built-in strategies:
- `ki18n.CookieStrategy(name string)` — reads the named cookie
- `ki18n.AcceptLanguageStrategy` — parses the `Accept-Language` request header

If no strategy resolves a language, the middleware falls back to `en-GB`.

### Formatting dates

```go
ki18n.FormatTime(ctx, time.Now()) // "12 May 2026" in en-GB, "12 mai 2026" in fr-FR
```

To add a language, pass a `Locale` to `Init`:

```go
ki18n.Init(localesFS, ki18n.Locale{
    Lang: "es-ES",
    FormatTime: func(t time.Time) string {
        return fmt.Sprintf("%d de %s de %d", t.Day(), spanishMonths[t.Month()-1], t.Year())
    },
})
```

`FormatTime` is optional on `Locale` — omit it to fall back to `en-GB` date formatting.

### Translating in templ files

```go
// ctx is the context passed to your templ component
@ki18n.Tr(ctx, "Hello %s", userName)
```

If the key is not found in the resolved language, the format string itself is returned.

## web

- [ ] Use reflection add startup time to add metadata to logging
- [ ] Save aggregate + an event with time & actor (event streaming)

## kcore

```go
handler.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
  kcore.RenderPage(r.Context(), pages.HomePage(), w)
})
```

```go
block, err := aes.NewCipher(secret)
kcore.Expect(err, "error creating AES cipher")
```

## kauth

- [ ] Auto logout for some errors (unreachable user, expired)
- [ ] Revisit the login and signup flow
