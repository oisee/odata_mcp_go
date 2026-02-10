package lexer

import (
	"testing"
)

func TestNextToken_Simple(t *testing.T) {
	input := `SELECT * FROM Orders`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{SELECT, "SELECT"},
		{MULTIPLY, "*"},
		{FROM, "FROM"},
		{IDENT, "Orders"},
		{EOF, ""},
	}

	l := New(input)
	for i, tt := range tests {
		tok := l.NextToken()
		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}
		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestNextToken_Query(t *testing.T) {
	input := `SELECT po.ID, po.Vendor
FROM MM.PurchaseOrders AS po
WHERE po.Status == 'OPEN' AND po.Amount > 1000
ORDER BY po.Amount DESC
LIMIT 50`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{SELECT, "SELECT"},
		{IDENT, "po"},
		{DOT, "."},
		{IDENT, "ID"},
		{COMMA, ","},
		{IDENT, "po"},
		{DOT, "."},
		{IDENT, "Vendor"},
		{FROM, "FROM"},
		{IDENT, "MM"},
		{DOT, "."},
		{IDENT, "PurchaseOrders"},
		{AS, "AS"},
		{IDENT, "po"},
		{WHERE, "WHERE"},
		{IDENT, "po"},
		{DOT, "."},
		{IDENT, "Status"},
		{EQ, "=="},
		{STRING, "OPEN"},
		{AND, "AND"},
		{IDENT, "po"},
		{DOT, "."},
		{IDENT, "Amount"},
		{GT, ">"},
		{NUMBER, "1000"},
		{ORDER, "ORDER"},
		{BY, "BY"},
		{IDENT, "po"},
		{DOT, "."},
		{IDENT, "Amount"},
		{DESC, "DESC"},
		{LIMIT, "LIMIT"},
		{NUMBER, "50"},
		{EOF, ""},
	}

	l := New(input)
	for i, tt := range tests {
		tok := l.NextToken()
		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%s, got=%s (literal=%q)",
				i, tt.expectedType, tok.Type, tok.Literal)
		}
		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestNextToken_Variables(t *testing.T) {
	input := `$orderId $newDate $result`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{VARIABLE, "$orderId"},
		{VARIABLE, "$newDate"},
		{VARIABLE, "$result"},
		{EOF, ""},
	}

	l := New(input)
	for i, tt := range tests {
		tok := l.NextToken()
		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%s, got=%s",
				i, tt.expectedType, tok.Type)
		}
		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestNextToken_Workflow(t *testing.T) {
	input := `WORKFLOW UpdatePO($poNumber, $newDate)
  STEP read_po:
    READ PurchaseOrders WHERE PONumber == $poNumber $po
    ASSERT $po != NULL
  STEP update:
    UPDATE PurchaseOrders SET DeliveryDate = $newDate
    FALLBACK => ROLLBACK`

	l := New(input)

	// Just verify no errors and correct keyword recognition
	expectedKeywords := []TokenType{
		WORKFLOW, IDENT, LPAREN, VARIABLE, COMMA, VARIABLE, RPAREN,
		STEP, IDENT, COLON,
		READ, IDENT, WHERE, IDENT, EQ, VARIABLE, VARIABLE,
		ASSERT, VARIABLE, NEQ, NULL,
		STEP, IDENT, COLON,
		UPDATE, IDENT, SET, IDENT, ASSIGN, VARIABLE,
		FALLBACK, FATARROW, ROLLBACK,
		EOF,
	}

	for i, expected := range expectedKeywords {
		tok := l.NextToken()
		if tok.Type != expected {
			t.Errorf("token[%d] wrong type. expected=%s, got=%s (literal=%q)",
				i, expected, tok.Type, tok.Literal)
		}
	}
}

func TestNextToken_Comments(t *testing.T) {
	input := `SELECT * FROM Orders -- this is a comment
WHERE status == 'OPEN' // another comment
/* block comment */ LIMIT 10`

	l := New(input)

	expected := []TokenType{
		SELECT, MULTIPLY, FROM, IDENT,
		WHERE, IDENT, EQ, STRING,
		LIMIT, NUMBER,
		EOF,
	}

	for i, exp := range expected {
		tok := l.NextToken()
		if tok.Type != exp {
			t.Errorf("token[%d] wrong. expected=%s, got=%s (lit=%q)",
				i, exp, tok.Type, tok.Literal)
		}
	}
}

func TestNextToken_Operators(t *testing.T) {
	input := `== != < > <= >= && || + - * / -> =>`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{EQ, "=="},
		{NEQ, "!="},
		{LT, "<"},
		{GT, ">"},
		{LTE, "<="},
		{GTE, ">="},
		{AND, "&&"},
		{OR, "||"},
		{PLUS, "+"},
		{MINUS, "-"},
		{MULTIPLY, "*"},
		{DIVIDE, "/"},
		{ARROW, "->"},
		{FATARROW, "=>"},
		{EOF, ""},
	}

	l := New(input)
	for i, tt := range tests {
		tok := l.NextToken()
		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%s, got=%s",
				i, tt.expectedType, tok.Type)
		}
		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestNextToken_Numbers(t *testing.T) {
	input := `123 45.67 0 100.00`

	tests := []struct {
		expectedLiteral string
	}{
		{"123"},
		{"45.67"},
		{"0"},
		{"100.00"},
	}

	l := New(input)
	for i, tt := range tests {
		tok := l.NextToken()
		if tok.Type != NUMBER {
			t.Fatalf("tests[%d] - expected NUMBER, got %s", i, tok.Type)
		}
		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestNextToken_LineNumbers(t *testing.T) {
	input := `SELECT
FROM
WHERE`

	l := New(input)

	tok := l.NextToken() // SELECT
	if tok.Line != 1 {
		t.Errorf("SELECT should be on line 1, got %d", tok.Line)
	}

	tok = l.NextToken() // FROM
	if tok.Line != 2 {
		t.Errorf("FROM should be on line 2, got %d", tok.Line)
	}

	tok = l.NextToken() // WHERE
	if tok.Line != 3 {
		t.Errorf("WHERE should be on line 3, got %d", tok.Line)
	}
}
