// Package eval
package eval

// This file holds the evaluator's core: the type itself, its constructors, the
// entry points that drive a program, and the handful of helpers every visit
// method leans on. The visit methods are split to match the two visitor
// interfaces they implement — StmtVisitor in statements.go, ExprVisitor in
// expressions.go — and function values live in function.go.

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"

	"magd/internal/ast"
	"magd/internal/env"
	"magd/internal/errs"
	"magd/internal/token"
)

type Evaluator struct {
	environment *env.Environment
	out         io.Writer
}

// Evaluator implements both visitor interfaces: statements are executed, and
// the expressions they contain are evaluated.
var (
	_ ast.ExprVisitor = (*Evaluator)(nil)
	_ ast.StmtVisitor = (*Evaluator)(nil)
)

func NewEvaluator() *Evaluator {
	return &Evaluator{environment: env.NewEnvironment()}
}

// NewEvaluatorTo is NewEvaluator with the print destination overridden.
// The REPL passes its own buffer. A nil writer means the current os.Stdout
// at print time, so tests that redirect os.Stdout keep working.
func NewEvaluatorTo(w io.Writer) *Evaluator {
	return &Evaluator{environment: env.NewEnvironment(), out: w}
}

// Eval executes a program. Statements produce no values, so there is nothing
// to return — only errors. There is deliberately no recover here: runtime
// errors are returned, so a panic is always a real bug and must crash.
func (e *Evaluator) Eval(stmts []ast.Stmt) error {
	for _, stmt := range stmts {
		if err := e.Execute(stmt); err != nil {
			if _, ok := errors.AsType[errs.ReturnSignal](err); ok {
				return errs.NewRuntimeError(
					0, "",
					"Can't return from top-level code.",
				)
			}
			return err
		}
	}
	return nil
}

// EvalExpression evaluates a single expression and returns its value.
// The REPL uses it to display bare expression statements.
func (e *Evaluator) EvalExpression(expr ast.Expression) (any, error) {
	return e.evaluate(expr)
}

func (e *Evaluator) Execute(stmt ast.Stmt) error {
	_, err := stmt.Accept(e)
	return err
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
	case *Function:
		return "<fun " + v.Declaration.Name.Lexeme + ">"
	default:
		return fmt.Sprintf("%v", v)
	}
}
