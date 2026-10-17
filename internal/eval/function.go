package eval

import (
	"errors"

	"magd/internal/ast"
	"magd/internal/env"
	"magd/internal/errs"
	"magd/internal/token"
)

type Callable interface {
	Arity() int
	Call(e *Evaluator, args []any) (any, error)
}

// Function is the runtime value for both a named declaration and an
// anonymous expression. Name is empty for the latter; Parameters and Body are
// all a call actually needs, which is why neither node type is stored here.
type Function struct {
	Name       string
	Parameters []token.Token
	Body       ast.Stmt
	Closure    *env.Environment
}

func (f *Function) Arity() int { return len(f.Parameters) }

func (f *Function) Call(e *Evaluator, args []any) (any, error) {
	previous := e.environment
	e.environment = env.NewEnvironmentEnclosing(f.Closure)
	defer func() { e.environment = previous }()

	for i, param := range f.Parameters {
		e.environment.Define(param.Lexeme, args[i])
	}

	err := e.Execute(f.Body)

	var ret errs.ReturnSignal
	if errors.As(err, &ret) {
		return ret.Value, nil
	}
	return nil, err
}
