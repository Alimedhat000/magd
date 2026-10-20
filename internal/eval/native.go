package eval

// Native functions are implemented in Go and handed to magd code as values.

import "time"

type NativeFunction struct {
	Name    string
	NumArgs int
	Fn      func(e *Evaluator, args []any) (any, error)
}

func (f *NativeFunction) Arity() int { return f.NumArgs }

func (f *NativeFunction) Call(e *Evaluator, args []any) (any, error) {
	return f.Fn(e, args)
}

// clock returns the number of seconds since the Unix epoch.
// The seconds are a float so a program can measure an interval shorter than
// one second.
func clock(_ *Evaluator, _ []any) (any, error) {
	now := time.Now()
	return float64(now.Unix()) + float64(now.Nanosecond())/1e9, nil
}
