// Package eval
package eval

import (
	"fmt"
	"reflect"
	"strconv"

	"magd/internal/ast"
	"magd/internal/errs"
	"magd/internal/token"
)

type Evaluator struct{}

// Eval evaluates expr and returns the resulting value.
//
// A runtime error deep inside the tree — say, 2 * (3 / -"muffin") — panics on
// its way out of the recursive visits. Eval recovers it here and returns it as
// an error, so the REPL stays alive and the user can type another line.
func (e *Evaluator) Eval(expr ast.Expression) (result any, err error) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}

		perr, ok := r.(error)
		if !ok {
			panic(r) // a real bug, not a runtime error
		}

		result, err = nil, perr
	}()

	return e.evaluate(expr)
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

func (e *Evaluator) isTruthy(value any) bool {
	if value == nil {
		return false
	}

	if v, ok := value.(bool); ok {
		return v
	}
	return true
}

func (e *Evaluator) isEqual(left any, right any) bool {
	if left == nil && right == nil {
		return true
	}
	if left == nil || right == nil {
		return false
	}

	return reflect.DeepEqual(left, right)
}

// checkNumberOperand reports an error unless operand is a number.
func checkNumberOperand(operator token.Token, operand any) error {
	if _, ok := operand.(float64); ok {
		return nil
	}
	return errs.NewRuntimeError(operator.Line, operator.Lexeme, "Operand must be a number.")
}

// checkNumberOperands reports an error unless both operands are numbers.
//
// Callers must evaluate both operands before calling this.
func checkNumberOperands(operator token.Token, left, right any) error {
	if err := checkNumberOperand(operator, left); err != nil {
		return err
	}
	return checkNumberOperand(operator, right)
}

func (e *Evaluator) evaluate(expr ast.Expression) (any, error) {
	return expr.Accept(e)
}

// Stringify renders a runtime value the way a Lox user expects to see it.
//
// It exists because two representations differ between Go and Lox. nil is Go's
// null but Lox calls it "nil", and Lox has no integer type — every number is a
// float64, so 123 would otherwise print as "123.0".
func Stringify(value any) string {
	switch v := value.(type) {
	case nil:
		return "nil"
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case string:
		return v
	case bool:
		return strconv.FormatBool(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}
