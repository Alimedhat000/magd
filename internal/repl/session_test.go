package repl

import (
	"bytes"
	"strings"
	"testing"
)

func TestStatePersistsAcrossEntries(t *testing.T) {
	var out bytes.Buffer
	s := NewSession(&out)

	s.Eval(`let a = 1;`)
	if out.String() != "" {
		t.Fatalf("let printed %q, want nothing", out.String())
	}

	out.Reset()
	s.Eval(`a + 1;`)
	if out.String() != "2\n" {
		t.Errorf("got %q, want %q", out.String(), "2\n")
	}
}

func TestPrintPrints(t *testing.T) {
	var out bytes.Buffer
	s := NewSession(&out)

	s.Eval(`print 1;`)
	if out.String() != "1\n" {
		t.Errorf("got %q, want %q", out.String(), "1\n")
	}
}

func TestBareExpressionDisplays(t *testing.T) {
	var out bytes.Buffer
	s := NewSession(&out)

	s.Eval(`1 + 2;`)
	if out.String() != "3\n" {
		t.Errorf("got %q, want %q", out.String(), "3\n")
	}
}

func TestAssignmentDisplays(t *testing.T) {
	var out bytes.Buffer
	s := NewSession(&out)

	s.Eval(`let x;`)
	s.Eval(`x = 3;`)
	if out.String() != "3\n" {
		t.Errorf("got %q, want %q", out.String(), "3\n")
	}

	out.Reset()
	s.Eval(`x;`)
	if out.String() != "3\n" {
		t.Errorf("after assignment, x = %q, want %q", out.String(), "3\n")
	}
}

func TestRuntimeErrorLeavesSessionWorking(t *testing.T) {
	var out bytes.Buffer
	s := NewSession(&out)

	s.Eval(`print nosuchvar;`)
	if !strings.Contains(out.String(), "Undefined Variable") {
		t.Fatalf("want undefined-variable error, got %q", out.String())
	}

	out.Reset()
	s.Eval(`print 1;`)
	if out.String() != "1\n" {
		t.Errorf("after error, got %q, want %q", out.String(), "1\n")
	}
}

func TestLexerErrorReported(t *testing.T) {
	var out bytes.Buffer
	s := NewSession(&out)

	s.Eval(`@`)
	if !strings.Contains(out.String(), "syntax error") {
		t.Errorf("want a syntax error, got %q", out.String())
	}

	out.Reset()
	s.Eval(`print 1;`)
	if out.String() != "1\n" {
		t.Errorf("after lexer error, got %q, want %q", out.String(), "1\n")
	}
}

func TestBlankLineIsNoOp(t *testing.T) {
	var out bytes.Buffer
	s := NewSession(&out)

	s.Eval(`   `)
	s.Eval(``)
	if out.String() != "" {
		t.Errorf("blank lines printed %q, want nothing", out.String())
	}
}

func TestBareExpressionWithoutSemicolon(t *testing.T) {
	var out bytes.Buffer
	s := NewSession(&out)

	s.Eval(`1 + 2`)
	if out.String() != "3\n" {
		t.Errorf("got %q, want %q", out.String(), "3\n")
	}
}

func TestStatementWithoutSemicolon(t *testing.T) {
	var out bytes.Buffer
	s := NewSession(&out)

	s.Eval(`let b = 4`)
	s.Eval(`print b`)
	if out.String() != "4\n" {
		t.Errorf("got %q, want %q", out.String(), "4\n")
	}
}

func TestGenuineErrorStillReported(t *testing.T) {
	var out bytes.Buffer
	s := NewSession(&out)

	s.Eval(`1 +`)
	if !strings.Contains(out.String(), "syntax error") {
		t.Errorf("want a syntax error, got %q", out.String())
	}
}

func TestRunReadsLines(t *testing.T) {
	var out bytes.Buffer
	s := NewSession(&out)

	if err := s.Run(strings.NewReader("let a = 1;\na + 1;\n")); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if out.String() != "2\n" {
		t.Errorf("got %q, want %q", out.String(), "2\n")
	}
}
