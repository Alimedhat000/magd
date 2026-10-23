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

// TODO: report an error if a local variable is never used.
//
// A scope has to know two things about each name now: whether its
// initializer has been resolved (ready) and whether anything has referred
// to it (used). The scope type goes here, and EndScope is where the check
// belongs.
//
// The decisions to work out:
//   - what counts as a use: does `a = 2` count, or only a read?
//   - are parameters exempt? A function may legitimately ignore one.
//   - which name to report when several are unused: map iteration order
//     is random, so the scope needs the declaration order too.
//
// Then thread the error out of EndScope: it returns an error now, and both
// VisitBlockStmt and ResolveFunction call it. ResolveFunction uses a defer,
// so it needs the named-return idiom to report it.

type Resolver struct {
	locals map[ast.Expression]int
	scopes []*scope
	// inFunction is true while resolving a function or method body. A
	// "return" is only legal there.
	inFunction bool
	// inClass is true while resolving a class body. A "this" is only legal
	// there.
	inClass       bool
	hasSuperClass bool
	inInitializer bool
}

type scope struct {
	names    []token.Token
	enteries map[string]*scopeEntry
}

type scopeEntry struct {
	ready bool
	used  bool
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
		scopes: []*scope{},
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
	r.pushScopes(&scope{enteries: map[string]*scopeEntry{}})
}

func (r *Resolver) EndScope() error {
	scope, ok := r.popScopes()
	if !ok {
		return nil
	}

	for _, name := range scope.names {
		if !scope.enteries[name.Lexeme].used {
			return errs.NewSyntaxError(
				name.Line, name.Lexeme,
				"Variable '"+name.Lexeme+"' is never used.")
		}
	}
	return nil
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

	if _, ok := scope.enteries[name.Lexeme]; ok {
		return errs.NewSyntaxError(
			name.Line, name.Lexeme,
			"Already a variable with this name in this scope.")
	}

	scope.enteries[name.Lexeme] = &scopeEntry{
		false,
		false,
	}

	scope.names = append(scope.names, name)

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
	scope.enteries[name.Lexeme].ready = true
}

// Exempt marks a name as used, so the unused check does not report it.
// Parameters are exempt: a function may legitimately ignore one.
func (r *Resolver) Exempt(name token.Token) {
	scope, ok := r.peekScopes()
	if !ok {
		return
	}
	scope.enteries[name.Lexeme].used = true
}

// ResolveLocal starts at the innermost scope and works outwards, looking in
// each map for a matching name. If we find the variable, we record how many
// scopes apart the binding is; names only found in outer or global scopes
// fall back to the environment chain at runtime.
func (r *Resolver) ResolveLocal(expr ast.Expression, name token.Token) {
	for i, scope := range slices.Backward(r.scopes) {
		if _, ok := scope.enteries[name.Lexeme]; ok {
			r.locals[expr] = len(r.scopes) - 1 - i

			// mark as used
			scope.enteries[name.Lexeme].used = true
			return
		}
	}
}

// ResolveFunction resolves the parameters and body of a function in a fresh
// scope. It records that we are inside a function so that return statements
// in the body resolve correctly, and whether the function is an initializer,
// where a return with a value is illegal. Named functions declare and define
// their name before calling this, which is what lets the body recurse.
func (r *Resolver) ResolveFunction(parameters []token.Token, body ast.Stmt, isInitializer bool) (err error) {
	enclosing := r.inFunction
	enclosingInitializer := r.inInitializer
	r.inFunction = true
	r.inInitializer = isInitializer
	defer func() {
		r.inFunction = enclosing
		r.inInitializer = enclosingInitializer
	}()

	r.BeginScope()
	defer func() {
		if Enderr := r.EndScope(); err == nil {
			err = Enderr
		}
	}()

	for _, param := range parameters {
		if err := r.Declare(param); err != nil {
			return err
		}
		r.Define(param)
		r.Exempt(param)
	}

	return r.ResolveStatement(body)
}

// declareThis adds "this" to the innermost scope. It is ready, and exempt
// from the unused check
func (r *Resolver) declareThis() error {
	this := token.Token{Type: token.THIS, Lexeme: "this"}
	if err := r.Declare(this); err != nil {
		return err
	}
	r.Define(this)
	r.Exempt(this)
	return nil
}

func (r *Resolver) declareSuper() error {
	super := token.Token{Type: token.SUPER, Lexeme: "super"}
	if err := r.Declare(super); err != nil {
		return err
	}
	r.Define(super)
	r.Exempt(super)
	return nil
}

func (r *Resolver) pushScopes(scope *scope) {
	r.scopes = append(r.scopes, scope)
}

func (r *Resolver) popScopes() (*scope, bool) {
	if len(r.scopes) == 0 {
		return nil, false
	}

	scope := r.scopes[len(r.scopes)-1]
	r.scopes = r.scopes[:len(r.scopes)-1]
	return scope, true
}

func (r *Resolver) peekScopes() (*scope, bool) {
	if len(r.scopes) == 0 {
		return nil, false
	}
	return r.scopes[len(r.scopes)-1], true
}
