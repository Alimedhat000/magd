package parser

// The statement half of the grammar. Each function below implements one rule.
// declaration() and statement() are the two dispatch points; everything else
// is reached from one of them.

import (
	"fmt"

	"magd/internal/ast"
	"magd/internal/token"
)

// declaration    → FunDecl | LetDecl | statement ;
// statement      → exprStmt | printStmt | ifStmt | whileStmt | forStmt | returnStmt | block ;
// returnStmt     → "return" expression? ";" ;
// ifStmt         → "if" "(" expression ")" statement ("else" statement)? ; this way we can chain if to else if as if is a stmt
// whileStmt      → "while" "(" expression ")" statement ;
// forStmt        → "for" "(" ( variableDecl | exprStmt | ";" ) expression? ";" expression? ")" statement ;
// exprStmt       → expression ";" ;
// printStmt      → "print" expression ";" ;
// variableDecl   → "let" IDENTIFIER ( "=" expression)? ";" ;
// FunDecl        → "fun" function ;
// function       → IDENTIFIER "(" parameters? ")" block ;
// parameters     → IDENTIFIER ( "," IDENTIFIER )* ;
// block          → "{" declaration* "}";

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
