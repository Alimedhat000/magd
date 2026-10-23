package eval

import (
	"magd/internal/errs"
	"magd/internal/token"
)

type Instance struct {
	class  *Class
	fields map[string]any
}

func NewInstance(class *Class) *Instance {
	return &Instance{
		class:  class,
		fields: map[string]any{},
	}
}

func (i *Instance) Get(name token.Token) (any, error) {
	if val, ok := i.fields[name.Lexeme]; ok {
		return val, nil
	}

	if method := i.class.FindMethod(name.Lexeme); method != nil {
		return method.Bind(i), nil
	}

	return nil, errs.NewRuntimeError(name.Line, name.Lexeme, "Undefined property '"+name.Lexeme+"'.")
}

func (i *Instance) Set(name token.Token, value any) (any, error) {
	i.fields[name.Lexeme] = value
	return value, nil
}

func (i *Instance) String() string { return i.class.Name + " instance" }

type Class struct {
	Name       string
	Methods    map[string]*Function
	SuperClass *Class
}

func (c *Class) Arity() int {
	initializer := c.FindMethod("init")
	if initializer == nil {
		return 0
	}
	return initializer.Arity()
}

func (c *Class) Call(e *Evaluator, args []any) (any, error) {
	instance := NewInstance(c)
	if initializer := c.FindMethod("init"); initializer != nil {
		_, err := initializer.Bind(instance).Call(e, args) // bind to the instance and call it
		if err != nil {
			return nil, err
		}
	}

	return instance, nil
}

func (c *Class) FindMethod(name string) *Function {
	if m := c.Methods[name]; m != nil {
		return m
	}

	if c.SuperClass != nil {
		return c.SuperClass.FindMethod(name)
	}
	return nil
}

func (c *Class) String() string { return c.Name + " Class" }
