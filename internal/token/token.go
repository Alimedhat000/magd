package token

// TODO: add the filename, and position to the token for better errors
type TokenType string

const (
	EOF     = "EOF"     // end of file
	ILLEGAL = "ILLEGAL" // a token that we don't know about

	IDENT = "IDENT" // identifier like function/variable names
	INT   = "INT"

	// Operators
	ASSIGN = "="
	PLUS   = "+"

	// Delimiters
	COMMA     = ","
	SEMICOLON = ";"

	LEFTPAREN  = "("
	RIGHTPAREN = ")"
	LEFTBRACE  = "{"
	RIGHTBRACE = "}"

	// Keywords
	LET      = "LET"
	FUNCTION = "FUNCTION"
)

type Token struct {
	Type    TokenType
	Literal string
}
