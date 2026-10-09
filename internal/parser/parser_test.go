package parser

import (
	"testing"

	"magd/internal/ast"
	"magd/internal/lexer"
)

// parse lexes and parses source, returning the printed AST. It fails the test
// if any syntax error is reported.
func parse(t *testing.T, source string) string {
	t.Helper()

	tokens := lexer.NewLexer(source).ScanTokens()
	p := NewParser(tokens)

	expr, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if errs := p.Errors(); len(errs) != 0 {
		t.Fatalf("parse(%q): unexpected errors: %v", source, errs)
	}

	printed, err := (&ast.Printer{}).Print(expr)
	if err != nil {
		t.Fatalf("Print() error = %v", err)
	}

	return printed
}

// parseErrors parses source expecting it to fail, returning the errors found.
func parseErrors(t *testing.T, source string) []error {
	t.Helper()

	tokens := lexer.NewLexer(source).ScanTokens()
	p := NewParser(tokens)

	p.Parse()

	found := p.Errors()
	if len(found) == 0 {
		t.Fatalf("parse(%q): want a syntax error, got none", source)
	}

	errs := make([]error, len(found))
	for i := range found {
		errs[i] = &found[i]
	}

	return errs
}

func TestPrecedence(t *testing.T) {
	tests := []struct {
		source string
		want   string
	}{
		// The book's motivating example: 6 / 3 - 1 parses as (6 / 3) - 1,
		// never 6 / (3 - 1).
		{"6 / 3 - 1", "(- (/ 6 3) 1)"},

		// Factor binds tighter than term.
		{"1 + 2 * 3", "(+ 1 (* 2 3))"},
		{"1 * 2 + 3", "(+ (* 1 2) 3)"},
		{"2 * 3 + 4 * 5", "(+ (* 2 3) (* 4 5))"},
		{"1 - 2 / 3", "(- 1 (/ 2 3))"},
		{"1 / 2 - 3", "(- (/ 1 2) 3)"},

		// Term binds tighter than comparison.
		{"1 + 2 < 3", "(< (+ 1 2) 3)"},
		{"1 < 2 + 3", "(< 1 (+ 2 3))"},
		{"3 > 2 + 1", "(> 3 (+ 2 1))"},

		// Comparison binds tighter than equality.
		{"1 < 2 == true", "(== (< 1 2) true)"},
		{"1 == 2 < 3", "(== 1 (< 2 3))"},
		{"1 < 2 == 3 < 4", "(== (< 1 2) (< 3 4))"},

		// Every level at once.
		{"1 + 2 * 3 < 4 == true", "(== (< (+ 1 (* 2 3)) 4) true)"},
		{"true == 1 < 2 + 3 * 4", "(== true (< 1 (+ 2 (* 3 4))))"},

		// The book's final example from section 6.4.
		{"1 - (2 * 3) < 4 == false", "(== (< (- 1 (group (* 2 3))) 4) false)"},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			if got := parse(t, tt.source); got != tt.want {
				t.Errorf("parse(%q)\nwant: %s\ngot:  %s", tt.source, tt.want, got)
			}
		})
	}
}

func TestAssociativity(t *testing.T) {
	tests := []struct {
		source string
		want   string
	}{
		// Subtraction and addition are left-associative.
		{"5 - 3 - 1", "(- (- 5 3) 1)"},
		{"1 - 2 - 3 - 4", "(- (- (- 1 2) 3) 4)"},
		{"1 + 2 + 3 + 4", "(+ (+ (+ 1 2) 3) 4)"},

		// Multiplication and division are left-associative.
		{"8 / 4 / 2", "(/ (/ 8 4) 2)"},
		{"2 * 3 * 4", "(* (* 2 3) 4)"},

		// Comparison and equality are left-associative.
		{"1 < 2 < 3", "(< (< 1 2) 3)"},
		{"1 == 2 == 3", "(== (== 1 2) 3)"},
		{"1 != 2 != 3", "(!= (!= 1 2) 3)"},

		// The book's a == b == c == d == e sequence, with literals.
		{"1 == 2 == 3 == 4 == 5", "(== (== (== (== 1 2) 3) 4) 5)"},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			if got := parse(t, tt.source); got != tt.want {
				t.Errorf("parse(%q)\nwant: %s\ngot:  %s", tt.source, tt.want, got)
			}
		})
	}
}

func TestGrouping(t *testing.T) {
	tests := []struct {
		source string
		want   string
	}{
		{"(1 + 2) * 3", "(* (group (+ 1 2)) 3)"},
		{"1 * (2 + 3)", "(* 1 (group (+ 2 3)))"},
		{"(1)", "(group 1)"},
		{"((1))", "(group (group 1))"},
		{"((1 + 2))", "(group (group (+ 1 2)))"},

		// Grouping overrides precedence in both directions.
		{"(1 + 2) * 3", "(* (group (+ 1 2)) 3)"},
		{"1 + (2 * 3)", "(+ 1 (group (* 2 3)))"},

		// Grouping around a single primary.
		{"(1)", "(group 1)"},
		{"(1 + 2)", "(group (+ 1 2))"},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			if got := parse(t, tt.source); got != tt.want {
				t.Errorf("parse(%q)\nwant: %s\ngot:  %s", tt.source, tt.want, got)
			}
		})
	}
}

func TestUnary(t *testing.T) {
	tests := []struct {
		source string
		want   string
	}{
		{"-1", "(- 1)"},
		{"!true", "(! true)"},
		{"-true", "(- true)"},

		// Unary operators nest, and are right-associative.
		{"!!true", "(! (! true))"},
		{"--1", "(- (- 1))"},
		{"!-1", "(! (- 1))"},

		// Unary binds tighter than any binary operator.
		{"-1 * 2", "(* (- 1) 2)"},
		{"!true == false", "(== (! true) false)"},
		{"-1 + 2", "(+ (- 1) 2)"},

		// But grouping loosens it: -(1 + 2) is not (-1) + 2.
		{"-(1 + 2)", "(- (group (+ 1 2)))"},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			if got := parse(t, tt.source); got != tt.want {
				t.Errorf("parse(%q)\nwant: %s\ngot:  %s", tt.source, tt.want, got)
			}
		})
	}
}

func TestLiterals(t *testing.T) {
	tests := []struct {
		source string
		want   string
	}{
		{"123", "123"},
		{"1234", "1234"},
		{"12.34", "12.34"},
		{`"hello"`, "hello"},
		{"true", "true"},
		{"false", "false"},
		{"nil", "nil"},

		// A number followed by an identifier boundary.
		{"1a", "1"},

		// Whitespace between everything changes nothing.
		{"  1  ", "1"},
		{"\n1\n", "1"},
		{"\t1\t", "1"},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			if got := parse(t, tt.source); got != tt.want {
				t.Errorf("parse(%q)\nwant: %s\ngot:  %s", tt.source, tt.want, got)
			}
		})
	}
}

func TestSyntaxErrors(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{"missing closing paren", "(1 + 2"},
		{"empty parens", "()"},
		{"missing operand after plus", "1 +"},
		{"missing operand after star", "1 *"},
		{"missing operand after equality", "1 =="},
		{"leading binary operator", "+ 1"},
		{"lone semicolon", ";"},
		{"bare operator", "*"},
		{"only whitespace", "   "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parseErrors(t, tt.source)
		})
	}
}

func TestErrorMessages(t *testing.T) {
	tests := []struct {
		source string
		want   string
	}{
		{
			source: "(1 + 2",
			want:   "syntax error on line 0 at end: Expect ')' after expression.",
		},
		{
			source: "+ 1",
			want:   "syntax error on line 0 at '+': Missing left-hand operand.",
		},
		{
			// Not a binary operator, so the catch-all handles it. The lexer
			// also flags '@', so @ 1 is two independent problems.
			source: "@ 1",
			want:   "syntax error on line 0 at '@': Expect expression.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			errs := parseErrors(t, tt.source)
			if got := errs[0].Error(); got != tt.want {
				t.Errorf("parse(%q)\nwant: %s\ngot:  %s", tt.source, tt.want, got)
			}
		})
	}
}

func TestErrorReportsLineNumber(t *testing.T) {
	tokens := lexer.NewLexer("1 +\n2 +\n;").ScanTokens()
	p := NewParser(tokens)

	p.Parse()

	errs := p.Errors()
	if len(errs) == 0 {
		t.Fatal("want a syntax error, got none")
	}

	// The stray ';' is on line 2.
	if want := "syntax error on line 2 at ';': Expect expression."; errs[0].Error() != want {
		t.Errorf("want: %s\ngot:  %s", want, errs[0].Error())
	}
}

// The parser must not crash on malformed input, which is valid *input* to a
// parser even when it isn't valid code.
func TestParserDoesNotCrashOnMalformedInput(t *testing.T) {
	sources := []string{
		"(", ")", "(())", "(((", ")))", "+", "-", "!", "1 +", "1 -",
		"* 1", "/ 1", "==", "!=", "<", ">", "<=", ">=", "1 == == 2",
		"1 + + 2", "1 * * 2", "((((1))))", "\"unterminated",
		"1 ; 2 ; 3", "let x =", "let = 1", "@", "1 @ 2",
	}

	for _, source := range sources {
		t.Run(source, func(t *testing.T) {
			tokens := lexer.NewLexer(source).ScanTokens()
			p := NewParser(tokens)

			// The only requirement is that this returns rather than panics.
			p.Parse()
		})
	}
}

func TestParseReturnsNilOnSyntaxError(t *testing.T) {
	tokens := lexer.NewLexer("+ 1").ScanTokens()
	p := NewParser(tokens)

	expr, err := p.Parse()
	if err != nil {
		t.Errorf("want nil error, got %v", err)
	}
	if expr != nil {
		t.Errorf("want nil expression on syntax error, got %#v", expr)
	}
	if len(p.Errors()) == 0 {
		t.Error("want the error recorded in Errors()")
	}
}
