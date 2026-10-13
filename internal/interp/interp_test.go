package interp

import "testing"

func TestCompile(t *testing.T) {
	tests := []struct {
		name      string
		source    string
		wantStmts int
		wantErrs  bool
	}{
		{"print", `print 1;`, 1, false},
		{"let", `let x = 1;`, 1, false},
		{"expression", `1 + 2;`, 1, false},
		{"block", `{ print 1; }`, 1, false},
		{"lexer error", `@`, 0, true},
		{"unterminated string", `"abc`, 0, true},
		{"parser error", `1 +`, 0, true},
		{"empty", ``, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stmts, errs := Compile(tt.source)
			if tt.wantErrs && len(errs) == 0 {
				t.Errorf("Compile(%q): want errors, got none", tt.source)
			}
			if !tt.wantErrs && len(errs) != 0 {
				t.Errorf("Compile(%q): unexpected errors: %v", tt.source, errs)
			}
			if len(stmts) != tt.wantStmts {
				t.Errorf("Compile(%q) = %d statements, want %d", tt.source, len(stmts), tt.wantStmts)
			}
		})
	}
}

// Lexer errors must short-circuit the parser: no statements come back.
func TestCompileLexerErrorSkipsParse(t *testing.T) {
	stmts, errs := Compile(`let x = 1; @`)
	if len(errs) == 0 {
		t.Fatal("want a lexer error, got none")
	}
	if stmts != nil {
		t.Errorf("want nil statements on lexer error, got %d", len(stmts))
	}
}
