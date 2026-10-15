package parser

import (
	"strings"
	"testing"

	"magd/internal/ast"
	"magd/internal/errs"
	"magd/internal/lexer"
)

// parseStmt prints a whole statement, which is what if/while/for tests need.
func parseStmt(t *testing.T, source string) string {
	t.Helper()

	tokens := lexer.NewLexer(source).ScanTokens()
	p := NewParser(tokens)
	stmts, _ := p.Parse()
	if errs := p.Errors(); len(errs) != 0 {
		t.Fatalf("parse(%q): unexpected errors: %v", source, errs)
	}

	printed, err := (&ast.Printer{}).Print(stmts)
	if err != nil {
		t.Fatalf("Print() error = %v", err)
	}

	return strings.TrimSuffix(printed, "\n")
}

func TestLogicalOperators(t *testing.T) {
	tests := []struct {
		source string
		want   string
	}{
		{"a or b", "(or a b)"},
		{"a and b", "(and a b)"},

		// and binds tighter than or.
		{"a or b and c", "(or a (and b c))"},
		{"a and b or c", "(or (and a b) c)"},

		// Both are left-associative.
		{"a or b or c", "(or (or a b) c)"},
		{"a and b and c", "(and (and a b) c)"},
		{"a or b or c or d", "(or (or (or a b) c) d)"},

		// or is looser than equality, so the comparison stays nested inside.
		{"a == b or c", "(or (== a b) c)"},
		{"a and b == c", "(and a (== b c))"},

		// ? : binds loosest of all: it wraps the whole logical expression.
		{"a or b ? 1 : 2", "(? (or a b) 1 2)"},
		{"a and b ? 1 : 2", "(? (and a b) 1 2)"},
		{"a ? b : c or d", "(? a b (or c d))"},

		// Mixing both, with and tighter.
		{"a or b and c or d", "(or (or a (and b c)) d)"},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			if got := parseStr(t, tt.source); got != tt.want {
				t.Errorf("parse(%q) = %q, want %q", tt.source, got, tt.want)
			}
		})
	}
}

func TestIfStatement(t *testing.T) {
	tests := []struct {
		source string
		want   string
	}{
		{"if (a) b;", "if (a) b;"},
		{"if (a) b; else c;", "if (a) b; else c;"},
		{"if (!a) b;", "if ((! a)) b;"},

		// else if chains: each else holds another if statement.
		{"if (a) b; else if (c) d; else e;", "if (a) b; else if (c) d; else e;"},

		// Braces change nothing about the shape.
		{"if (a) { print 1; }", "if (a) {\nprint 1;\n}"},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			if got := parseStmt(t, tt.source); got != tt.want {
				t.Errorf("parse(%q) = %q, want %q", tt.source, got, tt.want)
			}
		})
	}
}

// The dangling else binds to the nearest if: the inner one, so the inner if
// is complete and the outer has no else.
//
// Call syntax is deliberately absent: calls are a chapter 10 feature, and
// "c()" does not parse yet.
func TestDanglingElseBindsToNearestIf(t *testing.T) {
	got := parseStmt(t, "if (first) if (second) whenTrue; else whenFalse;")

	if want := "if (first) if (second) whenTrue; else whenFalse;"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestWhileStatement(t *testing.T) {
	tests := []struct {
		source string
		want   string
	}{
		{"while (a) b;", "while (a) b;"},
		{"while (a) { print 1; }", "while (a) {\nprint 1;\n}"},
		// The condition is parenthesised by the printer, so a comparison
		// arrives already wrapped: while ((< i 10)).
		{"while (i < 10) i = i + 1;", "while ((< i 10)) (= i (+ i 1));"},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			if got := parseStmt(t, tt.source); got != tt.want {
				t.Errorf("parse(%q) = %q, want %q", tt.source, got, tt.want)
			}
		})
	}
}

func TestForStatement(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		// For is a dedicated node rather than a desugared while, so each clause is
		// printed in its own slot. An omitted clause prints as nothing.
		{"all clauses", "for (let i = 0; i < 3; i = i + 1) print i;",
			"for (let i = 0; (< i 3); (= i (+ i 1))) print i;"},
		{"empty initializer", "for (; i < 3; i = i + 1) print i;",
			"for (; (< i 3); (= i (+ i 1))) print i;"},
		// An omitted condition means "forever", so the parser substitutes the
		// literal true and the node's Condition is never nil.
		{"empty condition", "for (let i = 0;;) print i;",
			"for (let i = 0; true; ) print i;"},
		{"empty increment", "for (let i = 0; i < 3;) print i;",
			"for (let i = 0; (< i 3); ) print i;"},
		{"all clauses empty", "for (;;) print i;",
			"for (; true; ) print i;"},
		{"expression initializer", "for (i = 0; i < 3; i = i + 1) print i;",
			"for ((= i 0); (< i 3); (= i (+ i 1))) print i;"},
		{"block body", "for (let i = 0; i < 2; i = i + 1) { print i; }",
			"for (let i = 0; (< i 2); (= i (+ i 1))) {\nprint i;\n}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseStmt(t, tt.source); got != tt.want {
				t.Errorf("parse(%q)\n got: %q\nwant: %q", tt.source, got, tt.want)
			}
		})
	}
}

// A missing semicolon between clauses must be an error, not a silent shift of
// the remaining clauses.
func TestForMissingSemicolonIsAnError(t *testing.T) {
	tokens := lexer.NewLexer("for (let i = 0 i < 3;) print i;").ScanTokens()
	p := NewParser(tokens)
	p.Parse()

	if len(p.Errors()) == 0 {
		t.Fatal("want a syntax error for the missing ';' after the initializer")
	}
}

// parseErrs returns the diagnostics for source, without failing the test.
func parseErrs(t *testing.T, source string) []errs.SyntaxError {
	t.Helper()

	p := NewParser(lexer.NewLexer(source).ScanTokens())
	p.Parse()
	return p.Errors()
}

// break and continue are only legal inside a loop. The depth counter covers
// the loop body alone, so a for's initializer, condition and increment do not
// count as loop context.
func TestBreakAndContinueAreRejectedOutsideALoop(t *testing.T) {
	for _, keyword := range []string{"break", "continue"} {
		t.Run(keyword+" at top level", func(t *testing.T) {
			if got := parseErrs(t, keyword+";"); len(got) == 0 {
				t.Errorf("parse(%q) = no errors, want one for '%s' outside a loop", keyword+";", keyword)
			}
		})

		t.Run(keyword+" in a block", func(t *testing.T) {
			if got := parseErrs(t, "{ "+keyword+"; }"); len(got) == 0 {
				t.Errorf("parse(%q) = no errors, want one", "{ "+keyword+"; }")
			}
		})

		t.Run(keyword+" in an if", func(t *testing.T) {
			if got := parseErrs(t, "if (true) "+keyword+";"); len(got) == 0 {
				t.Errorf("parse(%q) = no errors, want one", "if (true) "+keyword+";")
			}
		})

		t.Run(keyword+" after a loop", func(t *testing.T) {
			if got := parseErrs(t, "while (true) { } "+keyword+";"); len(got) == 0 {
				t.Errorf("parse(%q) = no errors, want one", "while (true) { } "+keyword+";")
			}
		})

		t.Run(keyword+" in a for increment", func(t *testing.T) {
			source := "for (let i = 0; i < 3; " + keyword + ") print i;"
			if got := parseErrs(t, source); len(got) == 0 {
				t.Errorf("parse(%q) = no errors, want one", source)
			}
		})
	}
}

func TestBreakAndContinueAreAcceptedInsideALoop(t *testing.T) {
	sources := []string{
		"while (true) break;",
		"while (true) continue;",
		"for (;;) break;",
		"for (;;) continue;",
		"while (true) { break; }",
		"while (true) { continue; }",
		"while (true) if (true) break;",
		"while (true) if (false) continue;",
		"for (let i = 0; i < 3; i = i + 1) { if (i == 1) break; }",
		"while (true) { while (true) { break; } continue; }",
	}

	for _, source := range sources {
		t.Run(source, func(t *testing.T) {
			if got := parseErrs(t, source); len(got) != 0 {
				t.Errorf("parse(%q) = %v, want no errors", source, got)
			}
		})
	}
}

// A syntax error inside a loop body unwinds through the panic that carries the
// error. The depth must still be restored, or every later break in the file is
// silently accepted.
func TestLoopDepthIsRestoredAfterASyntaxError(t *testing.T) {
	errs := parseErrs(t, "while (true) { 1 + } break;")

	// One error for the malformed body, one for the top-level break.
	if len(errs) != 2 {
		t.Errorf("got %d errors (%v), want 2: a stale loop depth swallows the break",
			len(errs), errs)
	}
}
