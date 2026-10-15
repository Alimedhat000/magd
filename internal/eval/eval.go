// Package eval
package eval

import (
	"errors"
	"fmt"
	"io"
	"os"
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
			var ret errs.ReturnSignal
			if errors.As(err, &ret) {
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

		switch err := e.Execute(stmt.Body); err.(type) {
		case errs.BreakSignal:
			return nil, nil
		case errs.ContinueSignal:
			// falls through on purpose: the increment must still run
		case nil:
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

		switch err.(type) {
		case errs.BreakSignal:
			return nil, nil

		case errs.ContinueSignal:
			continue
		case nil:
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

func (e *Evaluator) VisitVariableExpression(expr *ast.VariableExpression) (any, error) {
	value, err := e.environment.Get(expr.Name)
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

	_, err = e.environment.Assign(expr.Name, value)
	if err != nil {
		return nil, err
	}
	return value, nil
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

func (e *Evaluator) VisitFunctionStmt(stmt *ast.FunctionStmt) (any, error) {
	function := &Function{Declaration: stmt, Closure: e.environment}

	e.environment.Define(stmt.Name.Lexeme, function)

	return nil, nil
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
