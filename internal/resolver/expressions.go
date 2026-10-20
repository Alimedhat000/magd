package resolver

import (
	"magd/internal/ast"
	"magd/internal/errs"
)

func (r *Resolver) VisitVariableExpression(expr *ast.VariableExpression) (any, error) {
	if len(r.scopes) > 0 {
		scope, _ := r.peekScopes()
		if initialized, ok := scope[expr.Name.Lexeme]; ok && !initialized {
			return nil, errs.NewSyntaxError(expr.Name.Line, expr.Name.Lexeme,
				"Can't read local variable in its own initializer")
		}
	}

	r.ResolveLocal(expr, expr.Name)
	return nil, nil
}

func (r *Resolver) VisitAssignmentExpression(expr *ast.AssignmentExpression) (any, error) {
	if err := r.ResolveExpression(expr.Value); err != nil {
		return nil, err
	}
	r.ResolveLocal(expr, expr.Name)
	return nil, nil
}

func (r *Resolver) VisitBinaryExpression(expr *ast.BinaryExpression) (any, error) {
	if err := r.ResolveExpression(expr.Left); err != nil {
		return nil, err
	}
	if err := r.ResolveExpression(expr.Right); err != nil {
		return nil, err
	}
	return nil, nil
}

func (r *Resolver) VisitUnaryExpression(expr *ast.UnaryExpression) (any, error) {
	if err := r.ResolveExpression(expr.Right); err != nil {
		return nil, err
	}
	return nil, nil
}

func (r *Resolver) VisitGroupingExpression(expr *ast.GroupingExpression) (any, error) {
	if err := r.ResolveExpression(expr.Expr); err != nil {
		return nil, err
	}
	return nil, nil
}

func (r *Resolver) VisitLiteralExpression(expr *ast.LiteralExpression) (any, error) { return nil, nil }

func (r *Resolver) VisitConditionalExpression(expr *ast.ConditionalExpression) (any, error) {
	if err := r.ResolveExpression(expr.Condition); err != nil {
		return nil, err
	}
	if err := r.ResolveExpression(expr.Consequent); err != nil {
		return nil, err
	}
	if err := r.ResolveExpression(expr.Alternative); err != nil {
		return nil, err
	}
	return nil, nil
}

func (r *Resolver) VisitLogicalExpression(expr *ast.LogicalExpression) (any, error) {
	if err := r.ResolveExpression(expr.Left); err != nil {
		return nil, err
	}
	if err := r.ResolveExpression(expr.Right); err != nil {
		return nil, err
	}
	return nil, nil
}

func (r *Resolver) VisitCallExpression(expr *ast.CallExpression) (any, error) {
	if err := r.ResolveExpression(expr.Calle); err != nil {
		return nil, err
	}
	for _, argument := range expr.Arguments {
		if err := r.ResolveExpression(argument); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (r *Resolver) VisitFunctionExpression(expr *ast.FunctionExpression) (any, error) {
	if err := r.ResolveFunction(expr.Parameters, expr.Body); err != nil {
		return nil, err
	}
	return nil, nil
}
