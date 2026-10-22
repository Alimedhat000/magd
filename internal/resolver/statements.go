package resolver

import (
	"magd/internal/ast"
	"magd/internal/errs"
)

func (r *Resolver) VisitBlockStmt(stmt *ast.BlockStmt) (_ any, err error) {
	r.BeginScope()
	defer func() {
		if endErr := r.EndScope(); err == nil {
			err = endErr
		}
	}()

	err = r.ResolveStatements(stmt.Statements)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (r *Resolver) VisitVariableStmt(stmt *ast.VariableStmt) (any, error) {
	// The name is declared before the initializer is resolved and defined
	// after, so reading the name inside its own initializer is caught by the
	// variable visit.
	if err := r.Declare(stmt.Name); err != nil {
		return nil, err
	}
	if stmt.Initializer != nil {
		err := r.ResolveExpression(stmt.Initializer)
		if err != nil {
			return nil, err
		}
	}
	r.Define(stmt.Name)
	return nil, nil
}

func (r *Resolver) VisitFunctionStmt(stmt *ast.FunctionStmt) (any, error) {
	// Declaring and defining the name before resolving the body is what lets
	// the body call itself.
	if err := r.Declare(stmt.Name); err != nil {
		return nil, err
	}
	r.Define(stmt.Name)

	if err := r.ResolveFunction(stmt.Parameters, stmt.Body); err != nil {
		return nil, err
	}
	return nil, nil
}

func (r *Resolver) VisitClassStmt(stmt *ast.ClassStmt) (any, error) {
	enclosing := r.inClass
	r.inClass = true
	defer func() { r.inClass = enclosing }()

	err := r.Declare(stmt.Name)
	if err != nil {
		return nil, err
	}

	r.Define(stmt.Name)

	r.BeginScope()
	defer func() {
		if endErr := r.EndScope(); err == nil {
			err = endErr
		}
	}()

	if err := r.declareThis(); err != nil {
		return nil, err
	}

	for _, m := range stmt.Methods {
		if err := r.ResolveFunction(m.Parameters, m.Body); err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func (r *Resolver) VisitExpressionStmt(stmt *ast.ExpressionStmt) (any, error) {
	if err := r.ResolveExpression(stmt.Expr); err != nil {
		return nil, err
	}
	return nil, nil
}

func (r *Resolver) VisitPrintStmt(stmt *ast.PrintStmt) (any, error) {
	if err := r.ResolveExpression(stmt.Expr); err != nil {
		return nil, err
	}
	return nil, nil
}

func (r *Resolver) VisitIfStmt(stmt *ast.IfStmt) (any, error) {
	if err := r.ResolveExpression(stmt.Condition); err != nil {
		return nil, err
	}
	if err := r.ResolveStatement(stmt.ThenBranch); err != nil {
		return nil, err
	}
	if stmt.ElseBranch != nil {
		if err := r.ResolveStatement(stmt.ElseBranch); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (r *Resolver) VisitWhileStmt(stmt *ast.WhileStmt) (any, error) {
	if err := r.ResolveExpression(stmt.Condition); err != nil {
		return nil, err
	}
	if err := r.ResolveStatement(stmt.Body); err != nil {
		return nil, err
	}
	return nil, nil
}

// VisitForStmt mirrors the evaluator: the initializer runs in one fresh scope
// that the condition, body, and increment all resolve inside of too.
func (r *Resolver) VisitForStmt(stmt *ast.ForStmt) (_ any, err error) {
	r.BeginScope()
	defer func() {
		if endErr := r.EndScope(); err == nil {
			err = endErr
		}
	}()

	if stmt.Initializer != nil {
		if err := r.ResolveStatement(stmt.Initializer); err != nil {
			return nil, err
		}
	}
	if stmt.Condition != nil {
		if err := r.ResolveExpression(stmt.Condition); err != nil {
			return nil, err
		}
	}
	if err := r.ResolveStatement(stmt.Body); err != nil {
		return nil, err
	}
	if stmt.Increment != nil {
		if err := r.ResolveExpression(stmt.Increment); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (r *Resolver) VisitBreakStmt(stmt *ast.BreakStmt) (any, error) { return nil, nil }

func (r *Resolver) VisitContinueStmt(stmt *ast.ContinueStmt) (any, error) { return nil, nil }

func (r *Resolver) VisitReturnStmt(stmt *ast.ReturnStmt) (any, error) {
	// error if we are not in any function
	if !r.inFunction {
		return nil, errs.NewSyntaxError(
			stmt.Keyword.Line, stmt.Keyword.Lexeme,
			"Can't return from top-level code.")
	}

	if stmt.Value != nil {
		if err := r.ResolveExpression(stmt.Value); err != nil {
			return nil, err
		}
	}
	return nil, nil
}
