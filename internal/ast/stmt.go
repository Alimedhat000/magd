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
	VisitIfStmt(stmt *IfStmt) (any, error)
	VisitWhileStmt(stmt *WhileStmt) (any, error)
	VisitForStmt(stmt *ForStmt) (any, error)
	VisitBlockStmt(stmt *BlockStmt) (any, error)
	VisitBreakStmt(stmt *BreakStmt) (any, error)
	VisitContinueStmt(stmt *ContinueStmt) (any, error)
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

	IfStmt struct {
		Condition  Expression
		ThenBranch Stmt
		ElseBranch Stmt
	}

	WhileStmt struct {
		Condition Expression
		Body      Stmt
	}

	ForStmt struct {
		Initializer Stmt
		Condition   Expression
		Increment   Expression
		Body        Stmt
	}

	BreakStmt struct{}

	ContinueStmt struct{}
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

func (stmt *IfStmt) Accept(visitor StmtVisitor) (any, error) {
	return visitor.VisitIfStmt(stmt)
}

func (stmt *WhileStmt) Accept(visitor StmtVisitor) (any, error) {
	return visitor.VisitWhileStmt(stmt)
}

func (stmt *ForStmt) Accept(visitor StmtVisitor) (any, error) {
	return visitor.VisitForStmt(stmt)
}

func (stmt *ContinueStmt) Accept(visitor StmtVisitor) (any, error) {
	return visitor.VisitContinueStmt(stmt)
}

func (stmt *BreakStmt) Accept(visitor StmtVisitor) (any, error) {
	return visitor.VisitBreakStmt(stmt)
}
