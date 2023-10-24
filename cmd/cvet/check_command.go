package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/samber/lo"
	"golang.org/x/tools/go/analysis"
)

func isCommandFunc(node *ast.FuncDecl) bool {
	return strings.HasSuffix(node.Name.Name, "Command")
}

func loggerCommandKeyValueArgFinder(pass *analysis.Pass, commandName string) func(arg ast.Expr) bool {
	return func(arg ast.Expr) bool {
		call, ok := arg.(*ast.CallExpr)
		if !ok {
			return false
		}
		firstArg, ok := call.Args[0].(*ast.BasicLit)
		if !ok {
			return false
		}
		if firstArg.Kind == token.STRING && firstArg.Value == `"command"` {
			secondArg, ok := call.Args[1].(*ast.BasicLit)
			if !ok {
				pass.Reportf(arg.Pos(), "logger command name must be a literal")
				return true
			}
			if secondArg.Kind != token.STRING {
				pass.Reportf(arg.Pos(), "logger command name must be a string literal")
				return true
			}

			value := secondArg.Value[1 : len(secondArg.Value)-1]
			if value != commandName {
				pass.Reportf(arg.Pos(), fmt.Sprintf("logger command name must be function name %s, but found %s", commandName, value))
				return true
			}
			return true
		}

		return false
	}
}

func checkStmtHaveNoRawSlog(pass *analysis.Pass, stmts []ast.Stmt) {
	for _, stmt := range stmts[1:] {
		switch stmt := stmt.(type) {
		case *ast.ExprStmt:
			if call, ok := stmt.X.(*ast.CallExpr); ok {
				if selector, ok := call.Fun.(*ast.SelectorExpr); ok {
					if isIdent(selector.X, "slog") {
						pass.Reportf(stmt.Pos(), "logger must be used to log")
					}
				}
			}
		case *ast.IfStmt:
			checkStmtHaveNoRawSlog(pass, stmt.Body.List)
		}
	}
}

func checkCommandLogger(pass *analysis.Pass, node *ast.FuncDecl) {
	commandName := node.Name.Name
	firstStmt, ok := node.Body.List[0].(*ast.AssignStmt)
	if !ok || !isIdent(firstStmt.Lhs[0], "logger") {
		pass.Reportf(node.Pos(), "command first statement must be a logger creation")
		return
	}

	call, ok := firstStmt.Rhs[0].(*ast.CallExpr)
	if !ok || !isSelector(call.Fun, "slog", "With") {
		pass.Reportf(node.Pos(), "logger creation must use slog.With")
		return
	}
	if !lo.ContainsBy(call.Args, loggerCommandKeyValueArgFinder(pass, commandName)) {
		pass.Reportf(node.Pos(), "logger creation must contain command arg")
	}

	checkStmtHaveNoRawSlog(pass, node.Body.List[1:])
}

func checkCommandFunc(pass *analysis.Pass, node *ast.FuncDecl) {
	if node.Type.Results.NumFields() != 2 {
		pass.Reportf(node.Pos(), "command function must have 2 return value")
	} else {
		if !isIdent(node.Type.Results.List[0].Type, "int") {
			pass.Reportf(node.Pos(), "command function must return an int code as the first return value")
		}
		if !isIdent(node.Type.Results.List[1].Type, "error") {
			pass.Reportf(node.Pos(), "command function must return an error as the second return value")
		}
	}
	if lo.ContainsBy(node.Type.Params.List, func(field *ast.Field) bool {
		return isSelector(field.Type, "auth", "User")
	}) {
		pass.Reportf(node.Pos(), "command function must not have an auth.User parameter")
	}
	checkCommandLogger(pass, node)
}
