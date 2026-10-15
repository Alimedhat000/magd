// Package ast defines the abstract syntax tree: the in-memory representation
// of parsed source code.
package ast

// Nodes here are pure data with no behavior of their own. The parser builds
// them; other passes (printing, evaluating, name resolution) consume them, so
// no single phase owns them.
//
// Instead of adding a method per operation to every node, each node has one
// Accept method that dispatches to a visitor. Each operation — AstPrinter,
// Evaluator — implements one visitor interface per hierarchy with a method per
// node type. Adding a new operation means writing a new type, not editing
// these nodes.
//
// Expressions and statements are separate hierarchies because they are
// consumed separately: expressions produce values, statements do not. Each has
// its own Accept and visitor interface.

import "magd/internal/token"

type Expression interface {
	Accept(visitor ExprVisitor) (any, error)
}

type ExprVisitor interface {
	VisitBinaryExpression(expr *BinaryExpression) (any, error)
	VisitUnaryExpression(expr *UnaryExpression) (any, error)
	VisitGroupingExpression(expr *GroupingExpression) (any, error)
	VisitLiteralExpression(expr *LiteralExpression) (any, error)
	VisitConditionalExpression(expr *ConditionalExpression) (any, error)
	VisitVariableExpression(expr *VariableExpression) (any, error)
	VisitAssignmentExpression(expr *AssignmentExpression) (any, error)
	VisitLogicalExpression(expr *LogicalExpression) (any, error)

	VisitCallExpression(expr *CallExpression) (any, error)
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

type VariableExpression struct {
	Name token.Token
}

type AssignmentExpression struct {
	Name  token.Token
	Value Expression
}

type LogicalExpression struct {
	Left     Expression
	Operator token.Token
	Right    Expression
}

type CallExpression struct {
	Calle       Expression
	Parenthesis token.Token
	Arguments   []Expression
}

func (expr *VariableExpression) Accept(visitor ExprVisitor) (any, error) {
	return visitor.VisitVariableExpression(expr)
}

func (expr *BinaryExpression) Accept(visitor ExprVisitor) (any, error) {
	return visitor.VisitBinaryExpression(expr)
}

func (expr *UnaryExpression) Accept(visitor ExprVisitor) (any, error) {
	return visitor.VisitUnaryExpression(expr)
}

func (expr *LiteralExpression) Accept(visitor ExprVisitor) (any, error) {
	return visitor.VisitLiteralExpression(expr)
}

func (expr *GroupingExpression) Accept(visitor ExprVisitor) (any, error) {
	return visitor.VisitGroupingExpression(expr)
}

func (expr *ConditionalExpression) Accept(visitor ExprVisitor) (any, error) {
	return visitor.VisitConditionalExpression(expr)
}

func (expr *AssignmentExpression) Accept(visitor ExprVisitor) (any, error) {
	return visitor.VisitAssignmentExpression(expr)
}

func (expr *LogicalExpression) Accept(visitor ExprVisitor) (any, error) {
	return visitor.VisitLogicalExpression(expr)
}

func (expr *CallExpression) Accept(visitor ExprVisitor) (any, error) {
	return visitor.VisitCallExpression(expr)
}
