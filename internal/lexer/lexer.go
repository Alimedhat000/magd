package lexer

import "magd/internal/token"

// TODO: add support for unicode (and emojies) instead of ascii

type Lexer struct {
	input        string
	position     int  // current position in input (points to current char)
	readPosition int  // current reading position in input (after current char)
	character    byte // current char under examination
}

func New(input string) *Lexer {
	l := &Lexer{input: input}
	l.readChar()
	return l
}

// readChar puts the current character at index `l.readPosition` into
// `l.character` and increments the two pointers `l.position`, and `l.readPosition`
func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.character = 0 // 0 is the ascii for NUL
	} else {
		l.character = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition += 1
}

func (l *Lexer) NextToken() token.Token {
	var tok token.Token
	switch l.character {
	case '=':
		tok = newToken(token.ASSIGN, l.character)
	case ';':
		tok = newToken(token.SEMICOLON, l.character)
	case '(':
		tok = newToken(token.LEFTPAREN, l.character)
	case ')':
		tok = newToken(token.RIGHTPAREN, l.character)
	case ',':
		tok = newToken(token.COMMA, l.character)
	case '+':
		tok = newToken(token.PLUS, l.character)
	case '{':
		tok = newToken(token.LEFTBRACE, l.character)
	case '}':
		tok = newToken(token.RIGHTBRACE, l.character)
	case 0:
		tok.Literal = ""
		tok.Type = token.EOF
	default:
		if isLetter(l.character) {
			tok.Literal = l.readIdentifier()
			return tok
		} else {
			tok = newToken(token.ILLEGAL, l.character)
		}
	}
	l.readChar()
	return tok
}

func (l *Lexer) readIdentifier() string {
	startingPosition := l.position

	for isLetter(l.character) {
		l.readChar()
	}

	return l.input[startingPosition:l.position]
}

func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_'
}

func newToken(tokenType token.TokenType, ch byte) token.Token {
	return token.Token{Type: tokenType, Literal: string(ch)}
}
