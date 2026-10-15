package eval

// The StmtVisitor half of the evaluator: one method per statement node.
//
// Statements that open a scope — block, for — swap e.environment for a fresh
// child and restore it with a defer, so the restore happens on the error path
// too. Statements that can transfer control out of a loop or function do so by
// returning one of the signals in errs; those signals travel as the error value
// and are caught at the boundary that owns them.

import (
	"errors"
	"fmt"
	"os"

	"magd/internal/ast"
	"magd/internal/env"
	"magd/internal/errs"
)

func (e *Evaluator) VisitBlockStmt(stmt *ast.BlockStmt) (any, error) {
	previous := e.environment
	e.environment = env.NewEnvironmentEnclosing(previous)

	defer func() { e.environment = previous }()

	for _, inner := range stmt.Statements {
		if err := e.Execute(inner); err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func (e *Evaluator) VisitIfStmt(stmt *ast.IfStmt) (any, error) {
	condition, err := e.evaluate(stmt.Condition)
	if err != nil {
		return nil, err
	}

	if e.isTruthy(condition) {
		err = e.Execute(stmt.ThenBranch)
		if err != nil {
			return nil, err
		}
	} else if stmt.ElseBranch != nil {
		err = e.Execute(stmt.ElseBranch)
		if err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (e *Evaluator) VisitReturnStmt(stmt *ast.ReturnStmt) (any, error) {
	var value any

	if stmt.Value != nil {
		evaluated, err := e.evaluate(stmt.Value)
		if err != nil {
			return nil, err
		}
		value = evaluated
	}

	return nil, errs.ReturnSignal{Value: value}
}

func (e *Evaluator) VisitBreakStmt(stmt *ast.BreakStmt) (any, error) {
	return nil, errs.BreakSignal{}
}

func (e *Evaluator) VisitContinueStmt(stmt *ast.ContinueStmt) (any, error) {
	return nil, errs.ContinueSignal{}
}

func (e *Evaluator) VisitForStmt(stmt *ast.ForStmt) (any, error) {
	// The loop variable belongs to the loop, not the enclosing scope.
	previous := e.environment
	e.environment = env.NewEnvironmentEnclosing(previous)
	defer func() { e.environment = previous }()

	if stmt.Initializer != nil {
		if err := e.Execute(stmt.Initializer); err != nil {
			return nil, err
		}
	}
	for {
		condition, err := e.evaluate(stmt.Condition)
		if err != nil {
			return nil, err
		}
		if !e.isTruthy(condition) {
			break
		}
		err = e.Execute(stmt.Body)

		switch {
		case err == nil:
			// body executed successfully.
		case errors.As(err, new(errs.BreakSignal)):
			return nil, nil
		case errors.As(err, new(errs.ContinueSignal)):
			// falls through on purpose: the increment must still run
		default:
			return nil, err
		}

		if stmt.Increment != nil {
			if _, err := e.evaluate(stmt.Increment); err != nil {
				return nil, err
			}
		}
	}

	return nil, nil
}

func (e *Evaluator) VisitWhileStmt(stmt *ast.WhileStmt) (any, error) {
	for {
		condition, err := e.evaluate(stmt.Condition)
		if err != nil {
			return nil, err
		}

		if !e.isTruthy(condition) {
			break
		}

		err = e.Execute(stmt.Body)

		switch {
		case errors.As(err, new(errs.BreakSignal)):
			return nil, nil

		case errors.As(err, new(errs.ContinueSignal)):
			continue
		case err == nil:
			continue

		default:
			return nil, err
		}

	}

	return nil, nil
}

func (e *Evaluator) VisitExpressionStmt(stmt *ast.ExpressionStmt) (any, error) {
	return e.evaluate(stmt.Expr)
}

func (e *Evaluator) VisitVariableStmt(stmt *ast.VariableStmt) (any, error) {
	var value any

	if stmt.Initializer != nil {
		var err error
		value, err = e.evaluate(stmt.Initializer)
		if err != nil {
			return nil, err
		}
	}

	e.environment.Define(stmt.Name.Lexeme, value)

	return nil, nil
}

func (e *Evaluator) VisitPrintStmt(stmt *ast.PrintStmt) (any, error) {
	val, err := e.evaluate(stmt.Expr)
	if err != nil {
		return nil, err
	}
	out := e.out
	if out == nil {
		out = os.Stdout
	}
	fmt.Fprintln(out, Stringify(val))
	return nil, nil
}

func (e *Evaluator) VisitFunctionStmt(stmt *ast.FunctionStmt) (any, error) {
	function := &Function{Declaration: stmt, Closure: e.environment}

	e.environment.Define(stmt.Name.Lexeme, function)

	return nil, nil
}
