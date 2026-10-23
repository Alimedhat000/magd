package resolver_test

// Chapter 12 and 13 coverage: this, super, and inheritance diagnostics.
//
// The resolver is where "this is only legal in a class" and "super needs a
// superclass" are decided, so these tests run resolution explicitly and then
// evaluate, which also proves the recorded distances still line up at run
// time.

import (
	"strings"
	"testing"
)

// resolveError resolves source expecting an error containing want.
func resolveError(t *testing.T, source, want string) {
	t.Helper()

	_, err := run(t, source)
	if err == nil {
		t.Fatalf("eval(%q): want an error, got none", source)
	}
	if !strings.Contains(err.Error(), want) {
		t.Errorf("eval(%q) error = %q, want one containing %q", source, err, want)
	}
}

func TestThisOutsideClassIsRejected(t *testing.T) {
	resolveError(t, `this;`, "Can't use 'this' outside of a class.")
}

func TestSuperOutsideClassIsRejected(t *testing.T) {
	resolveError(t, `super.m();`, "Can't use 'super' outside of a class.")
}

func TestSuperWithoutSuperclassIsRejected(t *testing.T) {
	resolveError(t,
		`class A { m() { super.m(); } }`,
		"Can't use 'super' in a class with no superclass.")
}

func TestClassCannotInheritFromItself(t *testing.T) {
	resolveError(t, `class A < A {}`, "A class cannot inherit from itself.")
}

func TestReturnValueFromInitializerIsRejected(t *testing.T) {
	resolveError(t,
		`class A { init() { return 5; } }`,
		"Can't return a value from an initializer.")
}

// A bare return is fine, and so is a return with a value inside a nested
// function: only the initializer itself is restricted.
func TestInitializerReturnRules(t *testing.T) {
	evalProgram(t, `class A { init() { return; } } let a = A();`)

	evalProgram(t, `
class A {
	init() {
		let f = fun() { return 5; };
		f();
	}
}
let a = A();`)
}

// "super" is added to the class scope like "this" is, so the unused-variable
// check must not report it when a subclass never calls super.
func TestUnusedSuperIsNotReported(t *testing.T) {
	evalProgram(t, `class A {} class B < A {}`)

	evalProgram(t, `
class A {}
class B < A {
	m() { return 1; }
}
let b = B();`)
}

// An inner class starts fresh: a "super" inside it belongs to that class, not
// to the subclass that encloses it.
func TestNestedClassDoesNotInheritSuperState(t *testing.T) {
	resolveError(t, `
class A { m() {} }
class B < A {
	m() {
		class C {
			n() { super.m(); }
		}
	}
}`,
		"Can't use 'super' in a class with no superclass.")
}

// The distances for both "this" and "super" survive a nested function: the
// closure still reaches the instance and its superclass.
func TestThisAndSuperResolveThroughNestedFunction(t *testing.T) {
	evalPrints(t, `
class A { m() { return this.tag; } }
class B < A {
	init() { this.tag = "B"; }
	m() {
		let f = fun() { return super.m(); };
		print f();
	}
}
B().m();
`, "B\n")
}
