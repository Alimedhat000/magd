// Package errs holds diagnostics reported while compiling MAGD source:
// syntax errors from the lexer and parser, and later the runtime errors the
// evaluator produces.
package errs

import "fmt"

// SyntaxError is a problem with the structure of the source: an unexpected
// token, a missing operand, an unterminated string.
type SyntaxError struct {
	line    int
	lexeme  string // the offending token's text, empty at end of input
	message string
}

// NewSyntaxError reports a syntax error at the given line. Pass an empty
// lexeme for errors at the end of the input.
func NewSyntaxError(line int, lexeme, message string) *SyntaxError {
	return &SyntaxError{
		line,
		lexeme,
		message,
	}
}

func (e *SyntaxError) Error() string {
	if e.lexeme == "" {
		return fmt.Sprintf("syntax error on line %d at end: %s", e.line, e.message)
	}

	return fmt.Sprintf("syntax error on line %d at '%s': %s", e.line, e.lexeme, e.message)
}
