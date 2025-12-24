## Publishing a new version

- Push the code
- Once the pipeline is green, `git tag vx.x.x`
- `git push --tags`

## kdev

- [ ] Ignore regex
- [ ] Log level

## gettext

- Add gettext to CI
- Add time handling

1.  `go run github.com/martinlehoux/kagamigo/cmd/gettext -write`
2.  Use `Tr("key", value...)` in you templ files
3.  Complete generated files
4.  Run

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
