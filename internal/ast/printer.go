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
		result, err := expr.Accept(p)
		if err != nil {
			return "", err
		}

		builder.WriteString(result.(string))
	}

	builder.WriteString(")")
	return builder.String(), nil
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
