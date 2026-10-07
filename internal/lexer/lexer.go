// Package lexer
package lexer

import (
	"strconv"
	"unicode"
	"unicode/utf8"

	"magd/internal/lang"
	"magd/internal/token"
)

type Token = token.Token

var keywords = map[string]token.TokenType{
	"and":      token.AND,
	"break":    token.BREAK,
	"class":    token.CLASS,
	"continue": token.CONTINUE,
	"else":     token.ELSE,
	"false":    token.FALSE,
	"for":      token.FOR,
	"fun":      token.FUNCTION,
	"if":       token.IF,
	"let":      token.LET,
	"nil":      token.NIL,
	"or":       token.OR,
	"print":    token.PRINT,
	"return":   token.RETURN,
	"super":    token.SUPER,
	"this":     token.THIS,
	"true":     token.TRUE,
	"while":    token.WHILE,
}

type Lexer struct {
	source       string
	tokens       []Token
	lexemStart   int
	lexemCurrent int
	line         int
	errors       []lang.SyntaxError
}

func NewLexer(source string) *Lexer {
	return &Lexer{
		source: source,
	}
}

func (l *Lexer) Errors() []lang.SyntaxError {
	return l.errors
}

func (l *Lexer) ScanTokens() []Token {
	for !l.isAtEnd() {
		l.lexemStart = l.lexemCurrent
		l.scanToken()
	}

	l.tokens = append(l.tokens, Token{
		Type:    token.EOF,
		Lexeme:  "",
		Literal: nil,
		Line:    l.line,
	})

	return l.tokens
}

func (l *Lexer) advance() rune {
	if l.isAtEnd() {
		return 0
	}

	char, size := utf8.DecodeRuneInString(l.source[l.lexemCurrent:])
	l.lexemCurrent += size

	return char
}

func (l *Lexer) scanToken() {
	char := l.advance()

	switch char {
	case '(':
		l.addToken(token.LEFTPAREN)
	case ')':
		l.addToken(token.RIGHTPAREN)
	case '{':
		l.addToken(token.LEFTBRACE)
	case '}':
		l.addToken(token.RIGHTBRACE)
	case ',':
		l.addToken(token.COMMA)
	case '.':
		l.addToken(token.DOT)
	case '-':
		l.addToken(token.MINUS)
	case '+':
		l.addToken(token.PLUS)
	case ';':
		l.addToken(token.SEMICOLON)
	case '*':
		l.addToken(token.STAR)

	case '!':
		if l.matchNextCharacter('=') {
			l.addToken(token.BANGEQUAL)
		} else {
			l.addToken(token.BANG)
		}

	case '=':
		if l.matchNextCharacter('=') {
			l.addToken(token.EQUALEQUAL)
		} else {
			l.addToken(token.EQUAL)
		}

	case '<':
		if l.matchNextCharacter('=') {
			l.addToken(token.LESSEQUAL)
		} else {
			l.addToken(token.LESS)
		}

	case '>':
		if l.matchNextCharacter('=') {
			l.addToken(token.GREATEREQUAL)
		} else {
			l.addToken(token.GREATER)
		}

	case '/':

		if l.matchNextCharacter('/') {
			// found a comment skip till the next line
			for l.lookahead() != '\n' && !l.isAtEnd() {
				l.advance()
			}
		} else if l.matchNextCharacter('*') {
			l.handleMultiLineComment()
		} else {
			l.addToken(token.SLASH)
		}

	case '"':
		l.handleString()

	case '\n':
		l.line += 1

	case ' ', '\r', '\t':
	// Ignore whitespaces

	default:
		if isDigit(char) {
			l.handleNumber()
		} else if isAlpha(char) {
			l.handleIdentifier()
		} else {
			// we keep scanning. there may be other errors later in the program
			l.errors = append(l.errors, *lang.NewSyntaxError(l.line, nil, "Unexpexted character"))
			l.addToken(token.ILLEGAL)
		}
	}
}

func (l *Lexer) addToken(token token.TokenType) {
	l.addTokenWithLiteral(token, nil)
}

func (l *Lexer) addTokenWithLiteral(tokenType token.TokenType, literal any) {
	l.tokens = append(l.tokens, Token{
		Type:    tokenType,
		Lexeme:  l.source[l.lexemStart:l.lexemCurrent],
		Literal: literal,
		Line:    l.line,
	})
}

func (l *Lexer) handleString() {
	for l.lookahead() != '"' && !l.isAtEnd() {
		if l.lookahead() == '\n' {
			l.line += 1
		}
		l.advance()
	}

	if l.isAtEnd() {
		// no closing quote to skip, so only trim the opening one
		l.addTokenWithLiteral(token.STRING, l.source[l.lexemStart+1:l.lexemCurrent])

		l.errors = append(l.errors, *lang.NewSyntaxError(l.line, nil, "Unterminated string"))
		return
	}

	l.advance() // The closing "

	l.addTokenWithLiteral(token.STRING, l.source[l.lexemStart+1:l.lexemCurrent-1]) // trim the quotes
}

func (l *Lexer) handleIdentifier() {
	for isAlphaNumeric(l.lookahead()) {
		l.advance()
	}

	text := l.source[l.lexemStart:l.lexemCurrent]

	tokenType, ok := keywords[text]
	if !ok {
		tokenType = token.IDENT
	}

	l.addToken(tokenType)
}

func (l *Lexer) handleNumber() {
	for isDigit(l.lookahead()) {
		l.advance()
	}

	if l.lookahead() == '.' && isDigit(l.lookaheadNext()) {
		l.advance()
		for isDigit(l.lookahead()) {
			l.advance()
		}
	}

	text := l.source[l.lexemStart:l.lexemCurrent]

	value, err := strconv.ParseFloat(text, 64)
	if err != nil {
		l.errors = append(l.errors, *lang.NewSyntaxError(l.line, nil, err.Error()))
		return
	}

	l.addTokenWithLiteral(token.NUMBER, value)
}

func (l *Lexer) handleMultiLineComment() {
	for {
		if l.isAtEnd() {
			// unclosed multi-line comment
			l.errors = append(l.errors, *lang.NewSyntaxError(l.line, nil, "Unterminated block comment"))
			return
		}

		if l.lookahead() == '*' && l.lookaheadNext() == '/' {
			l.advance() // skip *
			l.advance() // skip /
			return
		}

		if l.lookahead() == '\n' {
			l.line += 1
		}

		l.advance()
	}
}

func (l *Lexer) matchNextCharacter(expected rune) bool {
	if l.isAtEnd() || rune(l.source[l.lexemCurrent]) != expected {
		return false
	}
	l.lexemCurrent += 1
	return true
}

func (l *Lexer) lookahead() rune {
	if l.isAtEnd() {
		return 0
	}
	char, _ := utf8.DecodeRuneInString(l.source[l.lexemCurrent:])
	return char
}

func (l *Lexer) lookaheadNext() rune {
	if l.lexemCurrent+1 >= len(l.source) {
		return 0
	}
	char, _ := utf8.DecodeRuneInString(l.source[l.lexemCurrent+1:])
	return char
}

func (l *Lexer) isAtEnd() bool {
	return l.lexemCurrent >= len(l.source)
}

func isDigit(char rune) bool {
	return char >= '0' && char <= '9' || unicode.IsDigit(char)
}

func isAlpha(char rune) bool {
	return char == '_' || unicode.IsLetter(char)
}

func isAlphaNumeric(char rune) bool {
	return isDigit(char) || isAlpha(char)
}
