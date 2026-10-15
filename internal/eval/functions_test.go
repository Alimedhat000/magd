package eval

import (
	"strings"
	"testing"
)

// evalResult runs source and returns the value the program bound to name.
// Functions and their results both have to live in one program, so this
// evaluates a whole source string and then reads one binding out of it.
func evalResult(t *testing.T, source, name string) any {
	t.Helper()

	e := evalTo(t, source)

	value, err := e.environment.Get(tokenForTest(name))
	if err != nil {
		t.Fatalf("looking up %q: %v", name, err)
	}

	return value
}

func TestFunctionCallReturnsValue(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   any
	}{
		{
			"two arguments",
			`fun add(a, b) { return a + b; } let r = add(1, 2);`,
			float64(3),
		},
		{
			"strings pass through",
			`fun first(a, b) { return a; } let r = first("x", "y");`,
			"x",
		},
		{
			"a string result",
			`fun greet(name) { return "hi " + name; } let r = greet("ali");`,
			"hi ali",
		},
		{
			"booleans",
			`fun neg(v) { return !v; } let r = neg(true);`,
			false,
		},
		{
			// Reaching the end of the body is equivalent to a bare return.
			"no return at all yields nil",
			`fun nothing() { } let r = nothing();`,
			nil,
		},
		{
			"bare return yields nil",
			`fun bare() { return; } let r = bare();`,
			nil,
		},
		{
			"return nil explicitly",
			`fun explicit() { return nil; } let r = explicit();`,
			nil,
		},
		{
			"a nested expression in the return value",
			`fun dbl(n) { return n * 2 + 1; } let r = dbl(5);`,
			float64(11),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := evalResult(t, tt.source, "r"); got != tt.want {
				t.Errorf("got %v (%T), want %v (%T)", got, got, tt.want, tt.want)
			}
		})
	}
}

func TestFunctionParameters(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   any
	}{
		{
			// Parameters bind in order, not by name similarity.
			"bind positionally",
			`fun f(x, y, z) { return x + y + z; } let r = f(1, 2, 3);`,
			float64(6),
		},
		{
			"a later parameter does not clobber an earlier one",
			`fun dup(a, b) { return a; } let r = dup(1, 2);`,
			float64(1),
		},
		{
			"zero parameters",
			`fun constant() { return 42; } let r = constant();`,
			float64(42),
		},
		{
			// A body may rebind its own parameter: it is a local write.
			"assignment to a parameter stays local to the call",
			`fun bump(n) { n = n + 1; return n; } let r = bump(1);`,
			float64(2),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := evalResult(t, tt.source, "r"); got != tt.want {
				t.Errorf("got %v (%T), want %v (%T)", got, got, tt.want, tt.want)
			}
		})
	}
}

func TestFunctionRecursion(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   any
	}{
		{
			"factorial",
			`fun fact(n) { if (n <= 1) return 1; return n * fact(n - 1); } let r = fact(5);`,
			float64(120),
		},
		{
			"fibonacci",
			`fun fib(n) { if (n <= 1) return n; return fib(n - 2) + fib(n - 1); } let r = fib(10);`,
			float64(55),
		},
		{
			// Each activation binds its own n, so the mutation below cannot
			// reach the frames still waiting on the recursive call.
			"interleaved activations keep separate bindings",
			`fun f(n) { if (n > 0) f(n - 1); n = n + 100; return n; } let r = f(2);`,
			float64(102),
		},
		{
			// Recursion resolves the name through the closure, which is the
			// environment captured at declaration.
			"mutual visibility of the function's own name",
			`fun loop(n) { if (n == 0) return "done"; return loop(n - 1); } let r = loop(2);`,
			"done",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := evalResult(t, tt.source, "r"); got != tt.want {
				t.Errorf("got %v (%T), want %v (%T)", got, got, tt.want, tt.want)
			}
		})
	}
}

func TestFunctionArityMismatch(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{"too few", `fun add(a, b) { return a + b; } add(1);`},
		{"too many", `fun add(a, b) { return a + b; } add(1, 2, 3);`},
		{"none at all", `fun add(a, b) { return a + b; } add();`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runProgram(t, tt.source)
			if err == nil {
				t.Fatalf("eval(%q): want an arity error, got none", tt.source)
			}
			if !strings.Contains(err.Error(), "arguments but got") {
				t.Errorf("error = %q, want an arity mismatch message", err)
			}
		})
	}
}

func TestCallNonFunction(t *testing.T) {
	tests := []string{
		`1();`,
		`"text"();`,
		`true();`,
		`nil();`,
		`let f = 3; f();`,
	}

	for _, source := range tests {
		t.Run(source, func(t *testing.T) {
			err := runProgram(t, source)
			if err == nil {
				t.Fatalf("eval(%q): want an error, got none", source)
			}
			if !strings.Contains(err.Error(), "Can only call functions") {
				t.Errorf("error = %q, want the call-type message", err)
			}
		})
	}
}

func TestFunctionScope(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{"a parameter is not visible outside", `fun f(a) { return a; } f(1); print a;`},
		{"a body local is not visible outside", `fun f() { let hidden = 1; } f(); print hidden;`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runProgram(t, tt.source)
			if err == nil {
				t.Errorf("eval(%q): want an undefined-variable error, got none", tt.source)
			} else if !strings.Contains(err.Error(), "Undefined Variable") {
				t.Errorf("error = %q, want an undefined-variable error", err)
			}
		})
	}
}

// A function body sees the scope it was written in, not the one it is called
// from. show() is declared inside a block that shadows name, then returned and
// called from the top level.
func TestFunctionReadsTheScopeItWasWrittenIn(t *testing.T) {
	got := evalPrints(t, `
		fun make() {
			let name = "outer";
			{
				let name = "shadowed";
				fun show() { return name; }
				return show;
			}
		}
		print make()();`)

	if got != "shadowed\n" {
		t.Errorf("printed %q, want %q: the body must resolve 'name' where it was written", got, "shadowed\n")
	}
}

// The book notes the interpreter still has a dynamic-scoping leak: nothing
// distinguishes the global scope from any other. Pinning it so the leak is
// visible rather than hidden.
func TestFunctionBodyCannotSeeCallerLocals(t *testing.T) {
	err := runProgram(t, `
		fun peek() { return secret; }
		{ let secret = "caller's local"; peek(); }`)

	if err == nil {
		t.Fatal("want an undefined-variable error, got none")
	}
	if !strings.Contains(err.Error(), "Undefined Variable") {
		t.Errorf("error = %q, want an undefined-variable error", err)
	}
}

func TestFunctionRedefinition(t *testing.T) {
	// Globals may be redefined, which is what lets the REPL iterate on a
	// function: the second definition replaces the first.
	if got := evalResult(t, `
		fun f() { return 1; }
		fun f() { return 2; }
		let r = f();`, "r"); got != float64(2) {
		t.Errorf("got %v, want 2: a redefinition must replace the earlier binding", got)
	}
}

func TestReturnUnwindsLoops(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   any
	}{
		{
			// return exits the loop and the function, so nothing after runs.
			"out of a while",
			`fun f() { let i = 0; while (i < 10) { i = i + 1; if (i == 3) return i; } return 0; }
			 let r = f();`,
			float64(3),
		},
		{
			"out of a for",
			`fun f() { for (let i = 0; i < 10; i = i + 1) { if (i == 2) return i; } return 0; }
			 let r = f();`,
			float64(2),
		},
		{
			"out of a for body block",
			`fun f() { for (let i = 0; i < 5; i = i + 1) { if (i == 4) { return i; } } return 0; }
			 let r = f();`,
			float64(4),
		},
		{
			"out of a nested block",
			`fun f() { { { return "deep"; } } return "shallow"; }
			 let r = f();`,
			"deep",
		},
		{
			// break would stop the loop; return stops the function.
			"out of a loop nested in a block",
			`fun f() { { let i = 0; while (true) { i = i + 1; if (i > 2) return i; } } return 0; }
			 let r = f();`,
			float64(3),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := evalResult(t, tt.source, "r"); got != tt.want {
				t.Errorf("got %v (%T), want %v (%T)", got, got, tt.want, tt.want)
			}
		})
	}
}

// return is caught at the function boundary, so a signal crossing two frames
// is the innermost function's return value.
func TestReturnStopsAtTheNearestFunction(t *testing.T) {
	if got := evalPrints(t, `
		fun inner() { return "inner"; }
		fun outer() { inner(); return "outer"; }
		print outer();`); got != "outer\n" {
		t.Errorf("printed %q, want %q", got, "outer\n")
	}
}

func TestClosures(t *testing.T) {
	t.Run("the book's makeCounter", func(t *testing.T) {
		got := evalPrints(t, `
			fun makeCounter() {
				let i = 0;
				fun count() { i = i + 1; print i; }
				return count;
			}
			let counter = makeCounter();
			counter();
			counter();`)

		if got != "1\n2\n" {
			t.Errorf("printed %q, want %q: the closure must keep its state across calls", got, "1\n2\n")
		}
	})

	t.Run("a closure outlives its defining scope", func(t *testing.T) {
		if got := evalResult(t, `
			fun make() { let x = "captured"; fun get() { return x; } return get; }
			let f = make();
			let r = f();`, "r"); got != "captured" {
			t.Errorf("got %v, want %q", got, "captured")
		}
	})

	t.Run("two closures over separate state", func(t *testing.T) {
		got := evalPrints(t, `
			fun makeCounter() { let i = 0; fun count() { i = i + 1; return i; } return count; }
			let a = makeCounter();
			let b = makeCounter();
			print a();
			print a();
			print b();`)

		if got != "1\n2\n1\n" {
			t.Errorf("printed %q, want %q: separate calls must not share state", got, "1\n2\n1\n")
		}
	})

	t.Run("a closure captures a block's local", func(t *testing.T) {
		if got := evalResult(t, `
			fun grab() { let fn = nil; { let captured = "block local"; fun use() { return captured; } fn = use; } return fn; }
			let f = grab();
			let r = f();`, "r"); got != "block local" {
			t.Errorf("got %v, want %q", got, "block local")
		}
	})

	t.Run("a closure writes back to its captured variable", func(t *testing.T) {
		if got := evalResult(t, `
			fun makeCounter() { let i = 0; fun count() { i = i + 1; return i; } return count; }
			let c = makeCounter();
			c(); c();
			let r = c();`, "r"); got != float64(3) {
			t.Errorf("got %v, want 3: assignment inside the closure must reach the captured variable", got)
		}
	})
}

func TestReturnAtTopLevel(t *testing.T) {
	tests := []string{
		`return 5;`,
		`return;`,
		`return "x";`,
	}

	for _, source := range tests {
		t.Run(source, func(t *testing.T) {
			err := runProgram(t, source)
			if err == nil {
				t.Fatalf("eval(%q): want an error, got none", source)
			}
			if !strings.Contains(err.Error(), "Can't return from top-level code") {
				t.Errorf("error = %q, want the top-level return message", err)
			}
		})
	}
}

// A return inside a function must not be reported as a top-level return.
func TestReturnInsideFunctionIsNotTopLevel(t *testing.T) {
	err := runProgram(t, `fun f() { return 1; } f();`)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestFunctionErrorsPropagate(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{"divide by zero in the body", `fun f() { return 1 / 0; } f();`},
		{"undefined variable in the body", `fun f() { return nope; } f();`},
		{"error on a later line of the body", `fun f() { print 1; return "a" + 1; } f();`},
		{"error after a nested call", `fun inner() { return missing; } fun outer() { return inner(); } outer();`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := runProgram(t, tt.source); err == nil {
				t.Errorf("eval(%q): want a runtime error, got none", tt.source)
			}
		})
	}
}

// The book's interpreter pushes arguments onto a stack, which evaluates them
// right to left. This one walks a slice, so arguments evaluate left to right.
// Pinned so nobody "fixes" it to match the book.
func TestArgumentsEvaluateLeftToRight(t *testing.T) {
	got := evalPrints(t, `
		fun note(n) { print n; return n; }
		fun add(a, b) { return a + b; }
		add(note(1), note(2));`)

	if got != "1\n2\n" {
		t.Errorf("printed %q, want %q: arguments must evaluate in source order", got, "1\n2\n")
	}
}

// The callee expression is evaluated before the arguments. Both effects here
// are assignments, so the order is observable without printing from inside a
// function body.
func TestCalleeEvaluatesBeforeArguments(t *testing.T) {
	got := evalPrints(t, `
		let order = "";
		fun pick(n) { return n; }
		fun makeIt(n) { order = order + "c"; return pick; }
		fun tick() { order = order + "a"; return 7; }
		makeIt(0)(tick());
		print order;`)

	if got != "ca\n" {
		t.Errorf("printed %q, want %q: the callee must be evaluated before the arguments", got, "ca\n")
	}
}

func TestStringifyFunction(t *testing.T) {
	got := evalPrints(t, `
		fun greet(name) { return name; }
		print greet;`)

	if got != "<fun greet>\n" {
		t.Errorf("printed %q, want %q", got, "<fun greet>\n")
	}
}

// Two declarations of different kinds coexist: Lox is a Lisp-1, so functions
// and variables share one namespace and a redeclaration replaces.
func TestFunctionsAndVariablesShareANamespace(t *testing.T) {
	if got := evalResult(t, `
		let x = 1;
		fun x() { return 2; }
		let r = x();`, "r"); got != float64(2) {
		t.Errorf("got %v, want 2: a function declaration must rebind the name", got)
	}
}
