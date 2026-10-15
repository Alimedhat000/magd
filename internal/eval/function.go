package eval

import (
	"errors"

	"magd/internal/ast"
	"magd/internal/env"
	"magd/internal/errs"
)

type Callable interface {
	Arity() int
	Call(e *Evaluator, args []any) (any, error)
}

type Function struct {
	Declaration *ast.FunctionStmt
	Closure     *env.Environment
}

func (f *Function) Arity() int { return len(f.Declaration.Parameters) }

func (f *Function) Call(e *Evaluator, args []any) (any, error) {
	previous := e.environment
	e.environment = env.NewEnvironmentEnclosing(f.Closure)
	defer func() { e.environment = previous }()

	for i, param := range f.Declaration.Parameters {
		e.environment.Define(param.Lexeme, args[i])
	}

	err := e.Execute(f.Declaration.Body)

	var ret errs.ReturnSignal
	if errors.As(err, &ret) {
		return ret.Value, nil
	}
	return nil, err
}
