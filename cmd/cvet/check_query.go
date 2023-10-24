package main

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
)

func isQueryFunc(node *ast.FuncDecl) bool {
	return strings.HasSuffix(node.Name.Name, "Query")
}

func checkQueryFunc(pass *analysis.Pass, node *ast.FuncDecl) {
	if node.Type.Results.NumFields() != 3 {
		pass.Reportf(node.Pos(), "query function must have 3 return values")
	} else {
		if !isIdent(node.Type.Results.List[1].Type, "int") {
			pass.Reportf(node.Pos(), "query function must return an int code as the second return value")
		}
		if !isIdent(node.Type.Results.List[2].Type, "error") {
			pass.Reportf(node.Pos(), "query function must return an error as the third return value")
		}
	}
}
