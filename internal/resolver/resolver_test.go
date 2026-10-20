// The tests live in an external test package because they evaluate programs,
// and the evaluator depends on the resolver. A test in the resolver package
// itself would close that loop.
package resolver_test

import (
	"bytes"
	"strings"
	"testing"

	"magd/internal/ast"
	"magd/internal/eval"
	"magd/internal/lexer"
	"magd/internal/parser"
	"magd/internal/resolver"
)

// Chapter 11 coverage: name resolution.
//
// The resolver is only useful if the distances it records match the runtime
// environment chain exactly: scope N up at resolve time must be scope N up at
// run time. These tests therefore resolve and then evaluate, and check the
// printed result — a distance that is off by one shows up as the wrong value
// or as an undefined variable.
//
// run resolves source and then evaluates it, returning what was printed.
func run(t *testing.T, source string) (string, error) {
	t.Helper()

	p := parser.NewParser(lexer.NewLexer(source).ScanTokens())
	stmts, _ := p.Parse()
	if errs := p.Errors(); len(errs) != 0 {
		t.Fatalf("parse(%q): unexpected errors: %v", source, errs)
	}

	var out bytes.Buffer
	e := eval.NewEvaluatorTo(&out)

	r := resolver.New(map[ast.Expression]int{})
	if err := r.ResolveStatements(stmts); err != nil {
		return out.String(), err
	}
	if err := e.Eval(stmts); err != nil {
		return out.String(), err
	}
	return out.String(), nil
}

// evalProgram resolves and evaluates source expecting success.
func evalProgram(t *testing.T, source string) {
	t.Helper()

	got, err := run(t, source)
	if err != nil {
		t.Fatalf("eval(%q): unexpected error: %v", source, err)
	}
	_ = got
}

// evalPrints resolves and evaluates source, expecting exactly want to be
// printed.
func evalPrints(t *testing.T, source, want string) {
	t.Helper()

	got, err := run(t, source)
	if err != nil {
		t.Fatalf("eval(%q): unexpected error: %v", source, err)
	}
	if got != want {
		t.Errorf("eval(%q) printed %q, want %q", source, got, want)
	}
}

// ---------------------------------------------------------------------------
// 11.2 Resolving and binding
// ---------------------------------------------------------------------------

// A name in the global scope is not a local: the resolver leaves it alone and
// the runtime walks the environment chain to find it.
func TestGlobalNameIsNotResolvedAsALocal(t *testing.T) {
	evalProgram(t, `let a = 1; print a;`)
	evalProgram(t, `let a = 1; let b = 2; print a + b;`)
}

// A name declared in a block is a local of that block, one scope up from
// where it is read inside the block.
func TestBlockLocalResolvesToTheBlock(t *testing.T) {
	evalProgram(t, `{ let a = 1; print a; }`)
}

// The innermost binding wins, and resolving it must not disturb the outer one.
func TestShadowingResolvesToTheInnerBinding(t *testing.T) {
	source := `let a = 1; { let a = 2; print a; } print a;`

	got, err := run(t, source)
	if err != nil {
		t.Fatalf("eval(%q): unexpected error: %v", source, err)
	}
	if got != "2\n1\n" {
		t.Errorf("printed %q, want %q", got, "2\n1\n")
	}
}

// A block reads an outer global without shadowing it.
func TestBlockReadsTheGlobalWithoutShadowing(t *testing.T) {
	source := `let g = "out"; { print g; }`

	got, err := run(t, source)
	if err != nil {
		t.Fatalf("eval(%q): unexpected error: %v", source, err)
	}
	if got != "out\n" {
		t.Errorf("printed %q, want %q", got, "out\n")
	}
}

// ---------------------------------------------------------------------------
// 11.3 The resolver and the runtime environment
// ---------------------------------------------------------------------------

// A parameter is a local of the call it belongs to.
func TestParameterResolvesToTheCall(t *testing.T) {
	evalProgram(t, `fun f(x) { print x; } f(7);`)
}

// A name declared in a function body is a local of that body.
func TestFunctionBodyLocalResolvesToTheBody(t *testing.T) {
	evalProgram(t, `fun f() { let a = 1; print a; } f();`)
}

// A function body reads the global scope it was written in.
func TestFunctionReadsTheScopeItWasWrittenIn(t *testing.T) {
	source := `let g = "out"; fun f() { print g; } f();`

	got, err := run(t, source)
	if err != nil {
		t.Fatalf("eval(%q): unexpected error: %v", source, err)
	}
	if got != "out\n" {
		t.Errorf("printed %q, want %q", got, "out\n")
	}
}

// The closure case the chapter is about: `a` is declared two scopes above the
// closure body, so the recorded distance has to skip the call's own scope and
// the body block as well. Getting this wrong reads the wrong environment.
func TestClosureReadsTheCapturedVariable(t *testing.T) {
	source := `fun outer() { let a = 1; fun inner() { print a; } inner(); } outer();`

	got, err := run(t, source)
	if err != nil {
		t.Fatalf("eval(%q): unexpected error: %v", source, err)
	}
	if got != "1\n" {
		t.Errorf("printed %q, want %q", got, "1\n")
	}
}

// A closure writes back to the variable it captured.
func TestClosureWritesBackToTheCapturedVariable(t *testing.T) {
	source := `fun outer() { let a = 1; fun inner() { a = a + 1; } inner(); print a; } outer();`

	got, err := run(t, source)
	if err != nil {
		t.Fatalf("eval(%q): unexpected error: %v", source, err)
	}
	if got != "2\n" {
		t.Errorf("printed %q, want %q", got, "2\n")
	}
}

// The book's makeCounter: two calls, one captured counter.
func TestClosureKeepsCapturedStateAcrossCalls(t *testing.T) {
	source := `fun makeCounter() {
		let count = 0;
		fun inc() { count = count + 1; return count; }
		return inc;
	}
	let c = makeCounter();
	print c();
	print c();`

	got, err := run(t, source)
	if err != nil {
		t.Fatalf("eval(%q): unexpected error: %v", source, err)
	}
	if got != "1\n2\n" {
		t.Errorf("printed %q, want %q", got, "1\n2\n")
	}
}

// A function assigns to a local of an enclosing function. The assignment
// resolves to that enclosing local, not to a global of the same name.
func TestAssignmentResolvesToTheEnclosingFunctionLocal(t *testing.T) {
	source := `fun outer() { let a = 1; fun inner() { a = 5; } inner(); print a; } outer();`

	got, err := run(t, source)
	if err != nil {
		t.Fatalf("eval(%q): unexpected error: %v", source, err)
	}
	if got != "5\n" {
		t.Errorf("printed %q, want %q", got, "5\n")
	}
}

// A function's own name is in scope inside its body, which is what makes
// recursion work.
func TestFunctionNameIsVisibleInItsOwnBody(t *testing.T) {
	source := `fun fib(n) { if (n < 2) return n; return fib(n - 1) + fib(n - 2); } print fib(10);`

	got, err := run(t, source)
	if err != nil {
		t.Fatalf("eval(%q): unexpected error: %v", source, err)
	}
	if got != "55\n" {
		t.Errorf("printed %q, want %q", got, "55\n")
	}
}

// ---------------------------------------------------------------------------
// 11.4 The for loop scope
// ---------------------------------------------------------------------------

// The loop variable is a local of the loop, and the increment assigns to it.
func TestForLoopVariableResolvesToTheLoop(t *testing.T) {
	source := `for (let i = 0; i < 3; i = i + 1) print i;`

	got, err := run(t, source)
	if err != nil {
		t.Fatalf("eval(%q): unexpected error: %v", source, err)
	}
	if got != "0\n1\n2\n" {
		t.Errorf("printed %q, want %q", got, "0\n1\n2\n")
	}
}

// A for loop with no declaration of its own assigns to the outer variable,
// which stays global and is found by walking the chain.
func TestForLoopCanAssignToAnOuterVariable(t *testing.T) {
	source := `let sum = 0; for (let i = 1; i <= 4; i = i + 1) sum = sum + i; print sum;`

	got, err := run(t, source)
	if err != nil {
		t.Fatalf("eval(%q): unexpected error: %v", source, err)
	}
	if got != "10\n" {
		t.Errorf("printed %q, want %q", got, "10\n")
	}
}

// A while loop does not open a scope, so the variable it tests is the outer
// one.
func TestWhileLoopReadsTheOuterVariable(t *testing.T) {
	source := `let i = 0; while (i < 2) { print i; i = i + 1; }`

	got, err := run(t, source)
	if err != nil {
		t.Fatalf("eval(%q): unexpected error: %v", source, err)
	}
	if got != "0\n1\n" {
		t.Errorf("printed %q, want %q", got, "0\n1\n")
	}
}

// ---------------------------------------------------------------------------
// 11.5 Errors the resolver catches before the program runs
// ---------------------------------------------------------------------------

// Reading a local inside its own initializer is a compile-time error, not a
// runtime one.
func TestReadingALocalInItsOwnInitializerIsAnError(t *testing.T) {
	_, err := run(t, `{ let a = a; }`)
	if err == nil {
		t.Fatal("want an error reading a local in its own initializer")
	}
	if !strings.Contains(err.Error(), "Can't read local variable in its own initializer") {
		t.Errorf("error = %q, want the own-initializer error", err)
	}
}

// A block that redeclares a name shadows the outer one, so reading the name
// inside its own initializer reads the not-yet-initialized local. The
// resolver rejects that at compile time, where the runtime used to accept it.
func TestReadingAShadowedNameInItsOwnInitializerIsAnError(t *testing.T) {
	_, err := run(t, `let a = 1; { let a = a + 1; print a; }`)
	if err == nil {
		t.Fatal("want an error reading a shadowed name in its own initializer")
	}
	if !strings.Contains(err.Error(), "Can't read local variable in its own initializer") {
		t.Errorf("error = %q, want the own-initializer error", err)
	}
}

// At the top level the name is a global, not a local, so the resolver has
// nothing to complain about. The read fails at runtime instead, because the
// global is not defined until the declaration completes.
func TestReadingAGlobalInItsOwnInitializerFailsAtRuntime(t *testing.T) {
	_, err := run(t, `let a = a;`)
	if err == nil {
		t.Fatal("want an error reading a global in its own initializer")
	}
	if !strings.Contains(err.Error(), "Undefined Variable 'a'") {
		t.Errorf("error = %q, want the undefined-variable error", err)
	}
}

// Return is only legal inside a function.
func TestReturnAtTopLevelIsAnError(t *testing.T) {
	_, err := run(t, `return 1;`)
	if err == nil {
		t.Fatal("want an error for a top-level return")
	}
	if !strings.Contains(err.Error(), "Can't return from top-level code") {
		t.Errorf("error = %q, want the top-level return error", err)
	}
}

// Return inside a function is fine.
func TestReturnInsideAFunctionIsAllowed(t *testing.T) {
	evalProgram(t, `fun f() { return 1; } print f();`)
}

// ---------------------------------------------------------------------------
// 11.5 Resolution errors
// ---------------------------------------------------------------------------

// The bug this chapter exists to fix. The closure is written before the
// inner a is declared, so the name refers to the global a — and it keeps
// referring to it, no matter what the block declares later. Before the
// resolver, the second call printed "block".
func TestClosureKeepsTheBindingItWasWrittenWith(t *testing.T) {
	source := `let a = "global";
	{
		fun showA() { print a; }
		showA();
		let a = "block";
		showA();
	}`

	evalPrints(t, source, "global\nglobal\n")
}

// Declaring the same name twice in one local scope is a mistake. The earlier
// binding was meant to be assigned to, not shadowed.
func TestRedeclaringALocalIsAnError(t *testing.T) {
	_, err := run(t, `{ let a = 1; let a = 2; }`)
	if err == nil {
		t.Fatal("want an error redeclaring a local")
	}
	if !strings.Contains(err.Error(), "Already a variable with this name in this scope.") {
		t.Errorf("error = %q, want the duplicate-declaration error", err)
	}
}

// Two functions with the same name in one local scope collide the same way.
func TestRedeclaringAFunctionIsAnError(t *testing.T) {
	_, err := run(t, `{ fun f() {} fun f() {} }`)
	if err == nil {
		t.Fatal("want an error redeclaring a function")
	}
	if !strings.Contains(err.Error(), "Already a variable with this name in this scope.") {
		t.Errorf("error = %q, want the duplicate-declaration error", err)
	}
}

// A function and a variable share a namespace, so they collide too.
func TestRedeclaringAFunctionAsAVariableIsAnError(t *testing.T) {
	_, err := run(t, `{ let f = 1; fun f() {} }`)
	if err == nil {
		t.Fatal("want an error for a function and a variable sharing a name")
	}
	if !strings.Contains(err.Error(), "Already a variable with this name in this scope.") {
		t.Errorf("error = %q, want the duplicate-declaration error", err)
	}
}

// The global scope is exempt: the REPL leans on it, and the book allows it.
func TestRedeclaringAGlobalIsAllowed(t *testing.T) {
	evalPrints(t, `let a = 1; let a = 2; print a;`, "2\n")
}

// Two parameters with the same name collide: the second would shadow the
// first inside the body.
func TestDuplicateParameterIsAnError(t *testing.T) {
	_, err := run(t, `fun f(a, a) { return a; }`)
	if err == nil {
		t.Fatal("want an error for a duplicate parameter name")
	}
	if !strings.Contains(err.Error(), "Already a variable with this name in this scope.") {
		t.Errorf("error = %q, want the duplicate-declaration error", err)
	}
}

// ---------------------------------------------------------------------------
// The resolver has to walk into every node
// ---------------------------------------------------------------------------

// A name buried in an operand is still a name. Each case reads a local from
// a different position, so a resolver that forgot to recurse into some kind
// of expression would resolve it to the wrong scope — or not at all.
func TestResolverWalksEveryExpression(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{"unary operand", `{ let a = 1; print -a; }`, "-1\n"},
		{"grouping operand", `{ let a = 1; print (a); }`, "1\n"},
		{"conditional branch", `{ let a = 1; print true ? a : 0; }`, "1\n"},
		{"logical operand", `{ let a = 1; print a and 2; }`, "2\n"},
		{"call argument", `{ let a = 1; fun id(x) { return x; } print id(a); }`, "1\n"},
		{"function expression body", `{ let a = 1; let f = fun() { return a; }; print f(); }`, "1\n"},
		{"break in a loop", `{ let i = 0; while (true) { break; } print "broke"; }`, "broke\n"},
		{"continue in a loop", `{ let i = 0; while (i < 1) { i = i + 1; continue; } print "done"; }`, "done\n"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			evalPrints(t, tc.source, tc.want)
		})
	}
}

// The resolver has to walk into every kind of statement too. A name in a
// branch the program may not take is still resolved: the analysis is static,
// so it visits both branches.
func TestResolverWalksEveryStatement(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{"expression statement", `{ let a = 1; a; }`, ""},
		{"if branch", `{ let a = true; if (a) { print "then"; } }`, "then\n"},
		{"else branch", `{ let a = false; if (a) { } else { print "else"; } }`, "else\n"},
		{"while body", `{ let a = true; while (a) { print "body"; a = false; } }`, "body\n"},
		{"for body", `{ for (let i = 0; i < 1; i = i + 1) { print "body"; } }`, "body\n"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			evalPrints(t, tc.source, tc.want)
		})
	}
}

// A resolution error is found wherever it is buried. The resolver reports
// the first one it meets on the way down.
func TestResolutionErrorIsFoundInsideAnyExpression(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{"unary operand", `{ let a = -a; }`},
		{"grouping operand", `{ let a = (a); }`},
		{"conditional branch", `{ let a = true ? a : a; }`},
		{"logical operand", `{ let a = a and a; }`},
		{"call argument", `{ let a = f(a); }`},
		{"assignment value", `{ let a = 1; let b = (b = b); }`},
		{"function expression body", `{ let a = fun() { let b = b; }; }`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := run(t, tc.source)
			if err == nil {
				t.Fatal("want a resolution error")
			}
			if !strings.Contains(err.Error(), "Can't read local variable in its own initializer") {
				t.Errorf("error = %q, want the own-initializer error", err)
			}
		})
	}
}

var _ ast.Stmt = nil
