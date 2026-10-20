package eval

import (
	"strings"
	"testing"

	"magd/internal/ast"
	"magd/internal/lexer"
	"magd/internal/parser"
)

// Chapter 8 coverage: global variables, assignment, blocks, and scope.
//
// These tests live in package eval so they can reach evaluate and the
// Evaluator's environment directly. Each drives the real pipeline — lex, parse,
// Eval — rather than hand-built ASTs, so they also cover the front end.

// runProgram parses and evaluates source, returning any runtime error.
func runProgram(t *testing.T, source string) error {
	t.Helper()

	p := parser.NewParser(lexer.NewLexer(source).ScanTokens())
	stmts, _ := p.Parse()
	if errs := p.Errors(); len(errs) != 0 {
		t.Fatalf("parse(%q): unexpected errors: %v", source, errs)
	}

	return NewEvaluator().Eval(stmts)
}

// evalProgram parses and evaluates source expecting success.
func evalProgram(t *testing.T, source string) {
	t.Helper()

	if err := runProgram(t, source); err != nil {
		t.Fatalf("eval(%q): unexpected error: %v", source, err)
	}
}

// evalTo runs a program and returns the Evaluator, so a test can inspect the
// globals it left behind. Statements discard their values, so reading the
// environment is the only way to observe a binding directly.
func evalTo(t *testing.T, source string) *Evaluator {
	t.Helper()

	p := parser.NewParser(lexer.NewLexer(source).ScanTokens())
	stmts, _ := p.Parse()
	if errs := p.Errors(); len(errs) != 0 {
		t.Fatalf("parse(%q): unexpected errors: %v", source, errs)
	}

	e := NewEvaluator()
	if err := e.Eval(stmts); err != nil {
		t.Fatalf("eval(%q): unexpected error: %v", source, err)
	}

	return e
}

// ---------------------------------------------------------------------------
// 8.2 Global variables
// ---------------------------------------------------------------------------

func TestGlobalVariableDeclaration(t *testing.T) {
	// The book's example: declare two globals and use them in one expression.
	evalProgram(t, `let a = 1; let b = 2; print a + b;`)
}

// A declaration without an initializer binds nil, per the book. This is the
// behaviour challenge 2 asks you to change to an error instead.
func TestGlobalVariableDefaultsToNil(t *testing.T) {
	got, err := evalTo(t, `let a;`).environment.Get(tokenForTest("a"))
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got != nil {
		t.Errorf("an uninitialised variable reads as %#v, want nil", got)
	}
}

// Redefinition in the same scope overwrites. The book allows this at the top
// level so that the REPL and a script behave the same way.
func TestGlobalRedefinitionOverwrites(t *testing.T) {
	e := NewEvaluator()
	p := parser.NewParser(lexer.NewLexer(`let a = "before";`).ScanTokens())
	stmts, _ := p.Parse()
	if err := e.Eval(stmts); err != nil {
		t.Fatalf("Eval() error = %v", err)
	}

	before, err := e.environment.Get(tokenForTest("a"))
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if before != "before" {
		t.Errorf("after first declaration a = %v, want \"before\"", before)
	}

	p = parser.NewParser(lexer.NewLexer(`let a = "after";`).ScanTokens())
	stmts, _ = p.Parse()
	if err := e.Eval(stmts); err != nil {
		t.Fatalf("Eval() error = %v", err)
	}

	after, err := e.environment.Get(tokenForTest("a"))
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if after != "after" {
		t.Errorf("after redefinition a = %v, want \"after\" (later declaration wins)", after)
	}
}

func TestUseBeforeDeclarationIsARuntimeError(t *testing.T) {
	// The book defers this to runtime rather than making it a syntax error,
	// because mentioning a name is not the same as evaluating it — that is what
	// makes recursion possible in chapter 11.
	err := runProgram(t, `print a; let a = "too late!";`)
	if err == nil {
		t.Fatal("want an error for using a variable before declaring it")
	}
	if !strings.Contains(err.Error(), "Undefined Variable 'a'") {
		t.Errorf("error = %q, want the undefined-variable error", err)
	}
}

// ---------------------------------------------------------------------------
// 8.4 Assignment
// ---------------------------------------------------------------------------

func TestAssignmentMutatesExistingVariable(t *testing.T) {
	evalProgram(t, `let a = 1; a = 2; print a;`)

	// Assignment returns the assigned value, so it works inside an expression.
	if got := evalPrint(t, `let a = 1; print a = 2;`); got != "2\n" {
		t.Errorf("print a = 2 printed %q, want \"2\\n\"", got)
	}
}

// Assignment must not create a name. Declaring and assigning are distinct:
// `a = 1` with no prior `let a` is an error, not an implicit declaration.
func TestAssignmentToUndeclaredIsAnError(t *testing.T) {
	err := runProgram(t, `a = 1;`)
	if err == nil {
		t.Fatal("want an error assigning to an undeclared variable")
	}
	if !strings.Contains(err.Error(), "Undefined Variable 'a'") {
		t.Errorf("error = %q, want the undefined-variable error", err)
	}
}

func TestAssignmentIsRightAssociative(t *testing.T) {
	// The parser recurses into assignment() for the right-hand side, so this
	// chains. Each level returns its value, so the whole thing evaluates to 1.
	e := NewEvaluator()
	p := parser.NewParser(lexer.NewLexer(`let a; let b; a = b = 1;`).ScanTokens())
	stmts, _ := p.Parse()
	if errs := p.Errors(); len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if err := e.Eval(stmts); err != nil {
		t.Fatalf("Eval() error = %v", err)
	}

	for _, name := range []string{"a", "b"} {
		got, err := e.environment.Get(tokenForTest(name))
		if err != nil {
			t.Fatalf("Get(%s) error = %v", name, err)
		}
		if got != float64(1) {
			t.Errorf("%s = %v, want 1", name, got)
		}
	}
}

// The left-hand side of `=` must be a bare name. These are all rejected by the
// parser with "Invalid assignment target." — the r-value was parsed as an
// expression, then rejected once the parser saw the `=`.
func TestInvalidAssignmentTargets(t *testing.T) {
	for _, source := range []string{
		`(a) = 1;`,   // grouping is not an l-value
		`1 = 2;`,     // a literal is not an l-value
		`a + b = 3;`, // a binary result is not an l-value
		`"s" = 1;`,   // neither is a string
	} {
		t.Run(source, func(t *testing.T) {
			p := parser.NewParser(lexer.NewLexer(source).ScanTokens())
			p.Parse()

			errs := p.Errors()
			if len(errs) == 0 {
				t.Fatalf("parse(%q): want a syntax error, got none", source)
			}
			if !strings.Contains(errs[0].Error(), "Invalid assignment target.") {
				t.Errorf("parse(%q): error = %q, want \"Invalid assignment target.\"", source, errs[0])
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 8.5 Scope and blocks
// ---------------------------------------------------------------------------

func TestBlockScopeIsLocal(t *testing.T) {
	// A name declared inside a block does not exist outside it.
	err := runProgram(t, `{ let q = 1; } print q;`)
	if err == nil {
		t.Fatal("want an error: q should not be visible outside its block")
	}
	if !strings.Contains(err.Error(), "Undefined Variable 'q'") {
		t.Errorf("error = %q, want the undefined-variable error", err)
	}
}

// The book warns that tracking-and-deleting fails here: the block shadows a
// global, and deleting on exit would destroy the global. The enclosing chain
// keeps them separate.
func TestBlockShadowsWithoutDestroyingOuter(t *testing.T) {
	// The book's own example: a global volume, zeroed, then a block declaring
	// its own volume. After the block the global must still be 0, not gone.
	source := `let volume = 11;
volume = 0;
{ let volume = 3 * 4 * 5; print volume; }
print volume;`

	if got := evalPrint(t, source); got != "60\n0\n" {
		t.Errorf("printed %q, want \"60\\n0\\n\"", got)
	}
}

func TestBlockReadsEnclosingScope(t *testing.T) {
	source := `let global = "outside"; { let local = "inside"; print global + local; }`
	if got := evalPrint(t, source); got != "outsideinside\n" {
		t.Errorf("printed %q, want \"outsideinside\\n\"", got)
	}
}

// Chapter challenge 3: what does this do?
//
//	let a = 1;
//	{ let a = a + 2; print a; }
//
// Chapter 8 answered that the initializer runs before the inner a is
// defined, so it reads the outer a (1) and the inner a becomes 3.
//
// Chapter 11 changes the answer. The inner a shadows the outer one, so
// reading the name inside its own initializer reads a variable that is not
// ready yet. The resolver rejects that before the program runs, which is
// where the book wants this kind of mistake caught.
func TestBlockInitializerCannotReadOuterBinding(t *testing.T) {
	source := `let a = 1; { let a = a + 2; print a; } print a;`

	err := runProgram(t, source)
	if err == nil {
		t.Fatal("want an error reading a shadowed name in its own initializer")
	}
	if !strings.Contains(err.Error(), "Can't read local variable in its own initializer") {
		t.Errorf("error = %q, want the own-initializer error", err)
	}
}

func TestNestedBlocks(t *testing.T) {
	source := `let a = 1; { let a = 2; { let a = 3; print a; } print a; } print a;`

	if got := evalPrint(t, source); got != "3\n2\n1\n" {
		t.Errorf("printed %q, want \"3\\n2\\n1\\n\"", got)
	}
}

func TestEmptyBlock(t *testing.T) {
	// A block with no statements is legal. Note there is no empty-statement
	// rule in the grammar, so a bare ";" inside a block is a syntax error —
	// TestStraySemicolonIsASyntaxError pins that.
	evalProgram(t, `{}`)
	evalProgram(t, `{ }`)
	evalProgram(t, `{ print 1; }`)
}

// A runtime error part-way through a block must leave the evaluator in the
// outer scope. VisitBlockStmt restores with a defer, so an error unwinding
// through it still restores.
//
// Undefined-variable is the error here; like every runtime error (including
// 1/0, which returns an error rather than panicking) it exercises the error
// *return* path out of VisitBlockStmt.
func TestBlockScopeRestoredAfterError(t *testing.T) {
	e := NewEvaluator()

	// Seed a global so there is something to resolve after the failed block.
	seed := parser.NewParser(lexer.NewLexer(`let a = 1;`).ScanTokens())
	stmts, _ := seed.Parse()
	if err := e.Eval(stmts); err != nil {
		t.Fatalf("seeding Eval() error = %v", err)
	}

	// A block whose only statement fails with a returned error.
	bad := parser.NewParser(lexer.NewLexer(`{ print missing; }`).ScanTokens())
	stmts, _ = bad.Parse()

	var evalErr error
	captureOutput(t, func() { evalErr = e.Eval(stmts) })
	if evalErr == nil {
		t.Fatal("want an error for the undefined variable inside the block")
	}

	// The outer binding must still be reachable. If VisitBlockStmt failed to
	// restore, the evaluator would still be scoped to the dead block and this
	// lookup would walk the wrong chain.
	got, err := e.environment.Get(tokenForTest("a"))
	if err != nil {
		t.Fatalf("after a failed block, Get(a) error = %v", err)
	}
	if got != float64(1) {
		t.Errorf("after a failed block, a = %v, want 1", got)
	}
}

// The same evaluator must be reusable after a failed block, the way the REPL
// keeps running between lines.
func TestEvaluatorReusableAfterBlockError(t *testing.T) {
	e := NewEvaluator()

	bad := parser.NewParser(lexer.NewLexer(`{ print missing; }`).ScanTokens())
	stmts, _ := bad.Parse()
	captureOutput(t, func() { _ = e.Eval(stmts) })

	good := parser.NewParser(lexer.NewLexer(`let a = "still working"; print a;`).ScanTokens())
	stmts, _ = good.Parse()

	out := captureOutput(t, func() {
		if err := e.Eval(stmts); err != nil {
			t.Fatalf("second Eval() error = %v", err)
		}
	})

	if out != "still working\n" {
		t.Errorf("after a failed block, printed %q, want \"still working\\n\"", out)
	}
}

// Statements after a runtime error must not run. VisitBlockStmt returns on the
// first error, so the trailing print never executes.
func TestBlockStopsAtFirstError(t *testing.T) {
	source := `{ print "before"; print 1 / 0; print "after"; }`

	p := parser.NewParser(lexer.NewLexer(source).ScanTokens())
	stmts, _ := p.Parse()
	if errs := p.Errors(); len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	var err error
	out := captureOutput(t, func() { err = NewEvaluator().Eval(stmts) })

	if err == nil {
		t.Fatal("want a runtime error from 1 / 0")
	}
	if out != "before\n" {
		t.Errorf("printed %q, want \"before\\n\": statements after a runtime error must not run", out)
	}
}

// There is no empty-statement rule, so a stray semicolon is a syntax error.
// This differs from C, which allows it, and is a common surprise when moving
// code between the two.
func TestStraySemicolonIsASyntaxError(t *testing.T) {
	for _, source := range []string{`;`, `{ ; }`, `print 1;;`} {
		t.Run(source, func(t *testing.T) {
			p := parser.NewParser(lexer.NewLexer(source).ScanTokens())
			p.Parse()

			if len(p.Errors()) == 0 {
				t.Errorf("parse(%q): want a syntax error, got none", source)
			}
		})
	}
}

var _ = ast.Stmt(nil)
