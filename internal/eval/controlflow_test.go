package eval

import (
	"strings"
	"testing"
)

// evalPrints runs source as a program and returns what it printed.
func evalPrints(t *testing.T, source string) string {
	t.Helper()

	var runErr error
	out := captureOutput(t, func() { runErr = runProgram(t, source) })
	if runErr != nil {
		t.Fatalf("eval(%q): unexpected error: %v", source, runErr)
	}

	return out
}

func TestLogicalValues(t *testing.T) {
	tests := []struct {
		source string
		want   any
	}{
		{"true or false", true},
		{"false or true", true},
		{"nil or 1", float64(1)},
		{"false or nil", nil},
		// Only nil and false are falsey, so an empty string is truthy and
		// the left operand wins.
		{`"" or "x"`, ""},
		{"1 or 2", float64(1)},

		{"true and 1", float64(1)},
		{"false and 1", false},
		{"nil and 1", nil},
		{"1 and 2", float64(2)},
		{`"a" and "b"`, "b"},

		// and binds tighter than or.
		{"false or true and true", true},
		{"false and true or true", true},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			if got := evalSource(t, tt.source); got != tt.want {
				t.Errorf("eval(%q) = %v (%T), want %v (%T)",
					tt.source, got, got, tt.want, tt.want)
			}
		})
	}
}

// The right operand must not be evaluated when the left already decides the
// result. An undefined variable there would raise an error if it were.
func TestLogicalShortCircuits(t *testing.T) {
	tests := []struct {
		source string
		want   any
	}{
		{"true or missing", true},
		{"nil and missing", nil},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			if got := evalSource(t, tt.source); got != tt.want {
				t.Errorf("eval(%q) = %v (%T), want %v (%T): the right operand "+
					"must not be evaluated", tt.source, got, got, tt.want, tt.want)
			}
		})
	}
}

// An error in the left operand is a real error, not a silent falsey value.
func TestLogicalPropagatesLeftError(t *testing.T) {
	tests := []string{
		"missing or true",
		"missing and true",
	}

	for _, source := range tests {
		t.Run(source, func(t *testing.T) {
			expr := parseOneExpression(t, source)
			if _, err := NewEvaluator().EvalExpression(expr); err == nil {
				t.Errorf("eval(%q): want an undefined-variable error, got none", source)
			}
		})
	}
}

// The test helpers must report an undefined variable as an error rather than
// panicking on a nil environment, so that a logical expression reaching its
// right operand behaves like production.
func TestUndefinedVariableIsAnErrorNotAPanic(t *testing.T) {
	for _, source := range []string{"missing", "1 and missing", "nil or missing"} {
		t.Run(source, func(t *testing.T) {
			err := evalError(t, source)
			if !strings.Contains(err.Error(), "Undefined Variable") {
				t.Errorf("error = %q, want an undefined-variable error", err)
			}
		})
	}
}

func TestIfStatement(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{"then branch", `if (true) print "yes";`, "yes\n"},
		{"else branch", `if (false) print "yes"; else print "no";`, "no\n"},
		{"no else and false condition", `if (false) print "yes";`, ""},
		{"else if", `if (false) print "a"; else if (true) print "b"; else print "c";`, "b\n"},
		{"first else if matches", `if (true) print "a"; else if (true) print "b";`, "a\n"},
		{"neither matches", `if (false) print "a"; else if (false) print "b"; else print "c";`, "c\n"},
		{"nested", `if (true) if (true) print "x";`, "x\n"},
		{"non-boolean truthy", `if (1) print "t";`, "t\n"},
		{"nil is falsy", `if (nil) print "t"; else print "f";`, "f\n"},

		// The then-branch statement is not evaluated when the condition is
		// false; the else-branch not when true.
		{"untaken branches are skipped", `if (true) print "t"; else print missing;`, "t\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := evalPrints(t, tt.source); got != tt.want {
				t.Errorf("printed %q, want %q", got, tt.want)
			}
		})
	}
}

// An error inside a taken branch must reach the caller.
func TestIfPropagatesBranchError(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{"then branch", `if (true) print missing;`},
		{"else branch", `if (false) print missing; else print 1 / 0;`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := runProgram(t, tt.source); err == nil {
				t.Errorf("eval(%q): want an error from the branch, got none", tt.source)
			}
		})
	}
}

func TestWhileStatement(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{
			"counts up to the bound",
			`let i = 0; while (i < 3) { print i; i = i + 1; }`,
			"0\n1\n2\n",
		},
		{
			"body never runs",
			`while (false) print "never";`,
			"",
		},
		{
			"condition re-evaluated each iteration",
			`let i = 0; while (i < 2) { i = i + 1; } print i;`,
			"2\n",
		},
		{
			"nested loop",
			`for (let i = 0; i < 2; i = i + 1) { let j = 0; while (j < 2) { print i; j = j + 1; } }`,
			"0\n0\n1\n1\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := evalPrints(t, tt.source); got != tt.want {
				t.Errorf("printed %q, want %q", got, tt.want)
			}
		})
	}
}

// An error in the body must escape the loop instead of spinning on it.
func TestWhilePropagatesBodyError(t *testing.T) {
	if err := runProgram(t, `while (true) { print 1 / 0; }`); err == nil {
		t.Error("want a runtime error from the loop body, got none")
	}
}

func TestForStatement(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{
			"counts up",
			`for (let i = 0; i < 3; i = i + 1) print i;`,
			"0\n1\n2\n",
		},
		{
			"sums with an accumulator",
			`let sum = 0; for (let i = 1; i <= 4; i = i + 1) sum = sum + i; print sum;`,
			"10\n",
		},
		{
			"empty initializer reuses an outer variable",
			`let i = 0; for (; i < 3; ) { print i; i = i + 1; }`,
			"0\n1\n2\n",
		},
		{
			"expression initializer assigns an outer variable",
			`let i; for (i = 0; i < 2; i = i + 1) print i; print i;`,
			"0\n1\n2\n",
		},
		{
			"block body",
			`for (let i = 0; i < 2; i = i + 1) { print i; }`,
			"0\n1\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := evalPrints(t, tt.source); got != tt.want {
				t.Errorf("printed %q, want %q", got, tt.want)
			}
		})
	}
}

// The desugared block scopes the loop variable, so it is not visible after the
// loop, as in C.
func TestForLoopVariableIsScopedToTheLoop(t *testing.T) {
	evalPrints(t, `for (let i = 0; i < 1; i = i + 1) print i;`)

	if err := runProgram(t, `print i;`); err == nil {
		t.Error("want an undefined-variable error after the loop, got none")
	} else if !strings.Contains(err.Error(), "Undefined Variable") {
		t.Errorf("error = %q, want an undefined-variable error", err)
	}
}
