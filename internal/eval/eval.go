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
	"magd/internal/resolver"
	"magd/internal/token"
)

type Evaluator struct {
	globals     *env.Environment
	environment *env.Environment
	Locals      map[ast.Expression]int
	out         io.Writer
}

// Evaluator implements both visitor interfaces: statements are executed, and
// the expressions they contain are evaluated.
var (
	_ ast.ExprVisitor = (*Evaluator)(nil)
	_ ast.StmtVisitor = (*Evaluator)(nil)
)

func NewEvaluator() *Evaluator {
	return newEvaluator(nil)
}

// NewEvaluatorTo is NewEvaluator with the print destination overridden.
// The REPL passes its own buffer. A nil writer means the current os.Stdout
// at print time, so tests that redirect os.Stdout keep working.
func NewEvaluatorTo(w io.Writer) *Evaluator {
	return newEvaluator(w)
}

// newEvaluator builds an evaluator over a fresh global scope.
func newEvaluator(w io.Writer) *Evaluator {
	globals := env.NewEnvironment()

	e := &Evaluator{globals: globals, environment: globals, out: w}
	e.defineNatives()

	return e
}

// defineNatives puts the interpreter's built-in functions in the global scope.
// They live there like any other global, so Lox code can call them by name
// and can shadow them with a binding of its own.
func (e *Evaluator) defineNatives() {
	e.globals.Define("clock", &NativeFunction{
		Name:    "clock",
		NumArgs: 0,
		Fn:      clock,
	})
}

// Eval executes a program. Statements produce no values, so there is nothing
// to return — only errors. There is deliberately no recover here: runtime
// errors are returned, so a panic is always a real bug and must crash.
func (e *Evaluator) Eval(stmts []ast.Stmt) error {
	if err := e.resolve(stmts); err != nil {
		return err
	}

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
	if err := e.resolveExpression(expr); err != nil {
		return nil, err
	}

	return e.evaluate(expr)
}

// resolve runs the resolver over the statements, recording the scope
// distances it finds. Each expression that names a local records how many
// scopes up that local is declared.
//
// The table lives on the evaluator and is kept between calls: the REPL
// evaluates one entry at a time, and a function defined in one entry is
// called in another.
func (e *Evaluator) resolve(stmts []ast.Stmt) error {
	if e.Locals == nil {
		e.Locals = map[ast.Expression]int{}
	}

	return resolver.New(e.Locals).ResolveStatements(stmts)
}

// resolveExpression is resolve for a single expression, which is what the
// REPL evaluates when an entry is a bare expression.
func (e *Evaluator) resolveExpression(expr ast.Expression) error {
	if e.Locals == nil {
		e.Locals = map[ast.Expression]int{}
	}

	return resolver.New(e.Locals).ResolveExpression(expr)
}

func (e *Evaluator) Execute(stmt ast.Stmt) error {
	_, err := stmt.Accept(e)
	return err
}

// LookUpVariable reads a name. The resolver records how many scopes up the
// binding lives; without an entry the name is global, so it is read from the
// global scope.
func (e *Evaluator) LookUpVariable(name token.Token, expr ast.Expression) (any, error) {
	if distance, ok := e.Locals[expr]; ok {
		return e.environment.GetAt(distance, name)
	}
	return e.globals.Get(name)
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

	// make sure both are comparable, so instances, classes, and functions are only equal
	// when they are the same object.
	if reflect.TypeOf(left).Comparable() && reflect.TypeOf(right).Comparable() {
		return left == right
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
