package parser

// Chapter 12 and 13 coverage: class declarations, inheritance, and super.

import (
	"strings"
	"testing"

	"magd/internal/ast"
	"magd/internal/lexer"
	"magd/internal/token"
)

// parseClass parses source that must be a single class declaration.
func parseClass(t *testing.T, source string) *ast.ClassStmt {
	t.Helper()

	p := NewParser(lexer.NewLexer(source).ScanTokens())
	stmts, _ := p.Parse()
	if errs := p.Errors(); len(errs) != 0 {
		t.Fatalf("parse(%q): unexpected errors: %v", source, errs)
	}

	if len(stmts) != 1 {
		t.Fatalf("parse(%q): want 1 statement, got %d", source, len(stmts))
	}

	class, ok := stmts[0].(*ast.ClassStmt)
	if !ok {
		t.Fatalf("parse(%q): want *ast.ClassStmt, got %T", source, stmts[0])
	}

	return class
}

func TestClassDeclaration(t *testing.T) {
	class := parseClass(t, "class Foo {}")

	if class.Name.Lexeme != "Foo" {
		t.Errorf("class name = %q, want %q", class.Name.Lexeme, "Foo")
	}
	if class.SuperClass != nil {
		t.Errorf("SuperClass = %v, want nil", class.SuperClass)
	}
	if len(class.Methods) != 0 {
		t.Errorf("Methods = %d, want 0", len(class.Methods))
	}
}

func TestClassDeclarationWithSuperclass(t *testing.T) {
	class := parseClass(t, "class Dog < Animal {}")

	if class.SuperClass == nil {
		t.Fatal("SuperClass = nil, want Animal")
	}
	if class.SuperClass.Name.Type != token.IDENT {
		t.Errorf("superclass token type = %v, want IDENT", class.SuperClass.Name.Type)
	}
	if class.SuperClass.Name.Lexeme != "Animal" {
		t.Errorf("superclass = %q, want %q", class.SuperClass.Name.Lexeme, "Animal")
	}
}

func TestClassMethods(t *testing.T) {
	class := parseClass(t, `class Foo {
		bar() {}
		greet(name, greeting) {}
	}`)

	if len(class.Methods) != 2 {
		t.Fatalf("Methods = %d, want 2", len(class.Methods))
	}
	if class.Methods[0].Name.Lexeme != "bar" {
		t.Errorf("first method = %q, want %q", class.Methods[0].Name.Lexeme, "bar")
	}
	if len(class.Methods[1].Parameters) != 2 {
		t.Errorf("greet parameters = %d, want 2", len(class.Methods[1].Parameters))
	}
}

func TestSuperExpressionParses(t *testing.T) {
	expr := parseOneExpr(t, "super.speak()")

	call, ok := expr.(*ast.CallExpression)
	if !ok {
		t.Fatalf("expr = %T, want *ast.CallExpression", expr)
	}

	super, ok := call.Calle.(*ast.SuperExpression)
	if !ok {
		t.Fatalf("callee = %T, want *ast.SuperExpression", call.Calle)
	}

	if super.Keyword.Type != token.SUPER {
		t.Errorf("keyword token type = %v, want SUPER", super.Keyword.Type)
	}
	if super.Method.Lexeme != "speak" {
		t.Errorf("method = %q, want %q", super.Method.Lexeme, "speak")
	}
}

// Classes print back as the source they were written as, superclass and all,
// which means the printer and parser agree on the shape of a class.
func TestClassPrintsAsSource(t *testing.T) {
	tests := []struct {
		source string
		want   string
	}{
		{"class Foo {}", "class Foo {\n}"},
		{"class Dog < Animal {}", "class Dog < Animal {\n}"},
		{
			"class Dog < Animal { speak() { return super.speak; } }",
			"class Dog < Animal {\nspeak() {\nreturn super.speak;\n}\n}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			if got := parseStmt(t, tt.source); got != tt.want {
				t.Errorf("parse(%q) = %q, want %q", tt.source, got, tt.want)
			}
		})
	}
}

func TestClassDeclarationErrors(t *testing.T) {
	tests := []struct {
		source string
		want   string
	}{
		{"class {}", "Expect class name."},
		{"class Dog < {}", "Expect superclass name."},
		{"class Dog Animal {}", "Expect { before class body."},
		{"class Dog { speak() {}", "Expect } after class body."},
		{"class Dog { let x = 1; }", "Expect function name."},
		{"super;", "Expect '.' after 'super'."},
		{"super.;", "Expect Superclass method name."},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			msgs := parseErrorsFor(t, tt.source)

			found := false
			for _, m := range msgs {
				if strings.Contains(m, tt.want) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("parse(%q): errors = %v, want one containing %q", tt.source, msgs, tt.want)
			}
		})
	}
}
