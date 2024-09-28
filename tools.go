//go:build tools

package main

import (
	_ "github.com/fzipp/gocyclo/cmd/gocyclo"
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "github.com/securego/gosec/v2/cmd/gosec"
	_ "honnef.co/go/tools/cmd/staticcheck"
)
