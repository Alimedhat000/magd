package ast

import "magd/internal/token"

// Stmt is a statement: an expression evaluated for effect, or a declaration.
// Unlike an expression, a statement produces no value.
type Stmt interface {
	Accept(visitor StmtVisitor) (any, error)
}

type StmtVisitor interface {
	VisitExpressionStmt(stmt *ExpressionStmt) (any, error)
	VisitPrintStmt(stmt *PrintStmt) (any, error)
	VisitVariableStmt(stmt *VariableStmt) (any, error)
	VisitBlockStmt(stmt *BlockStmt) (any, error)
}

type (
	ExpressionStmt struct {
		Expr Expression
	}

	PrintStmt struct {
		Expr Expression
	}

	VariableStmt struct {
		Name        token.Token
		Initializer Expression
	}

	// BlockStmt is a braced group of declarations and statements. The braces
	// are not stored: nothing outside the block needs them, and keeping them
	// would mean every visitor had to know about source layout.
	BlockStmt struct {
		Statements []Stmt
	}
)

func (stmt *ExpressionStmt) Accept(visitor StmtVisitor) (any, error) {
	return visitor.VisitExpressionStmt(stmt)
}

func (stmt *PrintStmt) Accept(visitor StmtVisitor) (any, error) {
	return visitor.VisitPrintStmt(stmt)
}

func (stmt *VariableStmt) Accept(visitor StmtVisitor) (any, error) {
	return visitor.VisitVariableStmt(stmt)
}

func (stmt *BlockStmt) Accept(visitor StmtVisitor) (any, error) {
	return visitor.VisitBlockStmt(stmt)
}
