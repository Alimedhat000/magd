package ast

import (
	"fmt"
	"strings"
)

type Printer struct{}

var (
	_ ExprVisitor = (*Printer)(nil)
	_ StmtVisitor = (*Printer)(nil)
)

func (p *Printer) Print(stmts []Stmt) (string, error) {
	var builder strings.Builder

	for _, stmt := range stmts {
		result, err := stmt.Accept(p)
		if err != nil {
			return "", err
		}
		builder.WriteString(result.(string))
		builder.WriteString("\n")
	}

	return builder.String(), nil
}

func (p *Printer) VisitBinaryExpression(expr *BinaryExpression) (any, error) {
	return p.parenthesize(expr.Operator.Lexeme, expr.Left, expr.Right)
}

func (p *Printer) VisitUnaryExpression(expr *UnaryExpression) (any, error) {
	return p.parenthesize(expr.Operator.Lexeme, expr.Right)
}

func (p *Printer) VisitGroupingExpression(expr *GroupingExpression) (any, error) {
	return p.parenthesize("group", expr.Expr)
}

func (p *Printer) VisitConditionalExpression(expr *ConditionalExpression) (any, error) {
	return p.parenthesize("?", expr.Condition, expr.Consequent, expr.Alternative)
}

func (p *Printer) VisitLogicalExpression(expr *LogicalExpression) (any, error) {
	return p.parenthesize(expr.Operator.Lexeme, expr.Left, expr.Right)
}

// break and continue carry no data, so they print as their source form,
// semicolon included, like print and let.
func (p *Printer) VisitBreakStmt(stmt *BreakStmt) (any, error) {
	return "break;", nil
}

func (p *Printer) VisitContinueStmt(stmt *ContinueStmt) (any, error) {
	return "continue;", nil
}

func (p *Printer) VisitIfStmt(stmt *IfStmt) (any, error) {
	condition, err := p.printExpression(stmt.Condition)
	if err != nil {
		return nil, err
	}

	thenBranch, err := p.printStatement(stmt.ThenBranch)
	if err != nil {
		return nil, err
	}

	var builder strings.Builder
	builder.WriteString("if (" + condition + ") " + thenBranch)

	if stmt.ElseBranch != nil {
		elseBranch, err := p.printStatement(stmt.ElseBranch)
		if err != nil {
			return nil, err
		}
		builder.WriteString(" else " + elseBranch)
	}

	return builder.String(), nil
}

func (p *Printer) VisitWhileStmt(stmt *WhileStmt) (any, error) {
	condition, err := p.printExpression(stmt.Condition)
	if err != nil {
		return nil, err
	}

	body, err := p.printStatement(stmt.Body)
	if err != nil {
		return nil, err
	}

	return "while (" + condition + ") " + body, nil
}

// funDecl → "fun" IDENTIFIER ( params? ) block
//
// A declaration prints without a trailing ';', like if and while: the body
// already supplies its own braces.
func (p *Printer) VisitFunctionStmt(stmt *FunctionStmt) (any, error) {
	var params []string
	for _, param := range stmt.Parameters {
		params = append(params, param.Lexeme)
	}

	body, err := p.printStatement(stmt.Body)
	if err != nil {
		return nil, err
	}

	return "fun " + stmt.Name.Lexeme + "(" + strings.Join(params, ", ") + ") " + body, nil
}

// returnStmt → "return" expression? ";"
//
// The value is optional, so a bare return prints without one rather than with
// an empty slot.
func (p *Printer) VisitReturnStmt(stmt *ReturnStmt) (any, error) {
	if stmt.Value == nil {
		return "return;", nil
	}

	value, err := p.printExpression(stmt.Value)
	if err != nil {
		return nil, err
	}

	return "return " + value + ";", nil
}

// functionExpr → "fun" "(" parameters? ")" block
//
// Printed like every other parenthesised node, with the body as its third
// element, so an anonymous function round-trips through the parser.
func (p *Printer) VisitFunctionExpression(expr *FunctionExpression) (any, error) {
	var params []string
	for _, param := range expr.Parameters {
		params = append(params, param.Lexeme)
	}

	body, err := p.printStatement(expr.Body)
	if err != nil {
		return nil, err
	}

	return "(fun (" + strings.Join(params, ", ") + ") " + body + ")", nil
}

// call → primary "(" arguments? ")"
//
// Printed like any other parenthesised node, so a zero-argument call keeps its
// empty pair rather than trailing a space.
func (p *Printer) VisitCallExpression(expr *CallExpression) (any, error) {
	callee, err := p.printExpression(expr.Calle)
	if err != nil {
		return nil, err
	}

	if len(expr.Arguments) == 0 {
		return "(" + callee + ")", nil
	}

	var args []string
	for _, arg := range expr.Arguments {
		printed, err := p.printExpression(arg)
		if err != nil {
			return nil, err
		}
		args = append(args, printed)
	}

	return "(" + callee + " " + strings.Join(args, ", ") + ")", nil
}

func (p *Printer) VisitForStmt(stmt *ForStmt) (any, error) {
	var condition string = ""
	var initializer string = ""
	var increment string = ""
	var err error

	if stmt.Condition != nil {
		condition, err = p.printExpression(stmt.Condition)
		if err != nil {
			return nil, err
		}
	}

	if stmt.Initializer != nil {
		initializer, err = p.printStatement(stmt.Initializer)
		if err != nil {
			return nil, err
		}
		// printStatement terminates a statement with ';', and the separator
		// below supplies the one belonging to the for header.
		initializer = strings.TrimSuffix(initializer, ";")
	}

	if stmt.Increment != nil {
		increment, err = p.printExpression(stmt.Increment)
		if err != nil {
			return nil, err
		}
	}
	body, err := p.printStatement(stmt.Body)
	if err != nil {
		return nil, err
	}

	return "for (" + initializer + "; " + condition + "; " + increment + ") " + body, nil
}

func (p *Printer) VisitLiteralExpression(expr *LiteralExpression) (any, error) {
	if expr.Value == nil {
		return "nil", nil
	}
	return fmt.Sprintf("%v", expr.Value), nil
}

func (p *Printer) VisitBlockStmt(stmt *BlockStmt) (any, error) {
	var builder strings.Builder

	builder.WriteString("{\n")

	for _, inner := range stmt.Statements {
		result, err := inner.Accept(p)
		if err != nil {
			return nil, err
		}
		builder.WriteString(result.(string))
		builder.WriteString("\n")
	}

	builder.WriteString("}")

	return builder.String(), nil
}

func (p *Printer) VisitVariableExpression(expr *VariableExpression) (any, error) {
	return expr.Name.Lexeme, nil
}

func (p *Printer) VisitAssignmentExpression(expr *AssignmentExpression) (any, error) {
	return p.parenthesize("=", &VariableExpression{Name: expr.Name}, expr.Value)
}

func (p *Printer) VisitVariableStmt(stmt *VariableStmt) (any, error) {
	if stmt.Initializer == nil {
		return "let " + stmt.Name.Lexeme + ";", nil
	}

	result, err := stmt.Initializer.Accept(p)
	if err != nil {
		return nil, err
	}

	return "let " + stmt.Name.Lexeme + " = " + result.(string) + ";", nil
}

func (p *Printer) parenthesize(name string, exprs ...Expression) (string, error) {
	var builder strings.Builder

	builder.WriteString("(")
	builder.WriteString(name)

	for _, expr := range exprs {
		builder.WriteString(" ")
		result, err := p.printExpression(expr)
		if err != nil {
			return "", err
		}

		builder.WriteString(result)
	}

	builder.WriteString(")")
	return builder.String(), nil
}

func (p *Printer) printExpression(expr Expression) (string, error) {
	result, err := expr.Accept(p)
	if err != nil {
		return "", err
	}

	return result.(string), nil
}

func (p *Printer) printStatement(stmt Stmt) (string, error) {
	result, err := stmt.Accept(p)
	if err != nil {
		return "", err
	}

	return result.(string), nil
}

func (p *Printer) VisitExpressionStmt(stmt *ExpressionStmt) (any, error) {
	result, err := stmt.Expr.Accept(p)
	if err != nil {
		return nil, err
	}
	return result.(string) + ";", nil
}

func (p *Printer) VisitPrintStmt(stmt *PrintStmt) (any, error) {
	result, err := stmt.Expr.Accept(p)
	if err != nil {
		return nil, err
	}
	return "print " + result.(string) + ";", nil
}
