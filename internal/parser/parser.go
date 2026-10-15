// Package parser
package parser

import (
	"fmt"

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
// program        → declaration* EOF ;
// declaration    → FunDecl | LetDecl | statement ;
// statement      → exprStmt | printStmt | ifStmt | whileStmt | forStmt | returnStmt | block ;
// returnStmt     → "return" expression? ";" ;
// ifStmt         → "if" "(" expression ")" statement ("else" statement)? ; this way we can chain if to else if as if is a stmt
// whileStmt      → "while" "(" expression ")" statement ;
// forStmt        → "for" "(" ( variableDecl | exprStmt | ";" ) expression? ";" expression? ")" statement ; desugars to while + a block with the increment appended to the body
// exprStmt       → expression ";" ;
// printStmt      → "print" expression ";" ;
// variableDecl   → "let" IDENTIFIER ( "=" expression)? ";" ;
// FunDecl        → "fun" function ;
// function       → IDENTIFIER "(" parameters? ")" block ;
// parameters     → IDENTIFIER ( "," IDENTIFIER )* ;
// block          → "{" declaration* "}";
//
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

func (p *Parser) declaration() (d ast.Stmt) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(parseError); !ok {
				panic(r) // a real bug, not a syntax error
			}
			p.synchronize()
			d = nil // caller skips it
		}
	}()

	if p.match(token.LET) {
		return p.varDeclaration()
	}

	if p.match(token.FUNCTION) {
		return p.funDeclaration()
	}

	return p.statement()
}

// TODO: add a breakStmt route here ("break" ";").

// TODO: add continueStmt only together with the forStatement change below.
// see this https://github.com/munificent/craftinginterpreters/issues/119#issuecomment-318807716

func (p *Parser) statement() ast.Stmt {
	if p.match(token.PRINT) {
		return p.printStatement()
	}
	if p.match(token.IF) {
		return p.ifStatement()
	}
	if p.match(token.FOR) {
		return p.forStatement()
	}
	if p.match(token.WHILE) {
		return p.whileStatement()
	}
	if p.match(token.BREAK) {
		return p.breakStatement()
	}
	if p.match(token.CONTINUE) {
		return p.continueStatement()
	}
	if p.match(token.LEFTBRACE) {
		return p.block()
	}
	if p.match(token.RETURN) {
		return p.returnStatement()
	}

	return p.expressionStatement()
}

func (p *Parser) returnStatement() ast.Stmt {
	keyword := p.previous()
	var value ast.Expression

	if !p.check(token.SEMICOLON) {
		value = p.expression()
	}

	p.consume(token.SEMICOLON, "Expect ; after return value.")

	return &ast.ReturnStmt{
		Keyword: keyword,
		Value:   value,
	}
}

func (p *Parser) continueStatement() ast.Stmt {
	if p.loopDepth == 0 {
		raise := p.error(
			p.previous(),
			"Cannot use 'continue' outside of a loop.",
		)
		panic(raise)
	}

	p.consume(token.SEMICOLON, "Expect ';' after continue.")

	return &ast.ContinueStmt{}
}

func (p *Parser) breakStatement() ast.Stmt {
	if p.loopDepth == 0 {
		raise := p.error(
			p.previous(),
			"Cannot use 'break' outside of a loop.",
		)
		panic(raise)
	}

	p.consume(token.SEMICOLON, "Expect ';' after break.")

	return &ast.BreakStmt{}
}

func (p *Parser) forStatement() ast.Stmt {
	p.consume(token.LEFTPAREN, "Expect '(' after 'for'.")

	// An empty initializer is the semicolon itself, so match it: there is
	// nothing else to consume. The declared nil is the empty case.
	var initializer ast.Stmt = nil
	if p.match(token.LET) {
		initializer = p.varDeclaration()
	} else if !p.match(token.SEMICOLON) {
		initializer = p.expressionStatement()
	}

	// Unlike the initializer, an empty condition is only looked ahead at.
	// Matching here would eat the semicolon and shift every later clause.
	var condition ast.Expression = nil
	if !p.check(token.SEMICOLON) {
		condition = p.expression()
	}
	p.consume(token.SEMICOLON, "Expect ';' after loop condition.")

	if condition == nil {
		condition = &ast.LiteralExpression{Value: true}
	}

	// match the ending )
	var increment ast.Expression = nil
	if !p.check(token.RIGHTPAREN) {
		increment = p.expression()
	}
	p.consume(token.RIGHTPAREN, "Expect ')' after for clauses.")

	p.loopDepth++
	defer func() { p.loopDepth-- }()
	body := p.statement()

	return &ast.ForStmt{
		Initializer: initializer,
		Condition:   condition,
		Increment:   increment,
		Body:        body,
	}
}

func (p *Parser) whileStatement() ast.Stmt {
	// consume (
	p.consume(token.LEFTPAREN, "Expect '(' after 'while'.")

	condition := p.expression()

	p.consume(token.RIGHTPAREN, "Expect ')' after while condition.")

	p.loopDepth += 1
	defer func() { p.loopDepth-- }()
	body := p.statement()

	return &ast.WhileStmt{
		Condition: condition,
		Body:      body,
	}
}

func (p *Parser) ifStatement() ast.Stmt {
	// consume (
	p.consume(token.LEFTPAREN, "Expect '(' after 'if'.")

	condition := p.expression()

	p.consume(token.RIGHTPAREN, "Expect ')' after if condition.")

	thenBranch := p.statement()
	var elseBranch ast.Stmt = nil

	if p.match(token.ELSE) {
		elseBranch = p.statement()
	}

	return &ast.IfStmt{
		Condition:  condition,
		ThenBranch: thenBranch,
		ElseBranch: elseBranch,
	}
}

func (p *Parser) funDeclaration() ast.Stmt {
	name := p.consume(token.IDENT, "Expect function name.")

	p.consume(token.LEFTPAREN, "Expect ( after a function name.")

	var params []token.Token

	for !p.check(token.RIGHTPAREN) {
		if len(params) >= maxArguments {
			str := fmt.Sprintf("Can't have more than %d arguments", maxArguments)
			raise := p.error(p.peek(), str)
			panic(raise)
		}

		params = append(params, p.consume(token.IDENT, "Expect parameter name."))

		if !p.match(token.COMMA) {
			break
		}
	}

	p.consume(token.RIGHTPAREN, "Expect ) after a function name.")

	p.consume(token.LEFTBRACE, "Expect { before a function body.")

	body := p.block()

	return &ast.FunctionStmt{Name: name, Parameters: params, Body: body}
}

func (p *Parser) varDeclaration() ast.Stmt {
	name := p.consume(token.IDENT, "Expect variable name.")

	var initializer ast.Expression
	if p.match(token.EQUAL) {
		initializer = p.expression()
	}

	p.consume(token.SEMICOLON, "Expect ';' after variable declaration.")

	return &ast.VariableStmt{Name: name, Initializer: initializer}
}

func (p *Parser) printStatement() ast.Stmt {
	value := p.expression()
	p.consume(token.SEMICOLON, "Expect ';' after value.")

	return &ast.PrintStmt{Expr: value}
}

func (p *Parser) expressionStatement() ast.Stmt {
	value := p.expression()
	p.consume(token.SEMICOLON, "Expect ';' after expression.")

	return &ast.ExpressionStmt{Expr: value}
}

// block parses a braced list of declarations and statements. The opening brace
// is already consumed by statement().
//
// There is no error recovery inside a block, matching the book: once inside
// braces, malformed input reports the error and unwinds out of the whole
// block. Resyncing mid-block would silently drop statements.
func (p *Parser) block() ast.Stmt {
	var statements []ast.Stmt

	for !p.check(token.RIGHTBRACE) && !p.isAtEnd() {
		if d := p.declaration(); d != nil {
			statements = append(statements, d)
		}
	}

	p.consume(token.RIGHTBRACE, "Expect '}' after block.")

	return &ast.BlockStmt{Statements: statements}
}

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
	expr := p.logic_or()

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

func (p *Parser) logic_or() ast.Expression {
	expr := p.logic_and()

	for p.match(token.OR) {
		operator := p.previous()
		right := p.logic_and()

		expr = &ast.LogicalExpression{Left: expr, Operator: operator, Right: right}
	}

	return expr
}

func (p *Parser) logic_and() ast.Expression {
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
