// Package interp is the shared lex → parse front end.
//
// Both the file runner and the REPL compile source the same way: scan all
// tokens first, then report lexer errors (they are only complete after the
// full scan — an unterminated string isn't known until EOF), and only parse
// when the lexer is clean.
package interp

import (
	"magd/internal/ast"
	"magd/internal/errs"
	"magd/internal/lexer"
	"magd/internal/parser"
)

// Compile lexes and parses source. It returns the statements with no errors,
// or nil statements with the diagnostics to show the user.
func Compile(source string) ([]ast.Stmt, []errs.SyntaxError) {
	l := lexer.NewLexer(source)
	tokens := l.ScanTokens()

	if errs := l.Errors(); len(errs) > 0 {
		return nil, errs
	}

	p := parser.NewParser(tokens)
	stmts, _ := p.Parse()
	if errs := p.Errors(); len(errs) > 0 {
		return nil, errs
	}

	return stmts, nil
}
