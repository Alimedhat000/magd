package lexer

import (
	"testing"

	"magd/internal/token"
)

func TestUnicodeIdentifier(t *testing.T) {
	tokenSeq(t, "let café = 1",
		Token{Type: token.LET, Lexeme: "let", Literal: nil, Line: 0},
		Token{Type: token.IDENT, Lexeme: "café", Literal: nil, Line: 0},
		Token{Type: token.EQUAL, Lexeme: "=", Literal: nil, Line: 0},
		Token{Type: token.NUMBER, Lexeme: "1", Literal: float64(1), Line: 0},
		Token{Type: token.EOF, Literal: nil, Line: 0},
	)
}

func TestUnicodeIdentifiersWithDigits(t *testing.T) {
	tokenSeq(t, "π2 naïve_日本",
		Token{Type: token.IDENT, Lexeme: "π2", Literal: nil, Line: 0},
		Token{Type: token.IDENT, Lexeme: "naïve_日本", Literal: nil, Line: 0},
		Token{Type: token.EOF, Literal: nil, Line: 0},
	)
}

func TestUnicodeStringLiteral(t *testing.T) {
	tokenSeq(t, `"héllo 🎉 世界"`,
		Token{Type: token.STRING, Lexeme: `"héllo 🎉 世界"`, Literal: "héllo 🎉 世界", Line: 0},
		Token{Type: token.EOF, Literal: nil, Line: 0},
	)
}

func TestUnicodeFollowedByOperators(t *testing.T) {
	tokenSeq(t, "α+β",
		Token{Type: token.IDENT, Lexeme: "α", Literal: nil, Line: 0},
		Token{Type: token.PLUS, Lexeme: "+", Literal: nil, Line: 0},
		Token{Type: token.IDENT, Lexeme: "β", Literal: nil, Line: 0},
		Token{Type: token.EOF, Literal: nil, Line: 0},
	)
}

func TestEmojiOutsideStringIsSingleIllegalToken(t *testing.T) {
	l := NewLexer("🎉")
	toks := l.ScanTokens()

	want := []Token{
		{Type: token.ILLEGAL, Lexeme: "🎉", Literal: nil, Line: 0},
		{Type: token.EOF, Literal: nil, Line: 0},
	}
	if len(toks) != len(want) {
		t.Fatalf("want %d tokens, got %d: %#v", len(want), len(toks), toks)
	}
	if toks[0].Type != token.ILLEGAL || toks[0].Lexeme != "🎉" {
		t.Errorf("want single ILLEGAL token %q, got %#v", "🎉", toks[0])
	}

	if errs := l.Errors(); len(errs) != 1 {
		t.Fatalf("want 1 error for 1 emoji, got %d: %v", len(errs), errs)
	}
}

func TestUnicodeDoesNotBreakLineCounting(t *testing.T) {
	tokenSeq(t, "α\nβ",
		Token{Type: token.IDENT, Lexeme: "α", Literal: nil, Line: 0},
		Token{Type: token.IDENT, Lexeme: "β", Literal: nil, Line: 1},
		Token{Type: token.EOF, Literal: nil, Line: 1},
	)
}

func TestUnicodeMultilineString(t *testing.T) {
	tokenSeq(t, "\"日本\n語\"",
		Token{Type: token.STRING, Lexeme: "\"日本\n語\"", Literal: "日本\n語", Line: 1},
		Token{Type: token.EOF, Literal: nil, Line: 1},
	)
}
