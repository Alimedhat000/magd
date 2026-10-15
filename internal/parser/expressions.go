package parser

// The expression half of the grammar, ordered by binding, loosest first. Each
// function handles the operators at its own level and delegates the rest down
// the chain, so reading this file top to bottom is reading the precedence
// table.

import (
	"magd/internal/ast"
	"magd/internal/token"
)

// expression     → assignment ;
// assignment     → IDENTIFIER "=" assignment | ternary ;
// ternary        → logic_or (? expression : ternary)? ; The alternative '?' recurses rather than looping, making it right-associative.
// logic_or       → logic_and ("or" logic_and)* ;
// logic_and      → equality ("and" equality)* ;
// equality       → comparison ( ( "!=" | "==" ) comparison )* ;
// comparison     → term ( ( ">" | ">=" | "<" | "<=" ) term )* ;
// term           → factor ( ( "-" | "+" ) factor )* ;
// factor         → unary ( ( "/" | "*" ) unary )* ;
// unary          → ( "!" | "-" ) unary | call ;
// call           → primary ( "(" arguments? ")" )* ;
// arguments      → expression ( "," expression )* ;
// primary        → NUMBER | STRING | "true" | "false" | "nil" | "(" expression ")" | IDENTIFIER ;

// TODO: add support for the comma operator like in comparison

func (p *Parser) expression() ast.Expression {
	return p.assignment()
}

func (p *Parser) assignment() ast.Expression {
	expr := p.ternary()

	if p.match(token.EQUAL) {
		equals := p.previous()
		value := p.assignment()

		// Assignment's target must be a bare name. A grouping or a literal is
		// a value, not somewhere to assign to.
		target, ok := expr.(*ast.VariableExpression)
		if !ok {
			raise := p.error(equals, "Invalid assignment target.")
			panic(raise)
		}

		return &ast.AssignmentExpression{
			Name:  target.Name,
			Value: value,
		}
	}

	return expr
}

func (p *Parser) ternary() ast.Expression {
	expr := p.logicOr()

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

func (p *Parser) logicOr() ast.Expression {
	expr := p.logicAnd()

	for p.match(token.OR) {
		operator := p.previous()
		right := p.logicAnd()

		expr = &ast.LogicalExpression{Left: expr, Operator: operator, Right: right}
	}

	return expr
}

func (p *Parser) logicAnd() ast.Expression {
	expr := p.equality()

	for p.match(token.AND) {
		operator := p.previous()
		right := p.equality()

		expr = &ast.LogicalExpression{Left: expr, Operator: operator, Right: right}
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
	return p.call()
}

func (p *Parser) call() ast.Expression {
	expr := p.primary()

	for {
		if p.match(token.LEFTPAREN) {
			expr = p.arguments(expr)
		} else {
			break
		}
	}

	return expr
}

func (p *Parser) arguments(calle ast.Expression) ast.Expression {
	var arguments []ast.Expression

	if !p.check(token.RIGHTPAREN) {
		for {

			if len(arguments) >= maxArguments {
				raise := p.error(p.peek(), "Can't have more than 255 arguments.")
				panic(raise)
			}
			arguments = append(arguments, p.expression())

			if !p.match(token.COMMA) {
				break
			}
		}
	}

	paren := p.consume(token.RIGHTPAREN, "Expect ) after arguments")

	return &ast.CallExpression{
		Calle:       calle,
		Parenthesis: paren,
		Arguments:   arguments,
	}
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

	if p.match(token.IDENT) {
		return &ast.VariableExpression{
			Name: p.previous(),
		}
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
