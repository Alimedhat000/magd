package ast

import (
	"fmt"
	"strings"
)

type Printer struct{}

func (p *Printer) Print(expr Expression) (string, error) {
	result, err := expr.Accept(p)
	if err != nil {
		return "", err
	}
	return result.(string), nil
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
