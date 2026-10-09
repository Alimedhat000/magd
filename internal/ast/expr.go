// Package ast defines the abstract syntax tree: the in-memory representation
// of parsed source code.
package ast

// Nodes here are pure data with no behavior of their own. The parser builds
// them; other passes (printing, evaluating, name resolution) consume them, so
// no single phase owns them.
//
// Instead of adding a method per operation to every node, each node has one
// Accept method that dispatches to a Visitor. Each operation — AstPrinter,
// Evaluator — implements Visitor with one method per node type. Adding a new
// operation means writing a new type, not editing these nodes.

import "magd/internal/token"

type Expression interface {
	Accept(visitor Visitor) (any, error)
}

type Visitor interface {
	VisitBinaryExpression(expr *BinaryExpression) (any, error)
	VisitUnaryExpression(expr *UnaryExpression) (any, error)
	VisitGroupingExpression(expr *GroupingExpression) (any, error)
	VisitLiteralExpression(expr *LiteralExpression) (any, error)
	VisitConditionalExpression(expr *ConditionalExpression) (any, error)
}

type BinaryExpression struct {
	Left     Expression
	Operator token.Token
	Right    Expression
}

type UnaryExpression struct {
	Operator token.Token
	Right    Expression
}

type GroupingExpression struct {
	Expr Expression
}

type ConditionalExpression struct {
	Condition    Expression
	QuestionMark token.Token
	Consequent   Expression
	Colon        token.Token
	Alternative  Expression
}

type LiteralExpression struct {
	Value any // this could have any literal value NUMBER | STRING | TRUE | FALSE | NIL
}

func (expr *BinaryExpression) Accept(visitor Visitor) (any, error) {
	return visitor.VisitBinaryExpression(expr)
}

func (expr *UnaryExpression) Accept(visitor Visitor) (any, error) {
	return visitor.VisitUnaryExpression(expr)
}

func (expr *LiteralExpression) Accept(visitor Visitor) (any, error) {
	return visitor.VisitLiteralExpression(expr)
}

func (expr *GroupingExpression) Accept(visitor Visitor) (any, error) {
	return visitor.VisitGroupingExpression(expr)
}

func (expr *ConditionalExpression) Accept(visitor Visitor) (any, error) {
	return visitor.VisitConditionalExpression(expr)
}
