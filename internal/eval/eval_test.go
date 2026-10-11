package eval

import (
	"bytes"
	"io"
	"math"
	"os"
	"strings"
	"testing"

	"magd/internal/ast"
	"magd/internal/lexer"
	"magd/internal/parser"
	"magd/internal/token"
)

// tokenForTest builds a token with a lexeme, for tests of the operand checks.
func tokenForTest(lexeme string) token.Token {
	return token.Token{Type: token.MINUS, Lexeme: lexeme}
}

// evalSource runs the whole pipeline: lex, parse, evaluate. It fails the test
// if either earlier stage reports an error, since these tests are only about
// evaluation.
//
// Eval executes a program of statements and yields no value, but most of these
// tests are about a single expression's value. So this unwraps the one
// ExpressionStmt the parser produced and evaluates its expression directly.
// Statement-level behaviour is covered separately, via Eval.
func evalSource(t *testing.T, source string) any {
	t.Helper()

	expr := parseOneExpression(t, source)

	value, err := (&Evaluator{}).evaluate(expr)
	if err != nil {
		t.Fatalf("eval(%q): unexpected error: %v", source, err)
	}

	return value
}

// evalError parses and evaluates source expecting a runtime error, returning it.
func evalError(t *testing.T, source string) error {
	t.Helper()

	expr := parseOneExpression(t, source)

	_, err := (&Evaluator{}).evaluate(expr)
	if err == nil {
		t.Fatalf("eval(%q): want a runtime error, got none", source)
	}

	return err
}

// parseOneExpression lexes and parses a bare expression into a single
// ExpressionStmt, then returns that statement's expression.
//
// The parser requires statements, so the semicolon is appended here. Tests
// below are about expression values, not statement syntax, so they spell their
// input as plain expressions and let this helper terminate it.
func parseOneExpression(t *testing.T, source string) ast.Expression {
	t.Helper()

	tokens := lexer.NewLexer(source + ";").ScanTokens()
	p := parser.NewParser(tokens)

	stmts, _ := p.Parse()
	if errs := p.Errors(); len(errs) != 0 {
		t.Fatalf("parse(%q): unexpected errors: %v", source, errs)
	}

	if len(stmts) != 1 {
		t.Fatalf("parse(%q): want 1 statement, got %d", source, len(stmts))
	}

	stmt, ok := stmts[0].(*ast.ExpressionStmt)
	if !ok {
		t.Fatalf("parse(%q): want *ast.ExpressionStmt, got %T", source, stmts[0])
	}

	return stmt.Expr
}

func TestLiterals(t *testing.T) {
	tests := []struct {
		source string
		want   any
	}{
		{"123", float64(123)},
		{"12.34", 12.34},
		{"0", float64(0)},
		{"-0", math.Copysign(0, -1)},
		{`"hello"`, "hello"},
		{`""`, ""},
		{"true", true},
		{"false", false},
		{"nil", nil},
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

// oneFifth is computed at run time so tests see the float64 result rather than
// a constant-folded one. Go folds 0.1 + 0.2 written as literals, which would
// hide exactly the rounding behaviour these tests check.
var oneFifth = 1.0 / 5.0

func TestArithmetic(t *testing.T) {
	tests := []struct {
		source string
		want   any
	}{
		{"1 + 2", float64(3)},
		{"2 + 2", float64(4)},
		{"1 - 2", float64(-1)},
		{"5 - 3", float64(2)},
		{"3 * 4", float64(12)},
		{"7 / 2", 3.5},
		{"1 + 2 * 3", float64(7)},
		{"2 * 3 + 4 * 5", float64(26)},
		{"(1 + 2) * 3", float64(9)},
		{"10 - 2 - 3", float64(5)},   // left-associative
		{"100 / 10 / 2", float64(5)}, // left-associative
		{"-(1 + 2)", float64(-3)},
		{"-3 + 2", float64(-1)},
		// Float rounding is visible: Lox numbers are doubles.
		{"0.1 + 0.2", 0.1 + oneFifth},
		{"1 + 2 == 3", true},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			got := evalSource(t, tt.source)
			if got != tt.want {
				t.Errorf("eval(%q) = %v (%T), want %v (%T)",
					tt.source, got, got, tt.want, tt.want)
			}
		})
	}
}

func TestStringConcatenation(t *testing.T) {
	tests := []struct {
		source string
		want   string
	}{
		{`"a" + "b"`, "ab"},
		{`"" + ""`, ""},
		{`"hello" + " " + "world"`, "hello world"},
		{`"a" + ("b" + "c")`, "abc"},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			if got := evalSource(t, tt.source); got != tt.want {
				t.Errorf("eval(%q) = %v, want %v", tt.source, got, tt.want)
			}
		})
	}
}

func TestComparison(t *testing.T) {
	tests := []struct {
		source string
		want   bool
	}{
		{"1 < 2", true},
		{"2 < 1", false},
		{"1 < 1", false},
		{"2 > 1", true},
		{"1 > 2", false},
		{"1 <= 1", true},
		{"1 <= 0", false},
		{"1 >= 1", true},
		{"1 >= 2", false},
		{"1 < 2 == true", true},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			if got := evalSource(t, tt.source); got != tt.want {
				t.Errorf("eval(%q) = %v, want %v", tt.source, got, tt.want)
			}
		})
	}
}

func TestEquality(t *testing.T) {
	tests := []struct {
		source string
		want   bool
	}{
		// Same type, same value.
		{"1 == 1", true},
		{"1 == 2", false},
		{`"a" == "a"`, true},
		{`"a" == "b"`, false},
		{"true == true", true},
		{"nil == nil", true},
		{"1 != 2", true},
		{"1 != 1", false},

		// No integer type in Lox, so 1 and 1.0 are the same value.
		{"1 == 1.0", true},

		// Mixed types are never equal. No implicit conversions.
		{"nil == false", false},
		{"nil == 0", false},
		{"nil == 1", false},
		{"true == 1", false},
		{`1 == "1"`, false},
		{`"1" == 1`, false},

		// Comparing a value with itself.
		{"(1 + 1) == 2", true},
		{`("a" + "b") == "ab"`, true},

		// nil from a conditional still compares as nil.
		{"nil == (true ? nil : 1)", true},
		{"nil == (false ? nil : 1)", false}, // the branch taken is 1
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			if got := evalSource(t, tt.source); got != tt.want {
				t.Errorf("eval(%q) = %v, want %v", tt.source, got, tt.want)
			}
		})
	}
}

func TestUnary(t *testing.T) {
	tests := []struct {
		source string
		want   any
	}{
		{"-1", float64(-1)},
		{"-1.5", -1.5},
		{"--1", float64(1)},
		{"- - 1", float64(1)}, // whitespace between them
		{"-(3)", float64(-3)},
		{"!true", false},
		{"!false", true},
		{"!!true", true},

		// Ruby-style truthiness: only false and nil are falsey.
		{"!nil", true},
		{"!0", false},
		{`!""`, false},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			if got := evalSource(t, tt.source); got != tt.want {
				t.Errorf("eval(%q) = %v, want %v", tt.source, got, tt.want)
			}
		})
	}
}

func TestConditional(t *testing.T) {
	tests := []struct {
		source string
		want   any
	}{
		{"true ? 1 : 2", float64(1)},
		{"false ? 1 : 2", float64(2)},
		{`true ? "a" : "b"`, "a"},

		// Truthiness decides the branch: only false and nil are falsey.
		{"nil ? 1 : 2", float64(2)},
		{"0 ? 1 : 2", float64(1)},
		{`"" ? 1 : 2`, float64(1)},
		{`"a" ? 1 : 2`, float64(1)},

		// The condition can be any expression.
		{"1 < 2 ? 1 : 2", float64(1)},
		{"1 == 2 ? 1 : 2", float64(2)},
		{"(1 + 1) == 2 ? \"y\" : \"n\"", "y"},

		// Nesting and associativity.
		{"true ? 1 : false ? 2 : 3", float64(1)},
		{"false ? 1 : false ? 2 : 3", float64(3)},
		{"false ? 1 : true ? 2 : 3", float64(2)},
		{"(true ? false : true) ? 1 : 2", float64(2)},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			if got := evalSource(t, tt.source); got != tt.want {
				t.Errorf("eval(%q) = %v, want %v", tt.source, got, tt.want)
			}
		})
	}
}

// Only the chosen branch is evaluated, so a runtime error in the untaken branch
// must not surface.
func TestConditionalOnlyEvaluatesChosenBranch(t *testing.T) {
	if got := evalSource(t, `false ? "a" + 1 : "ok"`); got != "ok" {
		t.Errorf("want \"ok\", got %v", got)
	}
	if got := evalSource(t, `true ? "ok" : 1 + "a"`); got != "ok" {
		t.Errorf("want \"ok\", got %v", got)
	}
}

func TestRuntimeErrors(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		message string
		lexeme  string
	}{
		{"negate a string", `-"a"`, "Operand must be a number.", "-"},
		{"negate a boolean", "-true", "Operand must be a number.", "-"},
		{"negate nil", "-nil", "Operand must be a number.", "-"},

		{"string times number", `"a" * 2`, "Operand must be a number.", "*"},
		{"divide string by number", `"a" / 2`, "Operand must be a number.", "/"},
		{"subtract string from number", `1 - "a"`, "Operand must be a number.", "-"},

		{"compare string with number", `1 < "b"`, "Operand must be a number.", "<"},
		{"compare two strings", `"a" < "b"`, "Operand must be a number.", "<"},
		{"greater than nil", `1 > nil`, "Operand must be a number.", ">"},

		// Chained comparison: the inner one yields a bool, which the outer
		// operator then rejects. Lox behaves the same way.
		{"chained comparison", `1 < 2 < 3`, "Operand must be a number.", "<"},
		{"chained greater than", `3 > 2 > 1`, "Operand must be a number.", ">"},

		{"number plus string", `1 + "a"`, "Operands must be two numbers or two strings.", "+"},
		{"string plus number", `"a" + 1`, "Operands must be two numbers or two strings.", "+"},
		{"string plus boolean", `"a" + true`, "Operands must be two numbers or two strings.", "+"},
		{"nil plus nil", `nil + nil`, "Operands must be two numbers or two strings.", "+"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := evalError(t, tt.source)

			if !strings.Contains(err.Error(), tt.message) {
				t.Errorf("eval(%q) error = %q, want it to contain %q",
					tt.source, err, tt.message)
			}
			if !strings.Contains(err.Error(), "at '"+tt.lexeme+"'") {
				t.Errorf("eval(%q) error = %q, want it to name the %q operator",
					tt.source, err, tt.lexeme)
			}
			if !strings.Contains(err.Error(), "runtime error") {
				t.Errorf("eval(%q) error = %q, want it labelled a runtime error",
					tt.source, err)
			}
		})
	}
}

// An error deep in the tree must surface with the operator where it happened,
// not where evaluation started.
func TestRuntimeErrorReportsDeepestOperator(t *testing.T) {
	err := evalError(t, `2 * (3 / -"muffin")`)

	want := "at '-'"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("error = %q, want it to report the innermost %s, not the outer operators",
			err, want)
	}
	if strings.Contains(err.Error(), "at '*'") {
		t.Errorf("error = %q, should not blame the outer '*' operator", err)
	}
}

// Once an error is recovered from, the evaluator must still work.
func TestEvalRecoversAndStaysUsable(t *testing.T) {
	e := NewEvaluator()

	stmts, _ := parser.NewParser(lexer.NewLexer(`-"a";`).ScanTokens()).Parse()
	if err := e.Eval(stmts); err == nil {
		t.Fatal("want an error from the first evaluation")
	}

	// Same evaluator, second program — this is the REPL case.
	stmts, _ = parser.NewParser(lexer.NewLexer("1 + 2;").ScanTokens()).Parse()
	if err := e.Eval(stmts); err != nil {
		t.Fatalf("second Eval() error = %v, want it to succeed", err)
	}

	// Eval yields no value, so check the result via the expression itself.
	value, err := e.evaluate(parseOneExpression(t, "1 + 2"))
	if err != nil {
		t.Fatalf("evaluate() error = %v", err)
	}
	if value != float64(3) {
		t.Errorf("second evaluation = %v, want 3", value)
	}
}

// A runtime error in one statement must prevent later statements from running:
// Eval recovers at the loop, so execution stops rather than continuing.
func TestEvalStopsAtFirstError(t *testing.T) {
	e := NewEvaluator()

	// The first statement fails; the second would print "boom" if it ran.
	source := `1 + (2 / 0); print "boom";`
	p := parser.NewParser(lexer.NewLexer(source).ScanTokens())
	stmts, _ := p.Parse()
	if errs := p.Errors(); len(errs) != 0 {
		t.Fatalf("parse(%q): unexpected errors: %v", source, errs)
	}

	var err error
	out := captureOutput(t, func() { err = e.Eval(stmts) })

	if err == nil {
		t.Fatal("want a runtime error from 2 / 0")
	}
	if !strings.Contains(err.Error(), "Cannot divide by zero") {
		t.Errorf("error = %q, want the division-by-zero error", err)
	}
	// The load-bearing assertion: Eval must stop, not skip the bad statement
	// and carry on. Only stdout proves the second statement never ran.
	if out != "" {
		t.Errorf("printed %q, want nothing: a statement after a runtime error must not execute", out)
	}
}

// Print statements write to stdout, so these go through evalPrint rather than
// the value-returning evalSource.
func TestPrintStmt(t *testing.T) {
	tests := []struct {
		source string
		want   string
	}{
		// The trailing \n is asserted: evalPrint does not trim, so a print
		// that lost its newline would fail here rather than pass silently.
		{`print 1;`, "1\n"},
		{`print 3;`, "3\n"}, // not "3.0" — Stringify's float formatting
		{`print "hi";`, "hi\n"},
		{`print "";`, "\n"}, // an empty string still prints a line
		{`print true;`, "true\n"},
		{`print nil;`, "nil\n"},
		{`print 10 / 4;`, "2.5\n"},
		{`print -3;`, "-3\n"},

		// Order matters, so a VisitPrintStmt that buffered or reversed would
		// fail here.
		{`print 1; print 2;`, "1\n2\n"},
		{`print 1 + 2;`, "3\n"},
		{`print 1 == 1 ? "y" : "n";`, "y\n"},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			if got := evalPrint(t, tt.source); got != tt.want {
				t.Errorf("print %q\n got: %q\nwant: %q", tt.source, got, tt.want)
			}
		})
	}
}

// A failed print must not emit a partial line. VisitPrintStmt evaluates first,
// so a runtime error leaves stdout untouched.
func TestPrintStmtRuntimeErrorPrintsNothing(t *testing.T) {
	source := `print 1 / 0;`

	p := parser.NewParser(lexer.NewLexer(source).ScanTokens())
	stmts, _ := p.Parse()

	var err error
	out := captureOutput(t, func() { err = NewEvaluator().Eval(stmts) })

	if err == nil {
		t.Fatal("want a runtime error from 1 / 0")
	}
	if out != "" {
		t.Errorf("printed %q, want nothing", out)
	}
}

// captureOutput runs fn with os.Stdout redirected to a pipe and returns
// everything written to it.
//
// VisitPrintStmt prints straight to os.Stdout via fmt.Println, so there is no
// other way to observe what a print statement produced. The alternative —
// injecting an io.Writer into Evaluator — would change production code to suit
// tests; redirection confines the workaround to the test.
func captureOutput(t *testing.T, fn func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}

	// Drain concurrently: a pipe holds a limited buffer before it blocks, so
	// printing more than that without a reader would deadlock.
	collected := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		collected <- buf.String()
	}()

	orig := os.Stdout
	os.Stdout = w
	defer func() {
		// Restore even if fn fails the test, so one failure can't redirect
		// stdout for the rest of the run.
		os.Stdout = orig
		_ = r.Close()
	}()

	fn()

	// The reader only sees EOF once the writer is closed, so this must happen
	// after fn returns and before waiting on the goroutine.
	os.Stdout = orig
	_ = w.Close()

	return <-collected
}

// evalPrint runs source as a program and returns what it printed to stdout.
func evalPrint(t *testing.T, source string) string {
	t.Helper()

	p := parser.NewParser(lexer.NewLexer(source).ScanTokens())
	stmts, _ := p.Parse()
	if errs := p.Errors(); len(errs) != 0 {
		t.Fatalf("parse(%q): unexpected errors: %v", source, errs)
	}

	return captureOutput(t, func() {
		// NewEvaluator, not &Evaluator{}: the zero value has a nil values map,
		// so any `let` in the program panics on assignment to a nil map.
		if err := NewEvaluator().Eval(stmts); err != nil {
			t.Fatalf("eval(%q): unexpected error: %v", source, err)
		}
	})
}

func TestStringify(t *testing.T) {
	sum := 0.1 + oneFifth

	tests := []struct {
		value any
		want  string
	}{
		{nil, "nil"},
		{float64(123), "123"}, // not "123.0"
		{float64(-1), "-1"},
		{float64(0), "0"},
		{float64(3.5), "3.5"},
		{float64(12.34), "12.34"},
		{sum, "0.30000000000000004"},
		{"hello", "hello"},
		{"", ""},
		{"multi\nline", "multi\nline"},
		{true, "true"},
		{false, "false"},
		// Lox has no scientific notation for ordinary magnitudes.
		{1e21, "1000000000000000000000"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := Stringify(tt.value); got != tt.want {
				t.Errorf("Stringify(%v) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

// Challenge 3: division by zero is a runtime error, not IEEE 754 infinity.
// Go would silently produce +Inf, -Inf or NaN where Java throws.
func TestDivisionByZero(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{"integer zero", "1 / 0"},
		{"float zero", "1 / 0.0"},
		{"negative zero", "1 / -0.0"},
		{"negative dividend", "-1 / 0"},
		{"zero dividend", "0 / 0"},
		{"computed zero divisor", "1 / (2 - 2)"},
		{"zero as a condition result", "4 / (1 == 2 ? 1 : 0)"},
		{"nested in a larger expression", "2 * (3 / (1 - 1))"},
		{"left operand is fine", "(1 / 0) == 1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := evalError(t, tt.source)

			if !strings.Contains(err.Error(), "Cannot divide by zero.") {
				t.Errorf("eval(%q) error = %q, want the divide-by-zero message",
					tt.source, err)
			}
			if !strings.Contains(err.Error(), "at '/'") {
				t.Errorf("eval(%q) error = %q, want it to blame the '/' operator",
					tt.source, err)
			}
		})
	}
}

// A zero divisor must be rejected whatever the dividend, and a nonzero one
// must still work.
func TestDivisionByNonZeroStillWorks(t *testing.T) {
	tests := []struct {
		source string
		want   float64
	}{
		{"1 / 2", 0.5},
		{"6 / 3", 2},
		{"1 / -2", -0.5},
		{"10 / 0.5", 20},
		{"0 / 5", 0},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			got := evalSource(t, tt.source)
			if got != tt.want {
				t.Errorf("eval(%q) = %v, want %v", tt.source, got, tt.want)
			}
		})
	}
}

func TestIsTruthy(t *testing.T) {
	tests := []struct {
		value any
		want  bool
	}{
		{nil, false},
		{false, false},
		{true, true},
		{float64(0), true},
		{float64(1), true},
		{"", true},
		{"a", true},
	}

	e := &Evaluator{}
	for _, tt := range tests {
		if got := e.isTruthy(tt.value); got != tt.want {
			t.Errorf("isTruthy(%v) = %v, want %v", tt.value, got, tt.want)
		}
	}
}

func TestIsEqual(t *testing.T) {
	tests := []struct {
		left, right any
		want        bool
	}{
		{nil, nil, true},
		{nil, false, false},
		{nil, float64(0), false},
		{false, nil, false},
		{false, false, true},
		{true, false, false},
		{true, true, true},
		{float64(1), float64(1), true},
		{float64(1), float64(1.0), true},
		{float64(1), float64(2), false},
		{"a", "a", true},
		{"a", "b", false},
		{float64(1), "1", false},

		// DeepEqual, not ==: uncomparable types must not panic.
		{[]float64{1, 2}, []float64{1, 2}, true},
		{[]float64{1, 2}, []float64{1, 3}, false},
	}

	e := &Evaluator{}
	for _, tt := range tests {
		if got := e.isEqual(tt.left, tt.right); got != tt.want {
			t.Errorf("isEqual(%v, %v) = %v, want %v", tt.left, tt.right, got, tt.want)
		}
	}
}

func TestCheckNumberOperand(t *testing.T) {
	op := tokenForTest("-")

	if err := checkNumberOperand(op, float64(1)); err != nil {
		t.Errorf("checkNumberOperand(_, 1) = %v, want nil", err)
	}

	for _, operand := range []any{nil, "a", true} {
		err := checkNumberOperand(op, operand)
		if err == nil {
			t.Errorf("checkNumberOperand(_, %v) = nil, want an error", operand)
			continue
		}
		if !strings.Contains(err.Error(), "Operand must be a number.") {
			t.Errorf("checkNumberOperand(_, %v) = %q, want the number message", operand, err)
		}
	}
}

func TestCheckNumberOperands(t *testing.T) {
	op := tokenForTest("*")

	if err := checkNumberOperands(op, float64(1), float64(2)); err != nil {
		t.Errorf("checkNumberOperands(_, 1, 2) = %v, want nil", err)
	}

	if err := checkNumberOperands(op, float64(1), "a"); err == nil {
		t.Error("checkNumberOperands(_, 1, \"a\") = nil, want an error")
	}
	if err := checkNumberOperands(op, "a", float64(2)); err == nil {
		t.Error("checkNumberOperands(_, \"a\", 2) = nil, want an error")
	}
	if err := checkNumberOperands(op, nil, nil); err == nil {
		t.Error("checkNumberOperands(_, nil, nil) = nil, want an error")
	}
}
