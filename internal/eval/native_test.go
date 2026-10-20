package eval

import (
	"bytes"
	"strings"
	"testing"
)

// Chapter 10 coverage: native functions.
//
// A native function is implemented in Go and handed to Lox code as a value.
// It lives in the global scope, so Lox code reaches it by name and can shadow
// it with a binding of its own.

// evalNative runs source against a fresh evaluator and returns what it printed.
func evalNative(t *testing.T, source string) string {
	t.Helper()

	var buf bytes.Buffer
	if err := evalToBuf(t, &buf, source); err != nil {
		t.Fatalf("eval(%q): unexpected error: %v", source, err)
	}
	return buf.String()
}

// clock is defined in the global scope, so it is found like any other global.
func TestClockIsDefinedInTheGlobalScope(t *testing.T) {
	e := NewEvaluator()

	got, err := e.environment.Get(tokenForTest("clock"))
	if err != nil {
		t.Fatalf("looking up clock: %v", err)
	}
	if _, ok := got.(*NativeFunction); !ok {
		t.Errorf("clock = %v (%T), want a *NativeFunction", got, got)
	}
}

// Calling clock returns a number: the seconds since the epoch.
func TestClockReturnsANumber(t *testing.T) {
	e := evalTo(t, `let t = clock();`)

	got, err := e.environment.Get(tokenForTest("t"))
	if err != nil {
		t.Fatalf("looking up t: %v", err)
	}
	seconds, ok := got.(float64)
	if !ok {
		t.Fatalf("clock() = %v (%T), want a float64", got, got)
	}
	if seconds < 1_000_000_000 {
		t.Errorf("clock() = %v, want a plausible epoch time", seconds)
	}
}

// Two calls measure the time between them.
func TestClockMeasuresElapsedTime(t *testing.T) {
	e := evalTo(t, `let first = clock(); let second = clock();`)

	first, err := e.environment.Get(tokenForTest("first"))
	if err != nil {
		t.Fatalf("looking up first: %v", err)
	}
	second, err := e.environment.Get(tokenForTest("second"))
	if err != nil {
		t.Fatalf("looking up second: %v", err)
	}

	elapsed := second.(float64) - first.(float64)
	if elapsed < 0 {
		t.Errorf("the second clock() = %v is before the first (%v)", second, first)
	}
	if elapsed > 1 {
		t.Errorf("the two clock() calls are %v seconds apart, want under one", elapsed)
	}
}

// A native function is reached from inside a function body, where the lookup
// walks the environment chain out to the global scope.
func TestClockIsReachableFromInsideAFunction(t *testing.T) {
	source := `fun now() { return clock(); } let t = now();`

	e := evalTo(t, source)
	got, err := e.environment.Get(tokenForTest("t"))
	if err != nil {
		t.Fatalf("looking up t: %v", err)
	}
	if _, ok := got.(float64); !ok {
		t.Errorf("now() = %v (%T), want a float64", got, got)
	}
}

// A native function is a value like any other, so it can be passed around.
func TestClockCanBePassedAsAValue(t *testing.T) {
	source := `fun call(f) { return f(); } let t = call(clock);`

	e := evalTo(t, source)
	got, err := e.environment.Get(tokenForTest("t"))
	if err != nil {
		t.Fatalf("looking up t: %v", err)
	}
	if _, ok := got.(float64); !ok {
		t.Errorf("call(clock) = %v (%T), want a float64", got, got)
	}
}

// The arity check applies to native functions too.
func TestClockTakesNoArguments(t *testing.T) {
	err := runProgram(t, `clock(1);`)
	if err == nil {
		t.Fatal("want an error passing an argument to clock()")
	}
	if !strings.Contains(err.Error(), "Expected 0 arguments but got 1.") {
		t.Errorf("error = %q, want the arity error", err)
	}
}

// Printing a native function shows that it is implemented in Go.
func TestPrintNativeFunction(t *testing.T) {
	if got := evalNative(t, `print clock;`); got != "<native fn>\n" {
		t.Errorf("printed %q, want %q", got, "<native fn>\n")
	}
}

// A native function shares the namespace with variables: a binding of the
// same name shadows it.
func TestNativeFunctionCanBeShadowed(t *testing.T) {
	source := `let clock = "not a function"; print clock;`

	if got := evalNative(t, source); got != "not a function\n" {
		t.Errorf("printed %q, want %q", got, "not a function\n")
	}
}
