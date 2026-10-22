package ast

import (
	"testing"

	"magd/internal/token"
)

func TestPrint(t *testing.T) {
	tests := []struct {
		name string
		expr Expression
		want string
	}{
		{
			name: "number literal",
			expr: &LiteralExpression{Value: float64(123)},
			want: "123",
		},
		{
			name: "string literal",
			expr: &LiteralExpression{Value: "hello"},
			want: "hello",
		},
		{
			name: "boolean literal",
			expr: &LiteralExpression{Value: true},
			want: "true",
		},
		{
			name: "nil literal",
			expr: &LiteralExpression{Value: nil},
			want: "nil",
		},
		{
			name: "grouping",
			expr: &GroupingExpression{
				Expr: &LiteralExpression{Value: float64(1)},
			},
			want: "(group 1)",
		},
		{
			name: "unary negation",
			expr: &UnaryExpression{
				Operator: token.Token{Type: token.MINUS, Lexeme: "-"},
				Right:    &LiteralExpression{Value: float64(123)},
			},
			want: "(- 123)",
		},
		{
			name: "unary not",
			expr: &UnaryExpression{
				Operator: token.Token{Type: token.BANG, Lexeme: "!"},
				Right:    &LiteralExpression{Value: false},
			},
			want: "(! false)",
		},
		{
			name: "binary addition",
			expr: &BinaryExpression{
				Left:     &LiteralExpression{Value: float64(1)},
				Operator: token.Token{Type: token.PLUS, Lexeme: "+"},
				Right:    &LiteralExpression{Value: float64(2)},
			},
			want: "(+ 1 2)",
		},
		{
			name: "book example: precedence is visible",
			expr: &BinaryExpression{
				Left: &UnaryExpression{
					Operator: token.Token{Type: token.MINUS, Lexeme: "-"},
					Right:    &LiteralExpression{Value: float64(123)},
				},
				Operator: token.Token{Type: token.STAR, Lexeme: "*"},
				Right: &GroupingExpression{
					Expr: &LiteralExpression{Value: 45.67},
				},
			},
			want: "(* (- 123) (group 45.67))",
		},
		{
			name: "deeply nested",
			expr: &BinaryExpression{
				Left: &BinaryExpression{
					Left:     &LiteralExpression{Value: float64(1)},
					Operator: token.Token{Type: token.PLUS, Lexeme: "+"},
					Right: &BinaryExpression{
						Left:     &LiteralExpression{Value: float64(2)},
						Operator: token.Token{Type: token.STAR, Lexeme: "*"},
						Right:    &LiteralExpression{Value: float64(3)},
					},
				},
				Operator: token.Token{Type: token.MINUS, Lexeme: "-"},
				Right:    &LiteralExpression{Value: float64(4)},
			},
			want: "(- (+ 1 (* 2 3)) 4)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := (&Printer{}).Print([]Stmt{&ExpressionStmt{Expr: tt.expr}})
			if err != nil {
				t.Fatalf("Print() error = %v", err)
			}
			// Print appends ";\n" per statement, so compare without it.
			if got != tt.want+";\n" {
				t.Errorf("Print() = %q, want %q", got, tt.want+";\n")
			}
		})
	}
}

type recordingVisitor struct {
	called string
}

func (v *recordingVisitor) VisitBlockStmt(stmt *BlockStmt) (any, error) {
	v.called = "block"
	return nil, nil
}

func (v *recordingVisitor) VisitExpressionStmt(stmt *ExpressionStmt) (any, error) {
	v.called = "expressionStmt"
	return nil, nil
}

func (v *recordingVisitor) VisitPrintStmt(stmt *PrintStmt) (any, error) {
	v.called = "printStmt"
	return nil, nil
}

func (v *recordingVisitor) VisitVariableStmt(stmt *VariableStmt) (any, error) {
	v.called = "variableStmt"
	return nil, nil
}

func (v *recordingVisitor) VisitConditionalExpression(expr *ConditionalExpression) (any, error) {
	v.called = "conditional"
	return nil, nil
}

func (v *recordingVisitor) VisitBinaryExpression(expr *BinaryExpression) (any, error) {
	v.called = "binary"
	return nil, nil
}

func (v *recordingVisitor) VisitUnaryExpression(expr *UnaryExpression) (any, error) {
	v.called = "unary"
	return nil, nil
}

func (v *recordingVisitor) VisitGroupingExpression(expr *GroupingExpression) (any, error) {
	v.called = "grouping"
	return nil, nil
}

func (v *recordingVisitor) VisitLiteralExpression(expr *LiteralExpression) (any, error) {
	v.called = "literal"
	return nil, nil
}

func (v *recordingVisitor) VisitVariableExpression(expr *VariableExpression) (any, error) {
	v.called = "variable"
	return nil, nil
}

func (v *recordingVisitor) VisitAssignmentExpression(expr *AssignmentExpression) (any, error) {
	v.called = "assignment"
	return nil, nil
}

func (v *recordingVisitor) VisitLogicalExpression(expr *LogicalExpression) (any, error) {
	v.called = "logical"
	return nil, nil
}

func (v *recordingVisitor) VisitIfStmt(stmt *IfStmt) (any, error) {
	v.called = "if"
	return nil, nil
}

func (v *recordingVisitor) VisitWhileStmt(stmt *WhileStmt) (any, error) {
	v.called = "while"
	return nil, nil
}

func (v *recordingVisitor) VisitBreakStmt(stmt *BreakStmt) (any, error) {
	v.called = "break"
	return nil, nil
}

func (v *recordingVisitor) VisitContinueStmt(stmt *ContinueStmt) (any, error) {
	v.called = "continue"
	return nil, nil
}

func (v *recordingVisitor) VisitForStmt(stmt *ForStmt) (any, error) {
	v.called = "for"
	return nil, nil
}

func (v *recordingVisitor) VisitFunctionStmt(stmt *FunctionStmt) (any, error) {
	v.called = "function"
	return nil, nil
}

func (v *recordingVisitor) VisitCallExpression(expr *CallExpression) (any, error) {
	v.called = "call"
	return nil, nil
}

func (v *recordingVisitor) VisitReturnStmt(stmt *ReturnStmt) (any, error) {
	v.called = "return"
	return nil, nil
}

func (v *recordingVisitor) VisitFunctionExpression(expr *FunctionExpression) (any, error) {
	v.called = "functionExpr"
	return nil, nil
}

func (v *recordingVisitor) VisitClassStmt(stmt *ClassStmt) (any, error) {
	v.called = "classStmt"
	return nil, nil
}

func (v *recordingVisitor) VisitGetExpression(expr *GetExpression) (any, error) {
	v.called = "get"
	return nil, nil
}

func (v *recordingVisitor) VisitSetExpression(expr *SetExpression) (any, error) {
	v.called = "set"
	return nil, nil
}

func (v *recordingVisitor) VisitThisExpression(expr *ThisExpression) (any, error) {
	v.called = "this"
	return nil, nil
}

func TestAcceptDispatchesToMatchingVisitMethod(t *testing.T) {
	tests := []struct {
		name string
		expr Expression
		want string
	}{
		{"binary", &BinaryExpression{}, "binary"},
		{"unary", &UnaryExpression{}, "unary"},
		{"grouping", &GroupingExpression{}, "grouping"},
		{"literal", &LiteralExpression{}, "literal"},
		{"conditional", &ConditionalExpression{}, "conditional"},
		{"variable", &VariableExpression{}, "variable"},
		{"assignment", &AssignmentExpression{}, "assignment"},
		{"logical", &LogicalExpression{}, "logical"},
		{"call", &CallExpression{}, "call"},
		{"functionExpr", &FunctionExpression{}, "functionExpr"},
		{"get", &GetExpression{}, "get"},
		{"set", &SetExpression{}, "set"},
		{"this", &ThisExpression{}, "this"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			visitor := &recordingVisitor{}
			if _, err := tt.expr.Accept(visitor); err != nil {
				t.Fatalf("Accept() error = %v", err)
			}
			if visitor.called != tt.want {
				t.Errorf("Accept() routed to Visit%s, want Visit%s",
					visitor.called, tt.want)
			}
		})
	}
}

// A variable declaration is rendered as source, so the output re-parses.
// The no-initializer case is the one that matters: varDeclaration leaves
// Initializer nil for `let x;`, and a nil Expression would panic on Accept.
func TestPrintVariableStmts(t *testing.T) {
	tests := []struct {
		name string
		stmt Stmt
		want string
	}{
		{
			name: "with initializer",
			stmt: &VariableStmt{
				Name:        token.Token{Type: token.IDENT, Lexeme: "x"},
				Initializer: &LiteralExpression{Value: float64(1)},
			},
			want: "let x = 1;\n",
		},
		{
			name: "no initializer",
			stmt: &VariableStmt{
				Name: token.Token{Type: token.IDENT, Lexeme: "x"},
			},
			want: "let x;\n",
		},
		{
			name: "expression initializer",
			stmt: &VariableStmt{
				Name: token.Token{Type: token.IDENT, Lexeme: "x"},
				Initializer: &BinaryExpression{
					Left:     &LiteralExpression{Value: float64(1)},
					Operator: token.Token{Type: token.PLUS, Lexeme: "+"},
					Right:    &LiteralExpression{Value: float64(2)},
				},
			},
			want: "let x = (+ 1 2);\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := (&Printer{}).Print([]Stmt{tt.stmt})
			if err != nil {
				t.Fatalf("Print() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Print() = %q, want %q", got, tt.want)
			}
		})
	}
}

// A variable reference prints as its bare name, not "(var x)" — a reference
// renders as the same source text that would be re-parsed into it.
func TestPrintVariableExpression(t *testing.T) {
	got, err := (&Printer{}).Print([]Stmt{
		&ExpressionStmt{Expr: &VariableExpression{
			Name: token.Token{Type: token.IDENT, Lexeme: "count"},
		}},
	})
	if err != nil {
		t.Fatalf("Print() error = %v", err)
	}
	if want := "count;\n"; got != want {
		t.Errorf("Print() = %q, want %q", got, want)
	}
}

// An assignment prints as an operator like every other binary-ish node, with
// the name rendered as a bare variable reference on the left.
func TestPrintAssignmentExpression(t *testing.T) {
	name := token.Token{Type: token.IDENT, Lexeme: "x"}

	tests := []struct {
		name string
		expr Expression
		want string
	}{
		{"literal value", &AssignmentExpression{
			Name:  name,
			Value: &LiteralExpression{Value: float64(1)},
		}, "(= x 1);\n"},
		{"expression value", &AssignmentExpression{
			Name: name,
			Value: &BinaryExpression{
				Left:     &LiteralExpression{Value: float64(1)},
				Operator: token.Token{Type: token.PLUS, Lexeme: "+"},
				Right:    &LiteralExpression{Value: float64(2)},
			},
		}, "(= x (+ 1 2));\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := (&Printer{}).Print([]Stmt{&ExpressionStmt{Expr: tt.expr}})
			if err != nil {
				t.Fatalf("Print() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Print() = %q, want %q", got, tt.want)
			}
		})
	}
}

// Statements get the same dispatch coverage as expressions. Without this the
// recording visitor could silently stop satisfying StmtVisitor.
func TestAcceptDispatchesStatements(t *testing.T) {
	tests := []struct {
		name string
		stmt Stmt
		want string
	}{
		{"expressionStmt", &ExpressionStmt{}, "expressionStmt"},
		{"printStmt", &PrintStmt{}, "printStmt"},
		{"variableStmt", &VariableStmt{}, "variableStmt"},
		{"blockStmt", &BlockStmt{}, "block"},
		{"ifStmt", &IfStmt{}, "if"},
		{"whileStmt", &WhileStmt{}, "while"},
		{"breakStmt", &BreakStmt{}, "break"},
		{"continueStmt", &ContinueStmt{}, "continue"},
		{"forStmt", &ForStmt{}, "for"},
		{"functionStmt", &FunctionStmt{}, "function"},
		{"returnStmt", &ReturnStmt{}, "return"},
		{"classStmt", &ClassStmt{}, "classStmt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			visitor := &recordingVisitor{}
			if _, err := tt.stmt.Accept(visitor); err != nil {
				t.Fatalf("Accept() error = %v", err)
			}
			if visitor.called != tt.want {
				t.Errorf("Accept() routed to %q, want %q", visitor.called, tt.want)
			}
		})
	}
}

// A logical operator prints like any other binary, using its own lexeme, so
// precedence between and/or stays visible in the output.
func TestPrintLogicalExpression(t *testing.T) {
	tests := []struct {
		name string
		expr Expression
		want string
	}{
		{"or", &LogicalExpression{
			Left:     &VariableExpression{Name: token.Token{Type: token.IDENT, Lexeme: "a"}},
			Operator: token.Token{Type: token.OR, Lexeme: "or"},
			Right:    &VariableExpression{Name: token.Token{Type: token.IDENT, Lexeme: "b"}},
		}, "(or a b);\n"},
		{"and", &LogicalExpression{
			Left:     &VariableExpression{Name: token.Token{Type: token.IDENT, Lexeme: "a"}},
			Operator: token.Token{Type: token.AND, Lexeme: "and"},
			Right:    &VariableExpression{Name: token.Token{Type: token.IDENT, Lexeme: "b"}},
		}, "(and a b);\n"},
		{"nested shows precedence", &LogicalExpression{
			Left:     &VariableExpression{Name: token.Token{Type: token.IDENT, Lexeme: "a"}},
			Operator: token.Token{Type: token.OR, Lexeme: "or"},
			Right: &LogicalExpression{
				Left:     &VariableExpression{Name: token.Token{Type: token.IDENT, Lexeme: "b"}},
				Operator: token.Token{Type: token.AND, Lexeme: "and"},
				Right:    &VariableExpression{Name: token.Token{Type: token.IDENT, Lexeme: "c"}},
			},
		}, "(or a (and b c));\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := (&Printer{}).Print([]Stmt{&ExpressionStmt{Expr: tt.expr}})
			if err != nil {
				t.Fatalf("Print() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Print() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPrintIfStmt(t *testing.T) {
	cond := &LiteralExpression{Value: true}
	then := &ExpressionStmt{Expr: &LiteralExpression{Value: float64(1)}}
	els := &ExpressionStmt{Expr: &LiteralExpression{Value: float64(2)}}

	tests := []struct {
		name string
		stmt Stmt
		want string
	}{
		{"no else", &IfStmt{Condition: cond, ThenBranch: then}, "if (true) 1;\n"},
		{"with else", &IfStmt{Condition: cond, ThenBranch: then, ElseBranch: els},
			"if (true) 1; else 2;\n"},
		{"compound condition", &IfStmt{
			Condition: &BinaryExpression{
				Left:     &VariableExpression{Name: token.Token{Type: token.IDENT, Lexeme: "x"}},
				Operator: token.Token{Type: token.LESS, Lexeme: "<"},
				Right:    &LiteralExpression{Value: float64(10)},
			},
			ThenBranch: then,
		}, "if ((< x 10)) 1;\n"},
		{"else if chains", &IfStmt{
			Condition:  cond,
			ThenBranch: then,
			ElseBranch: &IfStmt{
				Condition:  &LiteralExpression{Value: false},
				ThenBranch: els,
				ElseBranch: &ExpressionStmt{Expr: &LiteralExpression{Value: float64(3)}},
			},
		}, "if (true) 1; else if (false) 2; else 3;\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := (&Printer{}).Print([]Stmt{tt.stmt})
			if err != nil {
				t.Fatalf("Print() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Print() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPrintWhileStmt(t *testing.T) {
	tests := []struct {
		name string
		stmt Stmt
		want string
	}{
		{"expression body", &WhileStmt{
			Condition: &VariableExpression{Name: token.Token{Type: token.IDENT, Lexeme: "done"}},
			Body:      &ExpressionStmt{Expr: &LiteralExpression{Value: float64(1)}},
		}, "while (done) 1;\n"},
		{"block body", &WhileStmt{
			Condition: &LiteralExpression{Value: true},
			Body: &BlockStmt{Statements: []Stmt{
				&PrintStmt{Expr: &LiteralExpression{Value: float64(2)}},
			}},
		}, "while (true) {\nprint 2;\n}\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := (&Printer{}).Print([]Stmt{tt.stmt})
			if err != nil {
				t.Fatalf("Print() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Print() = %q, want %q", got, tt.want)
			}
		})
	}
}

// A block prints as braces around its statements, one per line.
func TestPrintBlockStmt(t *testing.T) {
	tests := []struct {
		name string
		stmt Stmt
		want string
	}{
		{"empty", &BlockStmt{}, "{\n}\n"},
		{"one statement", &BlockStmt{Statements: []Stmt{
			&ExpressionStmt{Expr: &LiteralExpression{Value: float64(1)}},
		}}, "{\n1;\n}\n"},
		{"print and declaration", &BlockStmt{Statements: []Stmt{
			&PrintStmt{Expr: &LiteralExpression{Value: float64(2)}},
			&VariableStmt{
				Name:        token.Token{Type: token.IDENT, Lexeme: "y"},
				Initializer: &LiteralExpression{Value: float64(3)},
			},
		}}, "{\nprint 2;\nlet y = 3;\n}\n"},
		{"nested", &BlockStmt{Statements: []Stmt{
			&BlockStmt{Statements: []Stmt{
				&ExpressionStmt{Expr: &LiteralExpression{Value: float64(1)}},
			}},
		}}, "{\n{\n1;\n}\n}\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := (&Printer{}).Print([]Stmt{tt.stmt})
			if err != nil {
				t.Fatalf("Print() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Print() = %q, want %q", got, tt.want)
			}
		})
	}
}

// A class prints as braces around its methods, one per line.
//
// A method has no "fun" in front of it: inside a class body the bare
// signature is what a method declaration looks like in source.
func TestPrintClassStmt(t *testing.T) {
	className := token.Token{Type: token.IDENT, Lexeme: "Foo"}

	tests := []struct {
		name string
		stmt Stmt
		want string
	}{
		{"empty class", &ClassStmt{Name: className},
			"class Foo {\n}\n"},
		{"one method", &ClassStmt{
			Name: className,
			Methods: []FunctionStmt{{
				Name: token.Token{Type: token.IDENT, Lexeme: "bar"},
				Body: &BlockStmt{},
			}},
		}, "class Foo {\nbar() {\n}\n}\n"},
		{"method with parameters", &ClassStmt{
			Name: className,
			Methods: []FunctionStmt{{
				Name: token.Token{Type: token.IDENT, Lexeme: "greet"},
				Parameters: []token.Token{
					{Type: token.IDENT, Lexeme: "name"},
					{Type: token.IDENT, Lexeme: "greeting"},
				},
				Body: &BlockStmt{},
			}},
		}, "class Foo {\ngreet(name, greeting) {\n}\n}\n"},
		{"two methods", &ClassStmt{
			Name: className,
			Methods: []FunctionStmt{
				{
					Name: token.Token{Type: token.IDENT, Lexeme: "bar"},
					Body: &BlockStmt{},
				},
				{
					Name: token.Token{Type: token.IDENT, Lexeme: "baz"},
					Body: &BlockStmt{},
				},
			},
		}, "class Foo {\nbar() {\n}\nbaz() {\n}\n}\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := (&Printer{}).Print([]Stmt{tt.stmt})
			if err != nil {
				t.Fatalf("Print() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Print() = %q, want %q", got, tt.want)
			}
		})
	}
}
