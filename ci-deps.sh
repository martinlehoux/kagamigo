#!/bin/sh
set -ev

go install github.com/a-h/templ/cmd/templ@latest
go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
go install honnef.co/go/tools/cmd/staticcheck@latest
go install github.com/securego/gosec/v2/cmd/gosec@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/a-h/templ/cmd/templ@latest