package eval

// Chapter 12 and 13 coverage: classes, instances, methods, inheritance, and
// super. Each test drives the real pipeline — lex, parse, resolve, evaluate —
// so a scope distance that is off by one shows up as a failure here.

import (
	"bytes"
	"strings"
	"testing"
)

// classRuntimeError runs source expecting a runtime error containing want.
func classRuntimeError(t *testing.T, source, want string) {
	t.Helper()

	err := runProgram(t, source)
	if err == nil {
		t.Fatalf("eval(%q): want a runtime error, got none", source)
	}
	if !strings.Contains(err.Error(), want) {
		t.Errorf("eval(%q) error = %q, want one containing %q", source, err, want)
	}
}

// evalBinding reads one global from an evaluator that has already run. Use it
// when a test needs two bindings from the same run: evalResult reruns the
// program, so two of its results never share an object.
func evalBinding(t *testing.T, e *Evaluator, name string) any {
	t.Helper()

	value, err := e.environment.Get(tokenForTest(name))
	if err != nil {
		t.Fatalf("looking up %q: %v", name, err)
	}
	return value
}

func TestClassCallCreatesInstance(t *testing.T) {
	got := evalResult(t, `class Foo {} let f = Foo();`, "f")

	instance, ok := got.(*Instance)
	if !ok {
		t.Fatalf("f = %T, want *Instance", got)
	}
	if instance.String() != "Foo instance" {
		t.Errorf("f = %q, want %q", instance.String(), "Foo instance")
	}
}

func TestInstanceFields(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   any
	}{
		{
			"read back what was set",
			`class Foo {} let f = Foo(); f.x = 7; let r = f.x;`,
			float64(7),
		},
		{
			"a field shadows a method",
			`class Foo { m() { return 1; } } let f = Foo(); f.m = 2; let r = f.m;`,
			float64(2),
		},
		{
			"later assignment wins",
			`class Foo {} let f = Foo(); f.x = "a"; f.x = "b"; let r = f.x;`,
			"b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := evalResult(t, tt.source, "r"); got != tt.want {
				t.Errorf("r = %v (%T), want %v (%T)", got, got, tt.want, tt.want)
			}
		})
	}
}

func TestMethodsBindThis(t *testing.T) {
	source := `class Foo { who() { return this; } } let f = Foo(); let r = f.who();`

	e := evalTo(t, source)
	f, ok := evalBinding(t, e, "f").(*Instance)
	if !ok {
		t.Fatal("f is not an instance")
	}
	r, ok := evalBinding(t, e, "r").(*Instance)
	if !ok {
		t.Fatal("r is not an instance")
	}

	if r != f {
		t.Errorf("r = %v, want the same instance as f (%v)", r, f)
	}
}

func TestMethodMutatesFieldsThroughThis(t *testing.T) {
	source := `class Foo {
		set() { this.x = 3; }
		get() { return this.x; }
	}
	let f = Foo();
	f.set();
	let r = f.get();`

	if got := evalResult(t, source, "r"); got != float64(3) {
		t.Errorf("r = %v, want 3", got)
	}
}

func TestInitializer(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   any
	}{
		{
			"runs on construction",
			`class Foo { init(x) { this.x = x; } } let r = Foo(7).x;`,
			float64(7),
		},
		{
			"no initializer means no arguments",
			`class Foo { m() { return 1; } } let r = Foo().m();`,
			float64(1),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := evalResult(t, tt.source, "r"); got != tt.want {
				t.Errorf("r = %v (%T), want %v (%T)", got, got, tt.want, tt.want)
			}
		})
	}
}

// A bare return in an initializer is legal; jlox only rejects return with a
// value.
func TestInitializerBareReturnStillConstructs(t *testing.T) {
	got := evalResult(t, `class Foo { init() { return; } } let r = Foo();`, "r")
	if _, ok := got.(*Instance); !ok {
		t.Errorf("r = %T, want *Instance", got)
	}
}

// An initializer always hands back the instance, even when called directly.
func TestInitializerReturnsTheInstance(t *testing.T) {
	source := `class Foo { init() {} } let f = Foo(); let r = f.init();`

	e := evalTo(t, source)
	r := evalBinding(t, e, "r")
	f := evalBinding(t, e, "f")
	if r != f {
		t.Errorf("r = %v, want the same instance as f (%v)", r, f)
	}
}

func TestInitializerArity(t *testing.T) {
	classRuntimeError(t,
		`class Foo { init(x) {} } Foo();`,
		"Expected 1 arguments but got 0.")
	classRuntimeError(t,
		`class Foo {} Foo(1);`,
		"Expected 0 arguments but got 1.")
}

func TestInheritance(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   any
	}{
		{
			"method is inherited",
			`class A { m() { return "A"; } } class B < A {} let r = B().m();`,
			"A",
		},
		{
			"override wins",
			`class A { m() { return "A"; } } class B < A { m() { return "B"; } } let r = B().m();`,
			"B",
		},
		{
			"initializer is inherited",
			`class A { init(x) { this.x = x; } } class B < A {} let r = B(3).x;`,
			float64(3),
		},
		{
			"the base method still works",
			`class A { m() { return "A"; } } class B < A { m() { return "B"; } } let r = A().m();`,
			"A",
		},
		{
			"this dispatches to the subclass override",
			`class A { m() { return this.n(); } n() { return "A"; } } class B < A { n() { return "B"; } } let r = B().m();`,
			"B",
		},
		{
			"lookup walks up several levels",
			`class A { m() { return "A"; } } class B < A {} class C < B {} let r = C().m();`,
			"A",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := evalResult(t, tt.source, "r"); got != tt.want {
				t.Errorf("r = %v (%T), want %v (%T)", got, got, tt.want, tt.want)
			}
		})
	}
}

func TestSuper(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   any
	}{
		{
			"calls the base method",
			`class A { m() { return "A"; } } class B < A { m() { return super.m() + "B"; } } let r = B().m();`,
			"AB",
		},
		{
			"skips the direct override",
			`class A { m() { return "A"; } } class B < A { m() { return "B"; } } class C < B { m() { return super.m(); } } let r = C().m();`,
			"B",
		},
		{
			"sees this from the instance",
			`class A { m() { return this.x; } } class B < A { init() { this.x = "x"; } m() { return super.m(); } } let r = B().m();`,
			"x",
		},
		{
			"works inside a nested function",
			`class A { m() { return "A"; } }
			 class B < A { m() { let f = fun() { return super.m(); }; return f(); } }
			 let r = B().m();`,
			"A",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := evalResult(t, tt.source, "r"); got != tt.want {
				t.Errorf("r = %v (%T), want %v (%T)", got, got, tt.want, tt.want)
			}
		})
	}
}

// Instances, like functions and classes, compare by identity: two objects are
// equal only when they are the same object.
func TestInstanceEquality(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   bool
	}{
		{"same instance", `class Foo {} let f = Foo(); let r = f == f;`, true},
		{"same fields, different instances", `class Foo {} let r = Foo() == Foo();`, false},
		{"different classes", `class A {} class B {} let r = A() == B();`, false},
		{"nil is never an instance", `class Foo {} let r = Foo() == nil;`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := evalResult(t, tt.source, "r"); got != tt.want {
				t.Errorf("r = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClassPrintString(t *testing.T) {
	var buf bytes.Buffer
	if err := evalToBuf(t, &buf, `class Foo {} print Foo();`); err != nil {
		t.Fatalf("Eval() error = %v", err)
	}
	if got, want := buf.String(), "Foo instance\n"; got != want {
		t.Errorf("printed %q, want %q", got, want)
	}
}

func TestClassErrors(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{
			"superclass must be a class",
			`let A = 1; class B < A {}`,
			"Superclass must be a class.",
		},
		{
			"undefined superclass",
			`class B < A {}`,
			"Undefined Variable 'A'.",
		},
		{
			"undefined property",
			`class Foo {} let r = Foo().x;`,
			"Undefined property 'x'.",
		},
		{
			"undefined base method through super",
			`class A {} class B < A { m() { return super.nope(); } } B().m();`,
			"Undefined property 'nope'.",
		},
		{
			"only instances have properties",
			`1.x;`,
			"Only instances have properties.",
		},
		{
			"only instances accept fields",
			`1.x = 1;`,
			"Only instances have properties.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			classRuntimeError(t, tt.source, tt.want)
		})
	}
}
