// Package parser
package parser

import (
	"magd/internal/ast"
	"magd/internal/errs"
	"magd/internal/token"
)

// This is a Recursive descent parser (which is the simplest way to build one)
// it starts from the top/ outermost grammmar rule and works
// it's way down into the nested subexpressions
//
// Here's the grammar directly from the book
// https://craftinginterpreters.com/parsing-expressions.html
//
// expression     → ternary ;
// ternary        → equality (? expression : ternary)? ; The alternative '?' recurses rather than looping, making it right-associative.
// equality       → comparison ( ( "!=" | "==" ) comparison )* ;
// comparison     → term ( ( ">" | ">=" | "<" | "<=" ) term )* ;
// term           → factor ( ( "-" | "+" ) factor )* ;
// factor         → unary ( ( "/" | "*" ) unary )* ;
// unary          → ( "!" | "-" ) unary | primary ;
// primary        → NUMBER | STRING | "true" | "false" | "nil" | "(" expression ")" ;

// TODO: add support for the comma operator like in comparison

// parseError unwinds recursive descent to Parse. It carries no information:
// the actual error is recorded in Parser.errors before panicking.
type parseError struct{}

type Parser struct {
	tokens       []token.Token
	currentToken int
	errors       []errs.SyntaxError
}

func NewParser(tokens []token.Token) *Parser {
	return &Parser{tokens: tokens}
}

// Errors returns the syntax errors found so far.
func (p *Parser) Errors() []errs.SyntaxError {
	return p.errors
}

// Parse parses a single expression. On a syntax error it recovers, records the
// error, and returns nil. Synchronization waits until the grammar has
// statements to resync on.
func (p *Parser) Parse() (ast.Expression, error) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		if _, ok := r.(parseError); !ok {
			panic(r) // a real bug, not a syntax error
		}
	}()

	expr := p.expression()

	if !p.isAtEnd() {
		raise := p.error(p.peek(), "Expect expression.")
		panic(raise)
	}

	return expr, nil
}

func (p *Parser) error(t token.Token, message string) parseError {
	p.errors = append(p.errors, *errs.NewSyntaxError(t.Line, t.Lexeme, message))
	return parseError{}
}

func (p *Parser) expression() ast.Expression {
	return p.ternary()
}

func (p *Parser) ternary() ast.Expression {
	expr := p.equality()

	if p.match(token.QUESTION) {
		questionMark := p.previous()
		consequent := p.expression()

		colon := p.consume(token.COLON, "Expect ':' after '?'.")

		alternative := p.ternary() // right-associative

		return &ast.ConditionalExpression{
			Condition:    expr,
			QuestionMark: questionMark,
			Consequent:   consequent,
			Colon:        colon,
			Alternative:  alternative,
		}
	}

	return expr
}

func (p *Parser) equality() ast.Expression {
	expr := p.comparison()

	for p.match(token.BANGEQUAL, token.EQUALEQUAL) {
		operator := p.previous()
		right := p.comparison()
		expr = &ast.BinaryExpression{
			Left:     expr,
			Operator: operator,
			Right:    right,
		}
	}

	return expr
}

func (p *Parser) comparison() ast.Expression {
	expr := p.term()

	for p.match(token.GREATER, token.GREATEREQUAL, token.LESS, token.LESSEQUAL) {
		operator := p.previous()
		right := p.term()

		expr = &ast.BinaryExpression{
			Left:     expr,
			Operator: operator,
			Right:    right,
		}
	}
	return expr
}

func (p *Parser) term() ast.Expression {
	expr := p.factor()

	for p.match(token.MINUS, token.PLUS) {
		operator := p.previous()
		right := p.factor()

		expr = &ast.BinaryExpression{
			Left:     expr,
			Operator: operator,
			Right:    right,
		}
	}
	return expr
}

func (p *Parser) factor() ast.Expression {
	expr := p.unary()

	for p.match(token.STAR, token.SLASH) {
		operator := p.previous()
		right := p.unary()

		expr = &ast.BinaryExpression{
			Left:     expr,
			Operator: operator,
			Right:    right,
		}
	}
	return expr
}

func (p *Parser) unary() ast.Expression {
	if p.match(token.BANG, token.MINUS) {
		operator := p.previous()
		right := p.unary()
		return &ast.UnaryExpression{
			Operator: operator,
			Right:    right,
		}
	}
	return p.primary()
}

func (p *Parser) primary() ast.Expression {
	if p.match(token.TRUE) {
		return &ast.LiteralExpression{Value: true}
	}
	if p.match(token.FALSE) {
		return &ast.LiteralExpression{Value: false}
	}
	if p.match(token.NIL) {
		return &ast.LiteralExpression{Value: nil}
	}

	if p.match(token.NUMBER, token.STRING) {
		return &ast.LiteralExpression{Value: p.previous().Literal}
	}

	if p.match(token.LEFTPAREN) {
		expr := p.expression()
		// we must have a closing parenthesis after the expression
		p.consume(token.RIGHTPAREN, "Expect ')' after expression.")
		return &ast.GroupingExpression{Expr: expr}
	}

	// How we get here: when the parser needs an expression it walks the whole
	// precedence chain — equality → comparison → term → factor → unary →
	// primary. Each rule checks whether the current token is an operator it
	// handles and loops if so, otherwise passes down. Every rule that hands
	// off has consumed nothing, so if we arrive at primary() on a token that
	// isn't a literal or "(", it means the chain descended without consuming
	// anything: there was no left-hand operand to bind that operator to.
	//
	// Nothing actually failed on the way down. term() matched "+" and entered
	// its loop correctly; it just had no left side to build from. The input
	// doesn't match the grammar — so this is a grammar-level problem, not a
	// broken rule.
	//
	// Because we now know the token is specifically a binary operator, we can
	// report it precisely and keep parsing.
	// That way the rest of the expression still reports its own
	// errors instead of being swallowed by this one.
	if token.BinaryOperators[p.peek().Type] {
		operator := p.advance()

		_ = p.error(operator, "Missing left-hand operand.")

		switch operator.Type {
		case token.PLUS, token.MINUS:
			p.factor()
		case token.STAR, token.SLASH:
			p.unary()
		default: // comparison and equality operators
			p.comparison()
		}
		return nil
	}

	if p.peek().Type == token.COLON {
		colon := p.advance() // this IS the colon
		_ = p.error(colon, "Missing ternary question mark.")
		return nil
	}

	// Nothing here can start an expression.
	raise := p.error(p.peek(), "Expect expression.")
	panic(raise)
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

	p.error(p.peek(), message)
	return token.Token{}
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
