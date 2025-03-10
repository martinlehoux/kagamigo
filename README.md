## Publishing a new version

- Push the code
- Once the pipeline is green, `git tag vx.x.x`
- `git push --tags`

## kdev

- [ ] Ignore bin files
- [ ] Ignore regex
- [ ] All config from CLI or file
  - panic: Error opening config file: open .kdev.yaml: no such file or directory

## gettext

- Add gettext to CI
- Add time handling

1.  `go run github.com/martinlehoux/kagamigo/cmd/gettext -write`
2.  Use `Tr("key", value...)` in you templ files
3.  Complete generated files
4.  Run

## web

- Use reflection add startup time to add metadata to logging
- Save aggregate + an event with time & actor (event streaming)
