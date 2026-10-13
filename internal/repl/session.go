// Package repl is the interactive loop: one persistent evaluator across
// entries, with bare expression results displayed.
package repl

import (
	"bufio"
	"io"
	"strings"

	"magd/internal/ast"
	"magd/internal/eval"
	"magd/internal/interp"
)

// Session keeps evaluator state across entries. All output — print results,
// displayed expression values, and diagnostics — goes to out.
type Session struct {
	evaluator *eval.Evaluator
	out       io.Writer
}

// NewSession creates a session writing to out.
func NewSession(out io.Writer) *Session {
	return &Session{evaluator: eval.NewEvaluatorTo(out), out: out}
}

// Eval compiles and runs a single entry. Errors are written to the session
// output and never end the session.
func (s *Session) Eval(source string) {
	if strings.TrimSpace(source) == "" {
		return
	}

	stmts, errs := interp.Compile(source)
	if len(errs) > 0 {
		// REPL-only leniency: the file runner requires ";" terminators,
		// but an interactive entry like `1 + 2` should just work. Retry
		// once with a semicolon appended; if that parses, run it.
		// Anything else reports the original errors.
		if trimmed := strings.TrimSpace(source); !strings.HasSuffix(trimmed, ";") {
			if retry, retryErrs := interp.Compile(source + ";"); len(retryErrs) == 0 {
				stmts, errs = retry, nil
			}
		}
		if len(errs) > 0 {
			for _, e := range errs {
				_, _ = io.WriteString(s.out, e.Error()+"\n")
			}
			return
		}
	}

	// A single bare expression statement displays its value (Ch. 8
	// challenge 1); everything else executes as written.
	if len(stmts) == 1 {
		if exprStmt, ok := stmts[0].(*ast.ExpressionStmt); ok {
			value, err := s.evaluator.EvalExpression(exprStmt.Expr)
			if err != nil {
				_, _ = io.WriteString(s.out, err.Error()+"\n")
				return
			}
			_, _ = io.WriteString(s.out, eval.Stringify(value)+"\n")
			return
		}
	}

	if err := s.evaluator.Eval(stmts); err != nil {
		_, _ = io.WriteString(s.out, err.Error()+"\n")
	}
}

// Run reads lines from in until EOF, evaluating each as an entry.
// It reports a read failure rather than silently dropping the tail.
func (s *Session) Run(in io.Reader) error {
	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		s.Eval(scanner.Text())
	}
	return scanner.Err()
}
