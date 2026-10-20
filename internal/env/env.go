// Package env
package env

// TODO: make redifining return an error instead of re assign

import (
	"fmt"

	"magd/internal/errs"
	"magd/internal/token"
)

type Environment struct {
	values    map[string]any
	enclosing *Environment // a linked list of scopes
}

func NewEnvironment() *Environment {
	return &Environment{values: map[string]any{}}
}

func NewEnvironmentEnclosing(enclosing *Environment) *Environment {
	return &Environment{values: map[string]any{}, enclosing: enclosing}
}

// Define declares a new variable in the current scope and creates it
func (e *Environment) Define(name string, value any) {
	e.values[name] = value
}

// Get searches for a variable name, recursing through the upper scopes,
// and errors if it's not found
func (e *Environment) Get(name token.Token) (any, error) {
	if v, ok := e.values[name.Lexeme]; ok {
		return v, nil
	}

	// Not in this scope, so search outward.
	if e.enclosing != nil {
		return e.enclosing.Get(name)
	}

	return nil, errs.NewRuntimeError(
		name.Line, name.Lexeme,
		"Undefined Variable '"+name.Lexeme+"'.",
	)
}

// Assign writes to an existing name, recursing through the upper scopes,
// and errors if it's not found
func (e *Environment) Assign(name token.Token, value any) (any, error) {
	if _, ok := e.values[name.Lexeme]; ok {
		e.values[name.Lexeme] = value
		return value, nil
	}

	// Not in this scope, so search outward.
	if e.enclosing != nil {
		return e.enclosing.Assign(name, value)
	}

	return nil, errs.NewRuntimeError(
		name.Line, name.Lexeme,
		"Undefined Variable '"+name.Lexeme+"'.",
	)
}

func (e *Environment) AssignAt(distance int, name token.Token, value any) (any, error) {
	env, err := e.Ancestor(distance)
	if err != nil {
		return nil, err
	}
	if _, ok := env.values[name.Lexeme]; !ok {
		return nil, errs.NewRuntimeError(
			name.Line, name.Lexeme,
			"Undefined Variable '"+name.Lexeme+"'.")
	}
	env.values[name.Lexeme] = value
	return value, nil
}

func (e *Environment) GetAt(distance int, name token.Token) (any, error) {
	env, err := e.Ancestor(distance)
	if err != nil {
		return nil, err
	}

	if value, ok := env.values[name.Lexeme]; ok {
		return value, nil
	}

	return nil, errs.NewRuntimeError(
		name.Line, name.Lexeme,
		"Undefined Variable '"+name.Lexeme+"'.",
	)
}

// Ancestor walks a fixed number of hops up the parent chain and returns the Environment
// there
func (e *Environment) Ancestor(distance int) (*Environment, error) {
	env := e
	for range distance {
		if env.enclosing == nil {
			return nil, fmt.Errorf(
				"environment: walked %d past the root scope", distance)
		}
		env = env.enclosing
	}
	return env, nil
}
