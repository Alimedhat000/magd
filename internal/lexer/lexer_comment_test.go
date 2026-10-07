package lexer

import (
	"testing"

	"magd/internal/token"
)

func TestLineComment(t *testing.T) {
	tokenSeq(t, "// a comment\nx",
		Token{Type: token.IDENT, Lexeme: "x", Literal: nil, Line: 1},
		Token{Type: token.EOF, Literal: nil, Line: 1},
	)
}

func TestLineCommentAtEndOfInput(t *testing.T) {
	tokenSeq(t, "x // trailing",
		Token{Type: token.IDENT, Lexeme: "x", Literal: nil, Line: 0},
		Token{Type: token.EOF, Literal: nil, Line: 0},
	)
}

func TestBlockComment(t *testing.T) {
	tokenSeq(t, "/* one line */ x",
		Token{Type: token.IDENT, Lexeme: "x", Literal: nil, Line: 0},
		Token{Type: token.EOF, Literal: nil, Line: 0},
	)
}

func TestBlockCommentMultiLine(t *testing.T) {
	tokenSeq(t, "/*\n * a doc comment\n * spanning lines\n */\nx",
		Token{Type: token.IDENT, Lexeme: "x", Literal: nil, Line: 4},
		Token{Type: token.EOF, Literal: nil, Line: 4},
	)
}

func TestEmptyBlockComment(t *testing.T) {
	tokenSeq(t, "/**/x",
		Token{Type: token.IDENT, Lexeme: "x", Literal: nil, Line: 0},
		Token{Type: token.EOF, Literal: nil, Line: 0},
	)
}

func TestStarInsideBlockCommentIsNotClosing(t *testing.T) {
	tokenSeq(t, "/* a * b */x",
		Token{Type: token.IDENT, Lexeme: "x", Literal: nil, Line: 0},
		Token{Type: token.EOF, Literal: nil, Line: 0},
	)
}

func TestStarSlashInsideBlockCommentIsNotClosing(t *testing.T) {
	tokenSeq(t, "/* look: * / still going */x",
		Token{Type: token.IDENT, Lexeme: "x", Literal: nil, Line: 0},
		Token{Type: token.EOF, Literal: nil, Line: 0},
	)
}

func TestSlashStarInsideBlockCommentDoesNotNest(t *testing.T) {
	tokenSeq(t, "/* outer /* inner */ x",
		Token{Type: token.IDENT, Lexeme: "x", Literal: nil, Line: 0},
		Token{Type: token.EOF, Literal: nil, Line: 0},
	)
}

func TestMultiplicationIsNotAComment(t *testing.T) {
	tokenSeq(t, "a * b",
		Token{Type: token.IDENT, Lexeme: "a", Literal: nil, Line: 0},
		Token{Type: token.STAR, Lexeme: "*", Literal: nil, Line: 0},
		Token{Type: token.IDENT, Lexeme: "b", Literal: nil, Line: 0},
		Token{Type: token.EOF, Literal: nil, Line: 0},
	)
}

func TestDivideIsNotAComment(t *testing.T) {
	tokenSeq(t, "a / b",
		Token{Type: token.IDENT, Lexeme: "a", Literal: nil, Line: 0},
		Token{Type: token.SLASH, Lexeme: "/", Literal: nil, Line: 0},
		Token{Type: token.IDENT, Lexeme: "b", Literal: nil, Line: 0},
		Token{Type: token.EOF, Literal: nil, Line: 0},
	)
}

func TestCommentBetweenTokens(t *testing.T) {
	tokenSeq(t, "a/* c */+/* d */b",
		Token{Type: token.IDENT, Lexeme: "a", Literal: nil, Line: 0},
		Token{Type: token.PLUS, Lexeme: "+", Literal: nil, Line: 0},
		Token{Type: token.IDENT, Lexeme: "b", Literal: nil, Line: 0},
		Token{Type: token.EOF, Literal: nil, Line: 0},
	)
}

func TestUnterminatedBlockComment(t *testing.T) {
	l := NewLexer("/* never closed")
	toks := l.ScanTokens()

	want := []Token{
		{Type: token.EOF, Literal: nil, Line: 0},
	}
	if len(toks) != len(want) {
		t.Fatalf("want %d tokens, got %d: %#v", len(want), len(toks), toks)
	}

	errs := l.Errors()
	if len(errs) != 1 {
		t.Fatalf("want 1 error, got %d: %v", len(errs), errs)
	}
}