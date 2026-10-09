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
			got, err := (&Printer{}).Print(tt.expr)
			if err != nil {
				t.Fatalf("Print() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Print() = %q, want %q", got, tt.want)
			}
		})
	}
}

type recordingVisitor struct {
	called string
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
