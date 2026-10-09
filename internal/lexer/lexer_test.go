package lexer

import (
	"reflect"
	"testing"

	"magd/internal/token"
)

func scan(source string) []Token {
	return NewLexer(source).ScanTokens()
}

func tokenSeq(t *testing.T, src string, want ...Token) {
	t.Helper()
	got := scan(src)

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("scan(%q):\nwant: %#v\ngot:  %#v", src, want, got)
	}
}

func TestEmptySource(t *testing.T) {
	tokenSeq(t, "",
		Token{Type: token.EOF, Literal: nil, Line: 0},
	)
}

func TestSingleCharacterTokens(t *testing.T) {
	cases := map[rune]token.TokenType{
		'(': token.LEFTPAREN,
		')': token.RIGHTPAREN,
		'{': token.LEFTBRACE,
		'}': token.RIGHTBRACE,
		',': token.COMMA,
		'.': token.DOT,
		'-': token.MINUS,
		'+': token.PLUS,
		';': token.SEMICOLON,
		'?': token.QUESTION,
		':': token.COLON,
		'*': token.STAR,
	}

	for r, wantType := range cases {
		src := string(r)
		tokenSeq(t, src,
			Token{Type: wantType, Lexeme: src, Literal: nil, Line: 0},
			Token{Type: token.EOF, Literal: nil, Line: 0},
		)
	}
}

func TestTwoCharacterTokens(t *testing.T) {
	cases := map[string]token.TokenType{
		"!=": token.BANGEQUAL,
		"==": token.EQUALEQUAL,
		"<=": token.LESSEQUAL,
		">=": token.GREATEREQUAL,
	}

	for src, wantType := range cases {
		tokenSeq(t, src,
			Token{Type: wantType, Lexeme: src, Literal: nil, Line: 0},
			Token{Type: token.EOF, Literal: nil, Line: 0},
		)
	}
}

func TestSingleVersusPairedOperators(t *testing.T) {
	tokenSeq(t, "! = < >",
		Token{Type: token.BANG, Lexeme: "!", Literal: nil, Line: 0},
		Token{Type: token.EQUAL, Lexeme: "=", Literal: nil, Line: 0},
		Token{Type: token.LESS, Lexeme: "<", Literal: nil, Line: 0},
		Token{Type: token.GREATER, Lexeme: ">", Literal: nil, Line: 0},
		Token{Type: token.EOF, Literal: nil, Line: 0},
	)
}

func TestIdentifiersAndKeywords(t *testing.T) {
	tokenSeq(t, "let lets and andy fun foo _bar",
		Token{Type: token.LET, Lexeme: "let", Literal: nil, Line: 0},
		Token{Type: token.IDENT, Lexeme: "lets", Literal: nil, Line: 0},
		Token{Type: token.AND, Lexeme: "and", Literal: nil, Line: 0},
		Token{Type: token.IDENT, Lexeme: "andy", Literal: nil, Line: 0},
		Token{Type: token.FUNCTION, Lexeme: "fun", Literal: nil, Line: 0},
		Token{Type: token.IDENT, Lexeme: "foo", Literal: nil, Line: 0},
		Token{Type: token.IDENT, Lexeme: "_bar", Literal: nil, Line: 0},
		Token{Type: token.EOF, Literal: nil, Line: 0},
	)
}

func TestAllKeywords(t *testing.T) {
	cases := []struct {
		spelling string
		typ      token.TokenType
	}{
		{"let", token.LET},
		{"fun", token.FUNCTION},
		{"true", token.TRUE},
		{"false", token.FALSE},
		{"for", token.FOR},
		{"while", token.WHILE},
		{"break", token.BREAK},
		{"continue", token.CONTINUE},
		{"if", token.IF},
		{"else", token.ELSE},
		{"return", token.RETURN},
		{"and", token.AND},
		{"or", token.OR},
		{"nil", token.NIL},
		{"print", token.PRINT},
	}

	src := ""
	for i, c := range cases {
		if i > 0 {
			src += " "
		}
		src += c.spelling
	}

	got := scan(src)

	if len(got) != len(cases)+1 {
		t.Fatalf("want %d tokens, got %d: %#v", len(cases)+1, len(got), got)
	}

	for i, c := range cases {
		want := Token{Type: c.typ, Lexeme: c.spelling, Literal: nil, Line: 0}
		if !reflect.DeepEqual(got[i], want) {
			t.Errorf("token %d: want %#v, got %#v", i, want, got[i])
		}
	}

	if last := got[len(got)-1]; last.Type != token.EOF {
		t.Errorf("want trailing EOF, got %#v", last)
	}
}

func TestDigitsInIdentifiersAndNumbers(t *testing.T) {
	tokenSeq(t, "abc123 12abc",
		Token{Type: token.IDENT, Lexeme: "abc123", Literal: nil, Line: 0},
		Token{Type: token.NUMBER, Lexeme: "12", Literal: float64(12), Line: 0},
		Token{Type: token.IDENT, Lexeme: "abc", Literal: nil, Line: 0},
		Token{Type: token.EOF, Literal: nil, Line: 0},
	)
}

func TestNumbers(t *testing.T) {
	tokenSeq(t, "1234 12.34",
		Token{Type: token.NUMBER, Lexeme: "1234", Literal: float64(1234), Line: 0},
		Token{Type: token.NUMBER, Lexeme: "12.34", Literal: float64(12.34), Line: 0},
		Token{Type: token.EOF, Literal: nil, Line: 0},
	)
}

func TestStrings(t *testing.T) {
	tokenSeq(t, `"hello"`,
		Token{Type: token.STRING, Lexeme: `"hello"`, Literal: "hello", Line: 0},
		Token{Type: token.EOF, Literal: nil, Line: 0},
	)
}

func TestWhitespaceIsSkipped(t *testing.T) {
	tokenSeq(t, " + \t +\r +\n +",
		Token{Type: token.PLUS, Lexeme: "+", Literal: nil, Line: 0},
		Token{Type: token.PLUS, Lexeme: "+", Literal: nil, Line: 0},
		Token{Type: token.PLUS, Lexeme: "+", Literal: nil, Line: 0},
		Token{Type: token.PLUS, Lexeme: "+", Literal: nil, Line: 1},
		Token{Type: token.EOF, Literal: nil, Line: 1},
	)
}

func TestLineTracking(t *testing.T) {
	tokenSeq(t, "a\nb",
		Token{Type: token.IDENT, Lexeme: "a", Literal: nil, Line: 0},
		Token{Type: token.IDENT, Lexeme: "b", Literal: nil, Line: 1},
		Token{Type: token.EOF, Literal: nil, Line: 1},
	)
}

func TestUnexpectedCharacter(t *testing.T) {
	l := NewLexer("1 @ 2")
	toks := l.ScanTokens()

	want := []Token{
		{Type: token.NUMBER, Lexeme: "1", Literal: float64(1), Line: 0},
		{Type: token.ILLEGAL, Lexeme: "@", Literal: nil, Line: 0},
		{Type: token.NUMBER, Lexeme: "2", Literal: float64(2), Line: 0},
		{Type: token.EOF, Literal: nil, Line: 0},
	}
	if !reflect.DeepEqual(toks, want) {
		t.Fatalf("want %#v, got %#v", want, toks)
	}

	errs := l.Errors()
	if len(errs) != 1 {
		t.Fatalf("want 1 error, got %d: %v", len(errs), errs)
	}
	if errs[0].Error() == "" {
		t.Error("error message is empty")
	}
}

func TestUnterminatedString(t *testing.T) {
	l := NewLexer("\"oops")
	toks := l.ScanTokens()

	want := []Token{
		{Type: token.STRING, Lexeme: `"oops`, Literal: "oops", Line: 0},
		{Type: token.EOF, Literal: nil, Line: 0},
	}
	if !reflect.DeepEqual(toks, want) {
		t.Fatalf("want %#v, got %#v", want, toks)
	}

	errs := l.Errors()
	if len(errs) != 1 {
		t.Fatalf("want 1 error, got %d: %v", len(errs), errs)
	}
	if errs[0].Error() == "" {
		t.Error("error message is empty")
	}
}
