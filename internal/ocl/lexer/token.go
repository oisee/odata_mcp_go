// Package lexer provides tokenization for OData Composer Language (OCL)
package lexer

// TokenType represents the type of a lexical token
type TokenType int

const (
	// Special tokens
	ILLEGAL TokenType = iota
	EOF
	COMMENT

	// Literals
	IDENT     // field names, service names, entity names
	STRING    // "string" or 'string'
	NUMBER    // 123, 123.45
	VARIABLE  // $variable

	// Operators
	ASSIGN    // =
	EQ        // ==
	NEQ       // !=
	LT        // <
	GT        // >
	LTE       // <=
	GTE       // >=
	AND       // AND, &&
	OR        // OR, ||
	NOT       // NOT, !
	PLUS      // +
	MINUS     // -
	MULTIPLY  // *
	DIVIDE    // /
	DOT       // .
	ARROW     // ->
	FATARROW  // =>

	// Delimiters
	COMMA     // ,
	COLON     // :
	SEMICOLON // ;
	LPAREN    // (
	RPAREN    // )
	LBRACE    // {
	RBRACE    // }
	LBRACKET  // [
	RBRACKET  // ]
	PIPE      // |

	// Keywords - Query
	FROM
	SELECT
	WHERE
	JOIN
	ON
	AS
	ORDER
	BY
	ASC
	DESC
	LIMIT
	OFFSET
	GROUP
	HAVING

	// Keywords - Aggregation
	COUNT
	SUM
	AVG
	MIN
	MAX

	// Keywords - Workflow
	WORKFLOW
	STEP
	CREATE
	UPDATE
	DELETE
	READ
	CALL

	// Keywords - Control Flow
	IF
	THEN
	ELSE
	ENDIF
	FOREACH
	IN
	ENDFOR
	RETRY
	TIMES
	DELAY

	// Keywords - Assertions & Fallbacks
	ASSERT
	EXPECT
	FALLBACK
	ROLLBACK
	THROW

	// Keywords - Transactions
	BEGIN
	COMMIT
	TRANSACTION

	// Keywords - Types
	TRUE
	FALSE
	NULL

	// Keywords - Other
	WITH
	SET
	TO
	USING
	RETURNS
	SERVICE
	EXPOSE
)

// Token represents a lexical token
type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Column  int
}

var keywords = map[string]TokenType{
	// Query keywords
	"FROM":   FROM,
	"SELECT": SELECT,
	"WHERE":  WHERE,
	"JOIN":   JOIN,
	"ON":     ON,
	"AS":     AS,
	"ORDER":  ORDER,
	"BY":     BY,
	"ASC":    ASC,
	"DESC":   DESC,
	"LIMIT":  LIMIT,
	"OFFSET": OFFSET,
	"GROUP":  GROUP,
	"HAVING": HAVING,

	// Aggregation keywords
	"COUNT": COUNT,
	"SUM":   SUM,
	"AVG":   AVG,
	"MIN":   MIN,
	"MAX":   MAX,

	// Workflow keywords
	"WORKFLOW": WORKFLOW,
	"STEP":     STEP,
	"CREATE":   CREATE,
	"UPDATE":   UPDATE,
	"DELETE":   DELETE,
	"READ":     READ,
	"CALL":     CALL,

	// Control flow keywords
	"IF":      IF,
	"THEN":    THEN,
	"ELSE":    ELSE,
	"ENDIF":   ENDIF,
	"FOREACH": FOREACH,
	"IN":      IN,
	"ENDFOR":  ENDFOR,
	"RETRY":   RETRY,
	"TIMES":   TIMES,
	"DELAY":   DELAY,

	// Assertion keywords
	"ASSERT":   ASSERT,
	"EXPECT":   EXPECT,
	"FALLBACK": FALLBACK,
	"ROLLBACK": ROLLBACK,
	"THROW":    THROW,

	// Transaction keywords
	"BEGIN":       BEGIN,
	"COMMIT":      COMMIT,
	"TRANSACTION": TRANSACTION,

	// Boolean/null keywords
	"TRUE":  TRUE,
	"FALSE": FALSE,
	"NULL":  NULL,

	// Other keywords
	"WITH":    WITH,
	"SET":     SET,
	"TO":      TO,
	"USING":   USING,
	"RETURNS": RETURNS,
	"SERVICE": SERVICE,
	"EXPOSE":  EXPOSE,

	// Logical operators as keywords
	"AND": AND,
	"OR":  OR,
	"NOT": NOT,
}

// LookupIdent checks if an identifier is a keyword
func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}

// String returns a string representation of the token type
func (t TokenType) String() string {
	names := map[TokenType]string{
		ILLEGAL:     "ILLEGAL",
		EOF:         "EOF",
		COMMENT:     "COMMENT",
		IDENT:       "IDENT",
		STRING:      "STRING",
		NUMBER:      "NUMBER",
		VARIABLE:    "VARIABLE",
		ASSIGN:      "=",
		EQ:          "==",
		NEQ:         "!=",
		LT:          "<",
		GT:          ">",
		LTE:         "<=",
		GTE:         ">=",
		AND:         "AND",
		OR:          "OR",
		NOT:         "NOT",
		PLUS:        "+",
		MINUS:       "-",
		MULTIPLY:    "*",
		DIVIDE:      "/",
		DOT:         ".",
		ARROW:       "->",
		FATARROW:    "=>",
		COMMA:       ",",
		COLON:       ":",
		SEMICOLON:   ";",
		LPAREN:      "(",
		RPAREN:      ")",
		LBRACE:      "{",
		RBRACE:      "}",
		LBRACKET:    "[",
		RBRACKET:    "]",
		PIPE:        "|",
		FROM:        "FROM",
		SELECT:      "SELECT",
		WHERE:       "WHERE",
		JOIN:        "JOIN",
		ON:          "ON",
		AS:          "AS",
		ORDER:       "ORDER",
		BY:          "BY",
		ASC:         "ASC",
		DESC:        "DESC",
		LIMIT:       "LIMIT",
		OFFSET:      "OFFSET",
		GROUP:       "GROUP",
		HAVING:      "HAVING",
		COUNT:       "COUNT",
		SUM:         "SUM",
		AVG:         "AVG",
		MIN:         "MIN",
		MAX:         "MAX",
		WORKFLOW:    "WORKFLOW",
		STEP:        "STEP",
		CREATE:      "CREATE",
		UPDATE:      "UPDATE",
		DELETE:      "DELETE",
		READ:        "READ",
		CALL:        "CALL",
		IF:          "IF",
		THEN:        "THEN",
		ELSE:        "ELSE",
		ENDIF:       "ENDIF",
		FOREACH:     "FOREACH",
		IN:          "IN",
		ENDFOR:      "ENDFOR",
		RETRY:       "RETRY",
		TIMES:       "TIMES",
		DELAY:       "DELAY",
		ASSERT:      "ASSERT",
		EXPECT:      "EXPECT",
		FALLBACK:    "FALLBACK",
		ROLLBACK:    "ROLLBACK",
		THROW:       "THROW",
		BEGIN:       "BEGIN",
		COMMIT:      "COMMIT",
		TRANSACTION: "TRANSACTION",
		TRUE:        "TRUE",
		FALSE:       "FALSE",
		NULL:        "NULL",
		WITH:        "WITH",
		SET:         "SET",
		TO:          "TO",
		USING:       "USING",
		RETURNS:     "RETURNS",
		SERVICE:     "SERVICE",
		EXPOSE:      "EXPOSE",
	}
	if name, ok := names[t]; ok {
		return name
	}
	return "UNKNOWN"
}
