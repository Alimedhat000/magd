package parser

import (
	"testing"

	"magd/internal/ast"
	"magd/internal/lexer"
	"magd/internal/token"
)

func TestTernary(t *testing.T) {
	tests := []struct {
		source string
		want   string
	}{
		// The basic shape.
		{"true ? 1 : 2", "(? true 1 2)"},
		{"nil ? 1 : 2", "(? nil 1 2)"},
		{`"s" ? 1 : 2`, "(? s 1 2)"},
		{"1 ? 2 : 3", "(? 1 2 3)"},

		// Any expression can be the condition.
		{"!true ? 1 : 2", "(? (! true) 1 2)"},
		{"1 + 2 ? 3 : 4", "(? (+ 1 2) 3 4)"},
		{"1 < 2 ? 3 : 4", "(? (< 1 2) 3 4)"},
		{"1 == 2 ? 3 : 4", "(? (== 1 2) 3 4)"},
		{"(1 == 2) ? 3 : 4", "(? (group (== 1 2)) 3 4)"},

		// The whole point of ?: a comparison is the condition. The ? must
		// attach to everything on its left, not to just the last operand.
		{"1 == 2 == 3 ? 4 : 5", "(? (== (== 1 2) 3) 4 5)"},
		{"1 < 2 == 3 ? 4 : 5", "(? (== (< 1 2) 3) 4 5)"},
		{"1 == 2 < 3 ? 4 : 5", "(? (== 1 (< 2 3)) 4 5)"},

		// Binary operators bind tighter than ?: they stay inside the arms.
		{"1 + 1 ? 2 * 3 : 4 - 5", "(? (+ 1 1) (* 2 3) (- 4 5))"},
		{"1 ? 2 == 3 : 4", "(? 1 (== 2 3) 4)"},
		{"1 ? 2 : 3 == 4", "(? 1 2 (== 3 4))"},
		{"1 ? 2 == 3 : 4 < 5", "(? 1 (== 2 3) (< 4 5))"},

		// Right-associative: the alternative nests to the right, so
		// a ? b : c ? d : e reads as a ? b : (c ? d : e).
		{"1 ? 2 : 3 ? 4 : 5", "(? 1 2 (? 3 4 5))"},
		{"1 ? 2 : 3 ? 4 : 5 ? 6 : 7", "(? 1 2 (? 3 4 (? 5 6 7)))"},

		// The consequent is a full expression, so it can hold a nested
		// ternary in parentheses.
		{"1 ? 2 ? 3 : 4 : 5", "(? 1 (? 2 3 4) 5)"},
		{"1 ? (2 ? 3 : 4) : 5", "(? 1 (group (? 2 3 4)) 5)"},

		// Deeply nested on both sides.
		{"1 ? 2 : 3 ? 4 : 5 ? 6 : 7", "(? 1 2 (? 3 4 (? 5 6 7)))"},
		{"1 == 2 ? 3 == 4 : 5 == 6", "(? (== 1 2) (== 3 4) (== 5 6))"},

		// Four levels, every one leaning right. If the alternative ever
		// parsed left-associatively this is the case that catches it.
		{"1 ? 2 : 3 ? 4 ? 5 : 6 : 7", "(? 1 2 (? 3 (? 4 5 6) 7))"},

		// A ternary nested in the alternative whose condition is itself a
		// comparison: right-associativity and precedence together.
		{"1 ? 2 : 3 == 4 ? 5 : 6", "(? 1 2 (? (== 3 4) 5 6))"},

		// A ternary inside grouping in the consequent.
		{"1 ? (2 ? 3 : 4) : 5", "(? 1 (group (? 2 3 4)) 5)"},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			tokens := lexer.NewLexer(tt.source).ScanTokens()
			p := NewParser(tokens)

			expr, _ := p.Parse()
			if errs := p.Errors(); len(errs) != 0 {
				t.Fatalf("parse(%q): unexpected errors: %v", tt.source, errs)
			}

			got, err := (&ast.Printer{}).Print(expr)
			if err != nil {
				t.Fatalf("Print() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("parse(%q)\nwant: %s\ngot:  %s", tt.source, tt.want, got)
			}
		})
	}
}

func TestTernaryErrors(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{"question mark with no consequent", "1 ? : 2"},
		{"question mark at end of input", "1 ?"},
		{"missing colon", "1 ? 2"},
		{"missing colon, alternative present", "1 ? 2 3"},
		{"colon with no question mark", "1 : 2"},
		{"colon at start of expression", ": 2"},
		{"only a question mark", "?"},
		{"only a colon", ":"},
		{"nested question mark with no consequent", "1 ? 2 : 3 ? : 4"},
		{"nested missing colon", "1 ? 2 : 3 ? 4"},

		// Repeated punctuation: one operator too many.
		{"double question mark", "1 ?? 2"},
		{"double colon", "1 :: 2"},
		{"colon in both arms", "1 ? 2 : 3 : 4"},
		{"question mark after colon", "1 ? 2 : 3 ? 4"},
		{"colon then question mark", "1 ? 2 : : 3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := lexer.NewLexer(tt.source).ScanTokens()
			p := NewParser(tokens)

			p.Parse()

			if errs := p.Errors(); len(errs) == 0 {
				t.Fatalf("parse(%q): want a syntax error, got none", tt.source)
			}
		})
	}
}

// Newlines are whitespace to the grammar, so a ternary may span lines. The
// line each token carries must still be right, which matters for errors once
// statements exist.
func TestTernaryAcrossLines(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
		line   int // line the error is expected on
	}{
		{
			name:   "condition on first line, error on second",
			source: "1 ? 2 : 3\n4",
			want:   "syntax error on line 1 at '4': Expect expression.",
			line:   1,
		},
		{
			name:   "newline after question mark",
			source: "1 ?\n2 : 3",
			want:   "(? 1 2 3)",
		},
		{
			name:   "newline before colon",
			source: "1 ? 2\n: 3",
			want:   "(? 1 2 3)",
		},
		{
			name:   "missing colon on later line",
			source: "1 ? 2\n3",
			want:   "syntax error on line 1 at '3': Expect ':' after '?'.",
			line:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := lexer.NewLexer(tt.source).ScanTokens()
			p := NewParser(tokens)

			expr, _ := p.Parse()
			errs := p.Errors()

			if tt.line == 0 && len(errs) == 0 {
				got, err := (&ast.Printer{}).Print(expr)
				if err != nil {
					t.Fatalf("Print() error = %v", err)
				}
				if got != tt.want {
					t.Errorf("parse(%q)\nwant: %s\ngot:  %s", tt.source, tt.want, got)
				}
				return
			}

			if len(errs) == 0 {
				t.Fatalf("parse(%q): want a syntax error, got none", tt.source)
			}
			if got := errs[0].Error(); got != tt.want {
				t.Errorf("parse(%q)\nwant: %s\ngot:  %s", tt.source, tt.want, got)
			}
		})
	}
}

func TestTernaryErrorMessages(t *testing.T) {
	tests := []struct {
		source string
		want   string
	}{
		{
			source: "1 ? 2",
			want:   "syntax error on line 0 at end: Expect ':' after '?'.",
		},
		{
			source: "1 ? : 2",
			want:   "syntax error on line 0 at ':': Missing ternary question mark.",
		},
		{
			source: "1 ?",
			want:   "syntax error on line 0 at end: Expect expression.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			tokens := lexer.NewLexer(tt.source).ScanTokens()
			p := NewParser(tokens)

			p.Parse()

			errs := p.Errors()
			if len(errs) == 0 {
				t.Fatalf("parse(%q): want a syntax error, got none", tt.source)
			}
			if got := errs[0].Error(); got != tt.want {
				t.Errorf("parse(%q)\nwant: %s\ngot:  %s", tt.source, tt.want, got)
			}
		})
	}
}

// ? and : must never be treated as binary operators, or the missing
// left-operand production would produce nonsense errors for them.
func TestTernaryTokensAreNotBinaryOperators(t *testing.T) {
	if token.BinaryOperators[token.QUESTION] {
		t.Error("QUESTION must not be in BinaryOperators")
	}
	if token.BinaryOperators[token.COLON] {
		t.Error("COLON must not be in BinaryOperators")
	}
}

// An expression with no ternary must still parse exactly as before, so
// adding ?: didn't disturb the existing precedence chain.
func TestTernaryDidNotChangeOtherPrecedence(t *testing.T) {
	tests := []struct {
		source string
		want   string
	}{
		{"1 + 2 * 3", "(+ 1 (* 2 3))"},
		{"1 - 2 - 3", "(- (- 1 2) 3)"},
		{"1 < 2 == true", "(== (< 1 2) true)"},
		{"1 == 2 < 3", "(== 1 (< 2 3))"},
		{"-1", "(- 1)"},
		{"!false", "(! false)"},
		{"(1 + 2) * 3", "(* (group (+ 1 2)) 3)"},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			tokens := lexer.NewLexer(tt.source).ScanTokens()
			p := NewParser(tokens)

			expr, _ := p.Parse()
			if errs := p.Errors(); len(errs) != 0 {
				t.Fatalf("parse(%q): unexpected errors: %v", tt.source, errs)
			}

			got, err := (&ast.Printer{}).Print(expr)
			if err != nil {
				t.Fatalf("Print() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("parse(%q)\nwant: %s\ngot:  %s", tt.source, tt.want, got)
			}
		})
	}
}
