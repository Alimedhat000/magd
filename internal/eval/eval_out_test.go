package eval

import (
	"bytes"
	"testing"

	"magd/internal/lexer"
	"magd/internal/parser"
)

func evalToBuf(t *testing.T, buf *bytes.Buffer, source string) error {
	t.Helper()

	p := parser.NewParser(lexer.NewLexer(source).ScanTokens())
	stmts, _ := p.Parse()
	if errs := p.Errors(); len(errs) != 0 {
		t.Fatalf("parse(%q): unexpected errors: %v", source, errs)
	}

	return NewEvaluatorTo(buf).Eval(stmts)
}

func TestPrintGoesToWriter(t *testing.T) {
	var buf bytes.Buffer
	if err := evalToBuf(t, &buf, `print 1;`); err != nil {
		t.Fatalf("Eval() error = %v", err)
	}
	if buf.String() != "1\n" {
		t.Errorf("printed %q, want %q", buf.String(), "1\n")
	}
}

func TestEvalExpressionReturnsValue(t *testing.T) {
	tests := []struct {
		source string
		want   any
	}{
		{"1 + 2", float64(3)},
		{`"a" + "b"`, "ab"},
		{"true", true},
		{"nil", nil},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			expr := parseOneExpression(t, tt.source)
			got, err := NewEvaluator().EvalExpression(expr)
			if err != nil {
				t.Fatalf("EvalExpression(%q) error = %v", tt.source, err)
			}
			if got != tt.want {
				t.Errorf("EvalExpression(%q) = %v (%T), want %v (%T)",
					tt.source, got, got, tt.want, tt.want)
			}
		})
	}
}

func TestEvalExpressionReturnsError(t *testing.T) {
	expr := parseOneExpression(t, "1 / 0")
	if _, err := NewEvaluator().EvalExpression(expr); err == nil {
		t.Fatal("want a runtime error from 1 / 0, got none")
	}
}
