// Package parser
package parser

import (
	"magd/internal/ast"
	"magd/internal/errs"
	"magd/internal/token"
)

// TODO: add better comments for each of the functions telling what it
// does and what grammar it uses

// This is a Recursive descent parser (which is the simplest way to build one)
// it starts from the top/ outermost grammmar rule and works
// it's way down into the nested subexpressions
//
// This file holds the machinery: the parser's own state, the error and
// recovery path, and the handful of primitives that move along the token
// stream. The grammar rules themselves live in statements.go and
// expressions.go, and each of those files carries the part of the grammar it
// implements.
//
// program        → declaration* EOF ;

// parseError unwinds recursive descent to Parse. It carries no information:
// the actual error is recorded in Parser.errors before panicking.
type parseError struct{}

type Parser struct {
	tokens       []token.Token
	currentToken int
	errors       []errs.SyntaxError

	loopDepth int
}

// This one's for you magd ⁶🤷‍♂️⁷
const maxArguments = 67

func NewParser(tokens []token.Token) *Parser {
	return &Parser{tokens: tokens, loopDepth: 0}
}

// Errors returns the syntax errors found so far.
func (p *Parser) Errors() []errs.SyntaxError {
	return p.errors
}

// Parse parses a single expression. On a syntax error it recovers, records the
// error, and returns nil. Synchronization waits until the grammar has
// statements to resync on.
func (p *Parser) Parse() ([]ast.Stmt, error) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		if _, ok := r.(parseError); !ok {
			panic(r) // a real bug, not a syntax error
		}
	}()

	statements := []ast.Stmt{}
	for !p.isAtEnd() {
		if d := p.declaration(); d != nil {
			statements = append(statements, d)
		}
	}

	return statements, nil
}

func (p *Parser) error(t token.Token, message string) parseError {
	p.errors = append(p.errors, *errs.NewSyntaxError(t.Line, t.Lexeme, message))
	return parseError{}
}

func (p *Parser) synchronize() {
	p.advance()

	for !p.isAtEnd() {
		if p.previous().Type == token.SEMICOLON {
			return
		}

		switch p.peek().Type {
		case token.CLASS, token.FUNCTION, token.LET, token.FOR, token.IF, token.WHILE, token.PRINT, token.RETURN:
			return
		}

		p.advance()
	}
}

// match checks if the current token has any of the given types,
// if so it consumes the token with advance() and returns true.
func (p *Parser) match(types ...token.TokenType) bool {
	for _, t := range types {
		if !p.check(t) {
			continue
		}
		p.advance()
		return true

	}
	return false
}

// check returns True if the current token is of the give type
func (p *Parser) check(t token.TokenType) bool {
	if p.isAtEnd() {
		return false
	}
	return p.peek().Type == t
}

// advance consumes the current token and increases the current
func (p *Parser) advance() token.Token {
	if !p.isAtEnd() {
		p.currentToken += 1
	}
	return p.previous()
}

// consume requires the next token to be of the given type. If it isn't, that
// is a syntax error: record it and unwind.
func (p *Parser) consume(t token.TokenType, message string) token.Token {
	if p.check(t) {
		return p.advance()
	}

	raise := p.error(p.peek(), message)
	panic(raise)
}

func (p *Parser) isAtEnd() bool {
	return p.peek().Type == token.EOF
}

func (p *Parser) peek() token.Token {
	return p.tokens[p.currentToken]
}

func (p *Parser) previous() token.Token {
	return p.tokens[p.currentToken-1]
}
