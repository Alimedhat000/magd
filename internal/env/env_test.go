package env

import (
	"testing"

	"magd/internal/token"
)

// name builds the token Get and Assign take. They read the lexeme as the key
// and the line for error messages, so both fields matter.
func name(lexeme string) token.Token {
	return token.Token{Type: token.IDENT, Lexeme: lexeme, Line: 3}
}

func TestDefineThenGet(t *testing.T) {
	tests := []struct {
		name  string
		value any
	}{
		{"number", float64(42)},
		{"string", "hello"},
		{"bool", true},
		{"nil", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewEnvironment()
			e.Define("x", tt.value)

			got, err := e.Get(name("x"))
			if err != nil {
				t.Fatalf("Get() error = %v", err)
			}
			if got != tt.value {
				t.Errorf("Get() = %#v, want %#v", got, tt.value)
			}
		})
	}
}

// A name defined as nil must still be found. This is why Get checks presence
// in the map rather than testing the value for nil.
func TestGetDistinguishesUndefinedFromDefinedAsNil(t *testing.T) {
	e := NewEnvironment()
	e.Define("x", nil)

	got, err := e.Get(name("x"))
	if err != nil {
		t.Fatalf("Get() error = %v, want a defined nil to be found", err)
	}
	if got != nil {
		t.Errorf("Get() = %#v, want nil", got)
	}
}

func TestGetUndefined(t *testing.T) {
	e := NewEnvironment()

	_, err := e.Get(name("missing"))
	if err == nil {
		t.Fatal("want an error for an undefined variable")
	}

	want := "runtime error on line 3 at 'missing': Undefined Variable 'missing'."
	if got := err.Error(); got != want {
		t.Errorf("error = %q, want %q", got, want)
	}
}

// Redefining in the same scope overwrites. The book makes this an error, but
// this implementation is a single flat scope, so last write wins — pinned here
// so the behaviour is deliberate rather than accidental.
func TestDefineOverwritesInSameScope(t *testing.T) {
	e := NewEnvironment()
	e.Define("x", float64(1))
	e.Define("x", float64(2))

	got, err := e.Get(name("x"))
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got != float64(2) {
		t.Errorf("Get() = %v, want 2 (the later Define)", got)
	}
}

func TestAssignExisting(t *testing.T) {
	e := NewEnvironment()
	e.Define("x", float64(1))

	got, err := e.Assign(name("x"), float64(99))
	if err != nil {
		t.Fatalf("Assign() error = %v", err)
	}

	// Assign returns the assigned value: assignment is an expression, so
	// "a = b = 1" needs each level to yield its value.
	if got != float64(99) {
		t.Errorf("Assign() = %v, want 99 (the assigned value)", got)
	}

	stored, err := e.Get(name("x"))
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if stored != float64(99) {
		t.Errorf("Get() = %v, want 99 (Assign must write through)", stored)
	}
}

func TestAssignUndefined(t *testing.T) {
	e := NewEnvironment()

	_, err := e.Assign(name("missing"), float64(1))
	if err == nil {
		t.Fatal("want an error assigning to an undefined variable")
	}

	want := "runtime error on line 3 at 'missing': Undefined Variable 'missing'."
	if got := err.Error(); got != want {
		t.Errorf("error = %q, want %q", got, want)
	}

	// A failed assign must not create the name.
	if _, err := e.Get(name("missing")); err == nil {
		t.Error("Assign created the variable despite failing")
	}
}

func TestEnvironmentsAreIndependent(t *testing.T) {
	a, b := NewEnvironment(), NewEnvironment()
	a.Define("x", float64(1))

	if _, err := b.Get(name("x")); err == nil {
		t.Error("b saw a's binding; the environments are sharing state")
	}
}

// Define on a zero-value Environment must not panic. A nil map is a valid
// zero value for map[string]any, and writing to one panics — so this pins the
// requirement that callers use NewEnvironment.
func TestZeroValueDefinePanicsWithoutConstructor(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("want a panic: zero-value Environment has a nil map")
		}
	}()

	var e Environment
	e.Define("x", float64(1))
}

// ---------------------------------------------------------------------------
// Scoping: the enclosing chain
// ---------------------------------------------------------------------------

func TestGetFindsEnclosingBinding(t *testing.T) {
	global := NewEnvironment()
	global.Define("a", float64(1))

	inner := NewEnvironmentEnclosing(global)
	inner.Define("b", float64(2))

	// "a" is not in the inner scope, so lookup walks outward.
	got, err := inner.Get(name("a"))
	if err != nil {
		t.Fatalf("Get(a) error = %v", err)
	}
	if got != float64(1) {
		t.Errorf("Get(a) = %v, want 1", got)
	}

	if got, err := inner.Get(name("b")); err != nil || got != float64(2) {
		t.Errorf("Get(b) = %v, %v; want 2", got, err)
	}
}

// The innermost scope wins. Starting the walk at the inner scope is what makes
// a local declaration shadow an outer one.
func TestInnerScopeShadowsOuter(t *testing.T) {
	global := NewEnvironment()
	global.Define("a", "outer")

	inner := NewEnvironmentEnclosing(global)
	inner.Define("a", "inner")

	got, err := inner.Get(name("a"))
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got != "inner" {
		t.Errorf("Get(a) = %v, want \"inner\" (the inner scope wins)", got)
	}

	// Shadowing hides the outer binding from the inner scope; it does not
	// destroy it.
	outer, err := global.Get(name("a"))
	if err != nil {
		t.Fatalf("Get() on the global scope error = %v", err)
	}
	if outer != "outer" {
		t.Errorf("global a = %v, want \"outer\": shadowing must not modify the outer binding", outer)
	}
}

// Define always writes to the current scope, never to an enclosing one.
func TestDefineStaysInCurrentScope(t *testing.T) {
	global := NewEnvironment()
	global.Define("a", float64(1))

	inner := NewEnvironmentEnclosing(global)
	inner.Define("a", float64(2))

	got, err := global.Get(name("a"))
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got != float64(1) {
		t.Errorf("global a = %v, want 1: Define must not write to an enclosing scope", got)
	}
}

// Assignment walks the chain to find an existing binding, so assigning from
// inside a block modifies the outer variable.
func TestAssignReachesEnclosingScope(t *testing.T) {
	global := NewEnvironment()
	global.Define("volume", float64(11))

	inner := NewEnvironmentEnclosing(global)
	if _, err := inner.Assign(name("volume"), float64(0)); err != nil {
		t.Fatalf("Assign() error = %v", err)
	}

	got, err := global.Get(name("volume"))
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got != float64(0) {
		t.Errorf("global volume = %v, want 0: assignment must reach the enclosing scope", got)
	}
}

// Assign must not create a name in the inner scope when it cannot find one
// anywhere — it reports the error instead.
func TestAssignUndefinedDoesNotCreateAnywhere(t *testing.T) {
	global := NewEnvironment()
	inner := NewEnvironmentEnclosing(global)

	if _, err := inner.Assign(name("nope"), float64(1)); err == nil {
		t.Fatal("want an error assigning an undefined name")
	}

	if _, ok := inner.values["nope"]; ok {
		t.Error("Assign created the name in the inner scope despite failing")
	}
	if _, ok := global.values["nope"]; ok {
		t.Error("Assign created the name in the outer scope despite failing")
	}
}

func TestGetThroughMultipleScopes(t *testing.T) {
	level1 := NewEnvironment()
	level1.Define("deep", "found")

	level2 := NewEnvironmentEnclosing(level1)
	level3 := NewEnvironmentEnclosing(level2)
	level4 := NewEnvironmentEnclosing(level3)

	got, err := level4.Get(name("deep"))
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got != "found" {
		t.Errorf("Get(deep) = %v, want \"found\"", got)
	}
}

// A global environment has no enclosing scope, so an undefined name reports an
// error rather than panicking on a nil dereference.
func TestGlobalScopeHasNoEnclosing(t *testing.T) {
	if NewEnvironment().enclosing != nil {
		t.Error("a global environment must have no enclosing scope")
	}
	if NewEnvironmentEnclosing(NewEnvironment()).enclosing == nil {
		t.Error("a local environment must record its enclosing scope")
	}
}
