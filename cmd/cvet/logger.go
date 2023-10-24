package main

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

func checkNotLog(pass *analysis.Pass, node *ast.ExprStmt) {
	prevCall, ok := node.X.(*ast.CallExpr)
	if !ok {
		return
	}
	selector, ok := prevCall.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}
	switch {
	case isIdent(selector.X, "slog"):
	case isIdent(selector.X, "logger"):
		pass.Reportf(node.Pos(), "no log before panic")
	}
}

func checkLogBeforePanic(pass *analysis.Pass, node *ast.BlockStmt) {
	for i, stmt := range node.List {
		expr, ok := stmt.(*ast.ExprStmt)
		if !ok {
			continue
		}
		call, ok := expr.X.(*ast.CallExpr)
		if !ok {
			continue
		}
		if i > 0 && isIdent(call.Fun, "panic") {
			if prev, ok := node.List[i-1].(*ast.ExprStmt); ok {
				checkNotLog(pass, prev)
			}
		}
	}
}

func checkLogWithArgs(node *ast.CallExpr, pass *analysis.Pass) {
	for _, arg := range node.Args {
		call, ok := arg.(*ast.CallExpr)
		if !ok {
			continue
		}
		if selector, ok := call.Fun.(*ast.SelectorExpr); ok {
			ident, ok := selector.X.(*ast.Ident)
			if ok && ident.Name != "slog" {
				pass.Reportf(node.Pos(), "slog.With and logger.With must be called with a slog arg")
			}
		} else {
			pass.Reportf(node.Pos(), "slog.With and logger.With must be called with a slog arg")
		}
	}
}

func checkLogUsage(pass *analysis.Pass, node *ast.CallExpr) {
	if selector, ok := node.Fun.(*ast.SelectorExpr); ok {
		if ident, ok := selector.X.(*ast.Ident); ok {
			if ident.Name == "log" {
				pass.Reportf(node.Pos(), "found old log usage")
			}
		}
	}
	if isSelector(node.Fun, "slog", "With") || isSelector(node.Fun, "logger", "With") {
		checkLogWithArgs(node, pass)
	}
}
