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
	Name          string
	Parameters    []token.Token
	Body          ast.Stmt
	Closure       *env.Environment
	Receiver      *Instance // the class binding this function
	isInitializer bool
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
		return f.result(ret.Value), nil
	}

	if err != nil {
		return nil, err
	}
	return f.result(nil), nil
}

// result is what call returns.
// an initializer always returns the instance it was called on
func (f *Function) result(value any) any {
	if f.isInitializer && f.Receiver != nil {
		return f.Receiver
	}
	return value
}

// Bind creates a new environment nested inside the method's closure, and
// declares "this" with the value of the given instance
func (f *Function) Bind(instance *Instance) *Function {
	bound := *f
	bound.Receiver = instance
	bound.Closure = env.NewEnvironmentEnclosing(f.Closure)
	bound.Closure.Define("this", instance)
	return &bound
}

func (f *Function) String() string {
	if f.Name == "" {
		return "<fun>"
	}
	return "<fun " + f.Name + ">"
}
