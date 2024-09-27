## Publishing a new version

- Push the code
- Once the pipeline is green, `git tag vx.x.x`
- `git push --tags`

## gettext

- Add gettext to CI
- Add time handling

1.  `go run github.com/martinlehoux/kagamigo/cmd/gettext -write`
2.  Use `Tr("key", value...)` in you templ files
3.  Complete generated files
4.  Run

## kdev

- Add a config file for excludes
