// Package lang has magd language errors for now
package lang

import (
	"fmt"

	"magd/internal/token"
)

type SyntaxError struct {
	line    int
	token   *token.Token
	message string
}

func NewSyntaxError(line int, token *token.Token, message string) *SyntaxError {
	return &SyntaxError{
		line,
		token,
		message,
	}
}

func (e *SyntaxError) Error() string {
	return fmt.Sprintf(
		"syntax error on line %d: %s",
		e.line,
		e.message,
	)
}
