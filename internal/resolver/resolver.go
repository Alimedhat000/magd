// Package resolver
//
// The resolver is the pass between parsing and executing. It walks the syntax
// tree once and works out, for every name the program mentions, which scope
// that name belongs to. The result is a side table: a map from an expression
// to the number of scopes between it and the scope the name was declared in.
//
// The resolver knows nothing about the interpreter. It produces the side
// table; the interpreter reads it.
package resolver

import (
	"slices"

	"magd/internal/ast"
	"magd/internal/errs"
	"magd/internal/token"
)

// functionType is a two-state flag that answers one question: "am I inside a function body right now?"
type functionType int

const (
	functionNone functionType = iota
	functionFunction
)

type Resolver struct {
	locals          map[ast.Expression]int
	scopes          []map[string]bool
	currentFunction functionType // which function are we in now
}

var (
	_ ast.ExprVisitor = (*Resolver)(nil)
	_ ast.StmtVisitor = (*Resolver)(nil)
)

// New returns a resolver that records the scope distances it finds in
// locals. The table must not be nil.
//
// The table is owned by the caller, not the resolver: a program that is run
// in pieces — the REPL evaluates one entry at a time — keeps the entries
// from earlier pieces, because a function defined in one entry is called in
// another.
func New(locals map[ast.Expression]int) *Resolver {
	return &Resolver{
		locals: locals,
		scopes: []map[string]bool{},
	}
}

func (r *Resolver) ResolveStatements(stmts []ast.Stmt) error {
	for _, stmt := range stmts {
		err := r.ResolveStatement(stmt)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Resolver) ResolveStatement(stmt ast.Stmt) error {
	_, err := stmt.Accept(r)
	if err != nil {
		return err
	}
	return nil
}

func (r *Resolver) ResolveExpression(expr ast.Expression) error {
	_, err := expr.Accept(r)
	if err != nil {
		return err
	}
	return nil
}

func (r *Resolver) BeginScope() {
	r.pushScopes(map[string]bool{})
}

func (r *Resolver) EndScope() {
	r.popScopes()
}

// Declare adds the variable to the innermost scope so that it shadows
// any outer one and so that we know the variable exists.
// It sets the value to false to mark it as "not ready yet".
//
// Declaring a name the scope already holds is a mistake: the earlier
// binding was meant to be assigned to, not shadowed. Globals are exempt —
// the resolver does not track them, so a global may be redeclared.
func (r *Resolver) Declare(name token.Token) error {
	if len(r.scopes) == 0 {
		return nil
	}

	scope, ok := r.peekScopes()
	if !ok {
		return nil
	}

	if _, ok := scope[name.Lexeme]; ok {
		return errs.NewSyntaxError(
			name.Line, name.Lexeme,
			"Already a variable with this name in this scope.")
	}

	scope[name.Lexeme] = false
	return nil
}

// Define sets the variable's value to true to mark it as fully initialized
func (r *Resolver) Define(name token.Token) {
	if len(r.scopes) == 0 {
		return
	}
	scope, ok := r.peekScopes()
	if !ok {
		return
	}
	scope[name.Lexeme] = true
}

// ResolveLocal starts at the innermost scope and works outwards, looking in
// each map for a matching name. If we find the variable, we record how many
// scopes apart the binding is; names only found in outer or global scopes
// fall back to the environment chain at runtime.
func (r *Resolver) ResolveLocal(expr ast.Expression, name token.Token) {
	for i, scope := range slices.Backward(r.scopes) {
		if _, ok := scope[name.Lexeme]; ok {
			r.locals[expr] = len(r.scopes) - 1 - i
			return
		}
	}
}

// ResolveFunction resolves the parameters and body of a function in a fresh
// scope. It records that we are inside a function so that return statements
// in the body resolve correctly. Named functions declare and define their
// name before calling this, which is what lets the body recurse.
func (r *Resolver) ResolveFunction(parameters []token.Token, body ast.Stmt) error {
	enclosingFunction := r.currentFunction
	r.currentFunction = functionFunction
	defer func() { r.currentFunction = enclosingFunction }()

	r.BeginScope()
	defer r.EndScope()

	for _, param := range parameters {
		if err := r.Declare(param); err != nil {
			return err
		}
		r.Define(param)
	}

	return r.ResolveStatement(body)
}

func (r *Resolver) pushScopes(scope map[string]bool) {
	r.scopes = append(r.scopes, scope)
}

func (r *Resolver) popScopes() (map[string]bool, bool) {
	if len(r.scopes) == 0 {
		return nil, false
	}

	scope := r.scopes[len(r.scopes)-1]
	r.scopes = r.scopes[:len(r.scopes)-1]
	return scope, true
}

func (r *Resolver) peekScopes() (map[string]bool, bool) {
	if len(r.scopes) == 0 {
		return nil, false
	}
	return r.scopes[len(r.scopes)-1], true
}
