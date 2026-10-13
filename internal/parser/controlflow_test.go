package parser

import (
	"strings"
	"testing"

	"magd/internal/ast"
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
		{"all clauses", "for (let i = 0; i < 3; i = i + 1) print i;",
			"{\nlet i = 0;\nwhile ((< i 3)) {\nprint i;\n(= i (+ i 1));\n}\n}"},
		{"empty initializer", "for (; i < 3; i = i + 1) print i;",
			"while ((< i 3)) {\nprint i;\n(= i (+ i 1));\n}"},
		// An empty condition means "forever", represented as the literal true.
		{"empty condition", "for (let i = 0;;) print i;",
			"{\nlet i = 0;\nwhile (true) {\nprint i;\n}\n}"},
		{"empty increment", "for (let i = 0; i < 3;) print i;",
			"{\nlet i = 0;\nwhile ((< i 3)) {\nprint i;\n}\n}"},
		{"expression initializer", "for (i = 0; i < 3; i = i + 1) print i;",
			"{\n(= i 0);\nwhile ((< i 3)) {\nprint i;\n(= i (+ i 1));\n}\n}"},
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
