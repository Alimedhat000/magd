package eval

// The ExprVisitor half of the evaluator: one method per expression node.
//
// These run bottom-up — a composite expression evaluates its operands, then
// applies its operator — so evaluation order here is the order operands are
// written in source, except where a node deliberately short-circuits.

import (
	"fmt"

	"magd/internal/ast"
	"magd/internal/errs"
	"magd/internal/token"
)

func (e *Evaluator) VisitVariableExpression(expr *ast.VariableExpression) (any, error) {
	value, err := e.LookUpVariable(expr.Name, expr)
	if err != nil {
		return nil, err
	}

	return value, nil
}

func (e *Evaluator) VisitAssignmentExpression(expr *ast.AssignmentExpression) (any, error) {
	value, err := e.evaluate(expr.Value)
	if err != nil {
		return nil, err
	}

	if distance, ok := e.Locals[expr]; ok {
		_, err = e.environment.AssignAt(distance, expr.Name, value)
	} else {
		_, err = e.globals.Assign(expr.Name, value)
	}
	if err != nil {
		return nil, err
	}

	return value, nil
}

func (e *Evaluator) VisitLiteralExpression(expr *ast.LiteralExpression) (any, error) {
	return expr.Value, nil
}

func (e *Evaluator) VisitGroupingExpression(expr *ast.GroupingExpression) (any, error) {
	return e.evaluate(expr.Expr)
}

func (e *Evaluator) VisitBinaryExpression(expr *ast.BinaryExpression) (any, error) {
	left, err := e.evaluate(expr.Left)
	if err != nil {
		return nil, err
	}

	right, err := e.evaluate(expr.Right)
	if err != nil {
		return nil, err
	}

	switch expr.Operator.Type {
	case token.GREATER:
		if err := checkNumberOperands(expr.Operator, left, right); err != nil {
			return nil, err
		}
		return left.(float64) > right.(float64), nil

	case token.GREATEREQUAL:
		if err := checkNumberOperands(expr.Operator, left, right); err != nil {
			return nil, err
		}
		return left.(float64) >= right.(float64), nil

	case token.LESS:
		if err := checkNumberOperands(expr.Operator, left, right); err != nil {
			return nil, err
		}
		return left.(float64) < right.(float64), nil

	case token.LESSEQUAL:
		if err := checkNumberOperands(expr.Operator, left, right); err != nil {
			return nil, err
		}
		return left.(float64) <= right.(float64), nil

	case token.BANGEQUAL:
		// Equality accepts any pair of types, including mixed ones.
		return !e.isEqual(left, right), nil

	case token.EQUALEQUAL:
		return e.isEqual(left, right), nil

	case token.MINUS:
		if err := checkNumberOperands(expr.Operator, left, right); err != nil {
			return nil, err
		}
		return left.(float64) - right.(float64), nil

	case token.STAR:
		if err := checkNumberOperands(expr.Operator, left, right); err != nil {
			return nil, err
		}
		return left.(float64) * right.(float64), nil

	case token.PLUS:
		// + is overloaded: numbers add, strings concatenate. Checked rather
		// than asserted, since neither type is guaranteed.
		if l, ok := left.(float64); ok {
			if r, ok := right.(float64); ok {
				return l + r, nil
			}
		}

		// hanlde string concate
		if l, ok := left.(string); ok {
			if r, ok := right.(string); ok {
				return l + r, nil
			}
		}

		return nil, errs.NewRuntimeError(
			expr.Operator.Line, expr.Operator.Lexeme,
			"Operands must be two numbers or two strings.",
		)

	case token.SLASH:
		if err := checkNumberOperands(expr.Operator, left, right); err != nil {
			return nil, err
		}

		// right is an `any`, so compare it to a float64 zero. Writing
		// `right == 0` boxes 0 as an int, which never matches a float64.
		if right == float64(0) {
			return nil, errs.NewRuntimeError(
				expr.Operator.Line, expr.Operator.Lexeme,
				"Cannot divide by zero.",
			)
		}

		return left.(float64) / right.(float64), nil
	}

	return nil, fmt.Errorf("unknown operator %q", expr.Operator.Lexeme)
}

func (e *Evaluator) VisitUnaryExpression(expr *ast.UnaryExpression) (any, error) {
	right, err := e.evaluate(expr.Right)
	if err != nil {
		return nil, err
	}

	switch expr.Operator.Type {
	case token.BANG:
		return !e.isTruthy(right), nil
	case token.MINUS:
		if err := checkNumberOperand(expr.Operator, right); err != nil {
			return nil, err
		}
		return -right.(float64), nil
	}

	return nil, nil
}

func (e *Evaluator) VisitConditionalExpression(expr *ast.ConditionalExpression) (any, error) {
	condition, err := e.evaluate(expr.Condition)
	if err != nil {
		return nil, err
	}

	if e.isTruthy(condition) {
		return e.evaluate(expr.Consequent)
	}
	return e.evaluate(expr.Alternative)
}

func (e *Evaluator) VisitLogicalExpression(expr *ast.LogicalExpression) (any, error) {
	left, err := e.evaluate(expr.Left)
	if err != nil {
		return nil, err
	}

	// handle fallthrough, or won't evaluate the second condn if the first is true
	// and and won't evaluate it if the first is false
	if expr.Operator.Type == token.OR {
		if e.isTruthy(left) {
			return left, nil
		}
	} else if !e.isTruthy(left) {
		return left, nil
	}

	return e.evaluate(expr.Right)
}

// VisitFunctionExpression returns An anonymous function that is the same as a declaration, minus the
// binding: it produces a function and names nothing.
func (e *Evaluator) VisitFunctionExpression(expr *ast.FunctionExpression) (any, error) {
	return &Function{
		Parameters: expr.Parameters,
		Body:       expr.Body,
		Closure:    e.environment,
	}, nil
}

func (e *Evaluator) VisitCallExpression(expr *ast.CallExpression) (any, error) {
	calle, err := e.evaluate(expr.Calle)
	if err != nil {
		return nil, err
	}

	var args []any
	for _, arguemnt := range expr.Arguments {

		arg, err := e.evaluate(arguemnt)
		if err != nil {
			return nil, err
		}

		args = append(args, arg)
	}

	function, ok := calle.(Callable)
	if !ok {
		return nil, errs.NewRuntimeError(
			expr.Parenthesis.Line, expr.Parenthesis.Lexeme,
			"Can only call functions.",
		)
	}

	if function.Arity() != len(args) {
		return nil, errs.NewRuntimeError(
			expr.Parenthesis.Line, expr.Parenthesis.Lexeme,
			fmt.Sprintf("Expected %d arguments but got %d.", function.Arity(), len(args)),
		)
	}

	return function.Call(e, args)
}
