// Package token
package token

import "fmt"

// TODO: add the filename, and position to the token for better errors

type TokenType int

const (
	EOF     = iota // end of file
	ILLEGAL        // a token that we don't know about

	IDENT // identifier like function/variable names
	NUMBER
	STRING

	// Operators
	ASSIGN
	PLUS
	MINUS
	DOT
	SLASH
	STAR
	BANG
	BANGEQUAL
	EQUAL
	EQUALEQUAL
	GREATER
	GREATEREQUAL
	LESS
	LESSEQUAL

	// Delimiters
	COMMA
	SEMICOLON
	LEFTPAREN
	RIGHTPAREN
	LEFTBRACE
	RIGHTBRACE

	// Keywords
	LET
	FUNCTION
	TRUE
	FALSE
	FOR
	WHILE
	BREAK
	CONTINUE
	IF
	ELSE
	RETURN
	AND
	OR
	NIL
	PRINT
	CLASS
	SUPER
	THIS
)

var TokenTypeStringMap = map[TokenType]string{
	EOF:     "EOF",     // end of file
	ILLEGAL: "ILLEGAL", // a token that we don't know about

	IDENT:  "IDENT", // identifier like function/variable names
	NUMBER: "INT",
	STRING: "STRING",

	// Operators
	ASSIGN: "=",
	PLUS:   "+",
	MINUS:  "-",
	DOT:    ".",
	SLASH:  "/",
	STAR:   "*",

	// Delimiters
	COMMA:        ",",
	SEMICOLON:    ";",
	BANG:         "!",
	BANGEQUAL:    "!=",
	EQUAL:        "=",
	EQUALEQUAL:   "==",
	GREATER:      ">",
	GREATEREQUAL: ">=",
	LESS:         "<",
	LESSEQUAL:    "<=",

	LEFTPAREN:  "(",
	RIGHTPAREN: ")",
	LEFTBRACE:  "{",
	RIGHTBRACE: "}",

	// Keywords
	LET:      "LET",
	FUNCTION: "FUNCTION",
	TRUE:     "TRUE",
	FALSE:    "FALSE",
	FOR:      "FOR",
	WHILE:    "WHILE",
	BREAK:    "BREAK",
	CONTINUE: "CONTINUE",
	IF:       "IF",
	ELSE:     "ELSE",
	RETURN:   "RETURN",
	AND:      "AND",
	OR:       "OR",
	NIL:      "NIL",
	PRINT:    "PRINT",
}

type Token struct {
	Type    TokenType
	Lexeme  string // the token extracted from the code as a string
	Literal any    // the literal value of the token
	Line    int
}

func (t Token) String() string {
	literalStr := ""

	if t.Literal != nil {
		literalStr = fmt.Sprintf("literal: %#v, ", t.Literal)
	}

	return fmt.Sprintf(
		"Token{type: %v, Lexeme: %q, %sLine: %d}",
		t.Type,
		t.Lexeme,
		literalStr,
		t.Line,
	)
}
