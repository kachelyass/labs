package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"unicode"
)

type TokenType string

const (
	Keyword     TokenType = "KEYWORD"
	Identifier  TokenType = "IDENTIFIER"
	ConstInt    TokenType = "CONSTANT_INT"
	ConstFloat  TokenType = "CONSTANT_FLOAT"
	ConstString TokenType = "CONSTANT_STRING"
	ConstBool   TokenType = "CONSTANT_BOOL"
	Operator    TokenType = "OPERATOR"
	Delimiter   TokenType = "DELIMITER"
	EOFToken    TokenType = "EOF"
)

type Token struct {
	Type   TokenType
	Lexeme string
	Line   int
	Column int
}

type LexError struct {
	Line        int
	Column      int
	ErrorType   string
	Explanation string
}

type Lexer struct {
	input []rune
	pos   int
	line  int
	col   int
}

var keywords = map[string]bool{
	"package": true,
	"import":  true,
	"func":    true,
	"var":     true,
	"go":      true,
	"defer":   true,
	"if":      true,
	"else":    true,
	"return":  true,
	"for":     true,
	"range":   true,
	"chan":    true,
	"make":    true,
}

var boolConstants = map[string]bool{
	"true":  true,
	"false": true,
}

var operators = map[string]bool{
	"=":  true,
	":=": true,
	"+":  true,
	"-":  true,
	"*":  true,
	"/":  true,
	"%":  true,
	"==": true,
	"!=": true,
	"<":  true,
	"<=": true,
	">":  true,
	">=": true,
	"<-": true,
	"&&": true,
	"||": true,
}

var delimiters = map[rune]bool{
	'(': true,
	')': true,
	'{': true,
	'}': true,
	'[': true,
	']': true,
	',': true,
	';': true,
	'.': true,
}

func NewLexer(input string) *Lexer {
	return &Lexer{input: []rune(input), line: 1, col: 1}
}

func (l *Lexer) current() rune {
	if l.pos >= len(l.input) {
		return 0
	}
	return l.input[l.pos]
}

func (l *Lexer) peek(offset int) rune {
	index := l.pos + offset
	if index >= len(l.input) {
		return 0
	}
	return l.input[index]
}

func (l *Lexer) advance() rune {
	ch := l.current()
	if ch == 0 {
		return 0
	}
	l.pos++
	if ch == '\n' {
		l.line++
		l.col = 1
	} else {
		l.col++
	}
	return ch
}

func (l *Lexer) skipWhitespace() {
	for unicode.IsSpace(l.current()) {
		l.advance()
	}
}

func isIdentifierStart(ch rune) bool {
	return unicode.IsLetter(ch) || ch == '_'
}

func isIdentifierPart(ch rune) bool {
	return unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_'
}

func (l *Lexer) readIdentifier() Token {
	startLine := l.line
	startCol := l.col
	var builder strings.Builder
	for isIdentifierPart(l.current()) {
		builder.WriteRune(l.advance())
	}
	lexeme := builder.String()
	switch {
	case keywords[lexeme]:
		return Token{Type: Keyword, Lexeme: lexeme, Line: startLine, Column: startCol}
	case boolConstants[lexeme]:
		return Token{Type: ConstBool, Lexeme: lexeme, Line: startLine, Column: startCol}
	default:
		return Token{Type: Identifier, Lexeme: lexeme, Line: startLine, Column: startCol}
	}
}

func (l *Lexer) readNumber() (Token, *LexError) {
	startLine := l.line
	startCol := l.col
	var builder strings.Builder
	hasDot := false

	for unicode.IsDigit(l.current()) {
		builder.WriteRune(l.advance())
	}

	if l.current() == '.' && unicode.IsDigit(l.peek(1)) {
		hasDot = true
		builder.WriteRune(l.advance())
		for unicode.IsDigit(l.current()) {
			builder.WriteRune(l.advance())
		}
	}

	if isIdentifierStart(l.current()) {
		for isIdentifierPart(l.current()) {
			builder.WriteRune(l.advance())
		}
		return Token{}, &LexError{Line: startLine, Column: startCol, ErrorType: "Некорректное число", Explanation: fmt.Sprintf("лексема %q содержит буквы в числовой константе", builder.String())}
	}

	if hasDot {
		return Token{Type: ConstFloat, Lexeme: builder.String(), Line: startLine, Column: startCol}, nil
	}
	return Token{Type: ConstInt, Lexeme: builder.String(), Line: startLine, Column: startCol}, nil
}

func (l *Lexer) readString() (Token, *LexError) {
	startLine := l.line
	startCol := l.col
	quote := l.advance()
	var builder strings.Builder
	builder.WriteRune(quote)

	for l.current() != 0 {
		ch := l.current()
		if quote == '"' && ch == '\n' {
			return Token{}, &LexError{Line: startLine, Column: startCol, ErrorType: "Незакрытая строка", Explanation: "строковый литерал не закрыт до конца строки"}
		}
		if ch == '\\' && quote == '"' {
			builder.WriteRune(l.advance())
			if l.current() != 0 {
				builder.WriteRune(l.advance())
			}
			continue
		}
		builder.WriteRune(l.advance())
		if ch == quote {
			return Token{Type: ConstString, Lexeme: builder.String(), Line: startLine, Column: startCol}, nil
		}
	}

	return Token{}, &LexError{Line: startLine, Column: startCol, ErrorType: "Незакрытая строка", Explanation: "строковый литерал не закрыт до конца файла"}
}

func (l *Lexer) readOperatorOrDelimiter() (Token, *LexError) {
	startLine := l.line
	startCol := l.col
	ch := l.current()

	if delimiters[ch] {
		l.advance()
		return Token{Type: Delimiter, Lexeme: string(ch), Line: startLine, Column: startCol}, nil
	}

	two := string([]rune{ch, l.peek(1)})
	if operators[two] {
		l.advance()
		l.advance()
		return Token{Type: Operator, Lexeme: two, Line: startLine, Column: startCol}, nil
	}

	one := string(ch)
	if operators[one] {
		l.advance()
		return Token{Type: Operator, Lexeme: one, Line: startLine, Column: startCol}, nil
	}

	l.advance()
	return Token{}, &LexError{Line: startLine, Column: startCol, ErrorType: "Недопустимый символ", Explanation: fmt.Sprintf("символ %q не может быть частью лексемы", one)}
}

func (l *Lexer) Scan() ([]Token, []LexError) {
	tokens := make([]Token, 0)
	errors := make([]LexError, 0)

	for l.current() != 0 {
		l.skipWhitespace()
		if l.current() == 0 {
			break
		}

		ch := l.current()
		switch {
		case isIdentifierStart(ch):
			tokens = append(tokens, l.readIdentifier())
		case unicode.IsDigit(ch):
			token, err := l.readNumber()
			if err != nil {
				errors = append(errors, *err)
			} else {
				tokens = append(tokens, token)
			}
		case ch == '"' || ch == '`':
			token, err := l.readString()
			if err != nil {
				errors = append(errors, *err)
			} else {
				tokens = append(tokens, token)
			}
		default:
			token, err := l.readOperatorOrDelimiter()
			if err != nil {
				errors = append(errors, *err)
			} else {
				tokens = append(tokens, token)
			}
		}
	}

	tokens = append(tokens, Token{Type: EOFToken, Lexeme: "<EOF>", Line: l.line, Column: l.col})
	return tokens, errors
}

type ASTNode struct {
	Kind     string
	Value    string
	Line     int
	Column   int
	Children []*ASTNode
}

func NewNode(kind string, value string, token Token, children ...*ASTNode) *ASTNode {
	return &ASTNode{Kind: kind, Value: value, Line: token.Line, Column: token.Column, Children: children}
}

func NewSyntheticNode(kind string, value string, children ...*ASTNode) *ASTNode {
	return &ASTNode{Kind: kind, Value: value, Children: children}
}

func (n *ASTNode) Add(children ...*ASTNode) {
	n.Children = append(n.Children, children...)
}

func (n *ASTNode) Print(indent string, last bool) {
	connector := "├── "
	nextIndent := indent + "│   "
	if last {
		connector = "└── "
		nextIndent = indent + "    "
	}
	label := n.Kind
	if n.Value != "" {
		label += ": " + n.Value
	}
	fmt.Println(indent + connector + label)
	for i, child := range n.Children {
		child.Print(nextIndent, i == len(n.Children)-1)
	}
}

type ParseError struct {
	Line     int
	Column   int
	Expected string
	Actual   Token
	Message  string
}

func (e ParseError) Error() string {
	return fmt.Sprintf("Синтаксическая ошибка [%d:%d]: %s. Ожидалось: %s, получено: (%s, %q)", e.Line, e.Column, e.Message, e.Expected, e.Actual.Type, e.Actual.Lexeme)
}

type Parser struct {
	tokens []Token
	pos    int
}

func NewParser(tokens []Token) *Parser {
	return &Parser{tokens: tokens}
}

func (p *Parser) current() Token {
	if p.pos >= len(p.tokens) {
		last := p.tokens[len(p.tokens)-1]
		return Token{Type: EOFToken, Lexeme: "<EOF>", Line: last.Line, Column: last.Column}
	}
	return p.tokens[p.pos]
}

func (p *Parser) peek(offset int) Token {
	index := p.pos + offset
	if index >= len(p.tokens) {
		return p.tokens[len(p.tokens)-1]
	}
	return p.tokens[index]
}

func (p *Parser) advance() Token {
	tok := p.current()
	if tok.Type != EOFToken {
		p.pos++
	}
	return tok
}

func (p *Parser) match(tt TokenType, lexeme string) bool {
	tok := p.current()
	if tok.Type != tt {
		return false
	}
	return lexeme == "" || tok.Lexeme == lexeme
}

func (p *Parser) expect(tt TokenType, lexeme string, expected string) (Token, error) {
	if p.match(tt, lexeme) {
		return p.advance(), nil
	}
	tok := p.current()
	return Token{}, ParseError{Line: tok.Line, Column: tok.Column, Expected: expected, Actual: tok, Message: "нарушена ожидаемая структура программы"}
}

func (p *Parser) optionalDelimiter(lexeme string) {
	if p.match(Delimiter, lexeme) {
		p.advance()
	}
}

func (p *Parser) ParseProgram() (*ASTNode, error) {
	root := NewSyntheticNode("Program", "")
	pkgKw, err := p.expect(Keyword, "package", "ключевое слово package")
	if err != nil {
		return nil, err
	}
	pkg, err := p.expect(Identifier, "", "имя пакета")
	if err != nil {
		return nil, err
	}
	root.Line = pkgKw.Line
	root.Column = pkgKw.Column
	root.Add(NewNode("Package", pkg.Lexeme, pkg))
	p.optionalDelimiter(";")

	if p.match(Keyword, "import") {
		imports, err := p.parseImportSection()
		if err != nil {
			return nil, err
		}
		root.Add(imports)
	}

	functions := NewSyntheticNode("Functions", "")
	for p.match(Keyword, "func") {
		fn, err := p.parseFunctionDecl()
		if err != nil {
			return nil, err
		}
		functions.Add(fn)
	}
	root.Add(functions)

	if !p.match(EOFToken, "") {
		tok := p.current()
		return nil, ParseError{Line: tok.Line, Column: tok.Column, Expected: "конец файла", Actual: tok, Message: "после объявления функций найдена лишняя лексема"}
	}
	return root, nil
}

func (p *Parser) parseImportSection() (*ASTNode, error) {
	kw, err := p.expect(Keyword, "import", "ключевое слово import")
	if err != nil {
		return nil, err
	}
	section := NewNode("ImportSection", "", kw)
	if p.match(Delimiter, "(") {
		p.advance()
		for !p.match(Delimiter, ")") {
			if p.match(EOFToken, "") {
				tok := p.current()
				return nil, ParseError{Line: tok.Line, Column: tok.Column, Expected: "закрывающая скобка )", Actual: tok, Message: "незакрытый список import"}
			}
			imp, err := p.expect(ConstString, "", "строковый литерал с именем пакета")
			if err != nil {
				return nil, err
			}
			section.Add(NewNode("Import", strings.Trim(imp.Lexeme, "\"`"), imp))
			p.optionalDelimiter(";")
		}
		if _, err := p.expect(Delimiter, ")", "закрывающая скобка )"); err != nil {
			return nil, err
		}
		p.optionalDelimiter(";")
		return section, nil
	}
	imp, err := p.expect(ConstString, "", "строковый литерал с именем пакета")
	if err != nil {
		return nil, err
	}
	section.Add(NewNode("Import", strings.Trim(imp.Lexeme, "\"`"), imp))
	p.optionalDelimiter(";")
	return section, nil
}

func (p *Parser) parseFunctionDecl() (*ASTNode, error) {
	kw, err := p.expect(Keyword, "func", "ключевое слово func")
	if err != nil {
		return nil, err
	}
	name, err := p.expect(Identifier, "", "имя функции")
	if err != nil {
		return nil, err
	}
	fn := NewNode("FuncDecl", name.Lexeme, kw)
	params, err := p.parseParameters()
	if err != nil {
		return nil, err
	}
	fn.Add(params)
	if p.isTypeStart() {
		ret, err := p.parseType()
		if err != nil {
			return nil, err
		}
		fn.Add(NewSyntheticNode("ReturnType", "", ret))
	}
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	fn.Add(body)
	return fn, nil
}

func (p *Parser) parseParameters() (*ASTNode, error) {
	open, err := p.expect(Delimiter, "(", "открывающая скобка параметров (")
	if err != nil {
		return nil, err
	}
	params := NewNode("Params", "", open)
	for !p.match(Delimiter, ")") {
		if p.match(EOFToken, "") {
			tok := p.current()
			return nil, ParseError{Line: tok.Line, Column: tok.Column, Expected: "закрывающая скобка параметров )", Actual: tok, Message: "незакрытый список параметров"}
		}
		name, err := p.expect(Identifier, "", "имя параметра")
		if err != nil {
			return nil, err
		}
		typeNode, err := p.parseType()
		if err != nil {
			return nil, err
		}
		params.Add(NewNode("Param", name.Lexeme, name, typeNode))
		if p.match(Delimiter, ",") {
			p.advance()
			continue
		}
		if !p.match(Delimiter, ")") {
			tok := p.current()
			return nil, ParseError{Line: tok.Line, Column: tok.Column, Expected: "запятая или закрывающая скобка )", Actual: tok, Message: "ошибка в списке параметров"}
		}
	}
	if _, err := p.expect(Delimiter, ")", "закрывающая скобка параметров )"); err != nil {
		return nil, err
	}
	return params, nil
}

func (p *Parser) isTypeStart() bool {
	return p.match(Identifier, "") || p.match(Keyword, "chan")
}

func (p *Parser) parseType() (*ASTNode, error) {
	if p.match(Keyword, "chan") {
		tok := p.advance()
		inner, err := p.parseType()
		if err != nil {
			return nil, err
		}
		return NewNode("Type", "chan", tok, inner), nil
	}
	name, err := p.expect(Identifier, "", "тип данных")
	if err != nil {
		return nil, err
	}
	value := name.Lexeme
	if p.match(Delimiter, ".") {
		p.advance()
		second, err := p.expect(Identifier, "", "имя типа после точки")
		if err != nil {
			return nil, err
		}
		value += "." + second.Lexeme
	}
	return NewNode("Type", value, name), nil
}

func (p *Parser) parseBlock() (*ASTNode, error) {
	open, err := p.expect(Delimiter, "{", "открывающая операторная скобка {")
	if err != nil {
		return nil, err
	}
	block := NewNode("Block", "", open)
	for !p.match(Delimiter, "}") {
		if p.match(EOFToken, "") {
			tok := p.current()
			return nil, ParseError{Line: tok.Line, Column: tok.Column, Expected: "закрывающая операторная скобка }", Actual: tok, Message: "незакрытый блок"}
		}
		stmt, err := p.parseStatement()
		if err != nil {
			return nil, err
		}
		if stmt != nil {
			block.Add(stmt)
		}
		p.optionalDelimiter(";")
	}
	if _, err := p.expect(Delimiter, "}", "закрывающая операторная скобка }"); err != nil {
		return nil, err
	}
	p.optionalDelimiter(";")
	return block, nil
}

func (p *Parser) parseStatement() (*ASTNode, error) {
	switch {
	case p.match(Keyword, "var"):
		return p.parseVarDecl()
	case p.match(Keyword, "return"):
		return p.parseReturnStmt()
	case p.match(Keyword, "for"):
		return p.parseForRangeStmt()
	case p.match(Keyword, "go"):
		return p.parseGoStmt()
	case p.match(Keyword, "defer"):
		return p.parseDeferStmt()
	case p.match(Keyword, "if"):
		return p.parseIfStmt()
	case p.match(Delimiter, ";"):
		p.advance()
		return nil, nil
	case p.match(Identifier, ""):
		return p.parseIdentifierStatement()
	default:
		tok := p.current()
		return nil, ParseError{Line: tok.Line, Column: tok.Column, Expected: "оператор: var, return, for, go, defer, if или выражение", Actual: tok, Message: "неожиданный токен в начале оператора"}
	}
}

func (p *Parser) parseVarDecl() (*ASTNode, error) {
	kw, err := p.expect(Keyword, "var", "ключевое слово var")
	if err != nil {
		return nil, err
	}
	name, err := p.expect(Identifier, "", "имя переменной")
	if err != nil {
		return nil, err
	}
	typeNode, err := p.parseType()
	if err != nil {
		return nil, err
	}
	node := NewNode("VarDecl", name.Lexeme, kw, typeNode)
	if p.match(Operator, "=") {
		p.advance()
		expr, err := p.parseExpression(0)
		if err != nil {
			return nil, err
		}
		node.Add(expr)
	}
	return node, nil
}

func (p *Parser) parseReturnStmt() (*ASTNode, error) {
	kw, err := p.expect(Keyword, "return", "ключевое слово return")
	if err != nil {
		return nil, err
	}
	if p.match(Delimiter, "}") || p.match(Delimiter, ";") || p.match(EOFToken, "") {
		return NewNode("ReturnStmt", "", kw), nil
	}
	expr, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}
	return NewNode("ReturnStmt", "", kw, expr), nil
}

func (p *Parser) parseForRangeStmt() (*ASTNode, error) {
	kw, err := p.expect(Keyword, "for", "ключевое слово for")
	if err != nil {
		return nil, err
	}
	name, err := p.expect(Identifier, "", "переменная цикла")
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(Operator, ":=", "оператор :="); err != nil {
		return nil, err
	}
	if _, err := p.expect(Keyword, "range", "ключевое слово range"); err != nil {
		return nil, err
	}
	rangeExpr, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	return NewNode("ForRangeStmt", "", kw, NewNode("Iterator", name.Lexeme, name), NewSyntheticNode("RangeExpr", "", rangeExpr), body), nil
}

func (p *Parser) parseGoStmt() (*ASTNode, error) {
	kw, err := p.expect(Keyword, "go", "ключевое слово go")
	if err != nil {
		return nil, err
	}
	if p.match(Keyword, "func") {
		lit, err := p.parseFuncLiteral()
		if err != nil {
			return nil, err
		}
		if p.match(Delimiter, "(") {
			args, err := p.parseArguments()
			if err != nil {
				return nil, err
			}
			return NewNode("GoStmt", "", kw, NewSyntheticNode("FuncLiteralCall", "", lit, args)), nil
		}
		return NewNode("GoStmt", "", kw, lit), nil
	}
	expr, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}
	return NewNode("GoStmt", "", kw, expr), nil
}

func (p *Parser) parseDeferStmt() (*ASTNode, error) {
	kw, err := p.expect(Keyword, "defer", "ключевое слово defer")
	if err != nil {
		return nil, err
	}
	expr, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}
	return NewNode("DeferStmt", "", kw, expr), nil
}

func (p *Parser) parseIfStmt() (*ASTNode, error) {
	kw, err := p.expect(Keyword, "if", "ключевое слово if")
	if err != nil {
		return nil, err
	}
	condition, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}
	thenBlock, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	ifNode := NewNode("IfStmt", "", kw, NewSyntheticNode("Condition", "", condition), NewSyntheticNode("Then", "", thenBlock))
	if p.match(Keyword, "else") {
		p.advance()
		elseBlock, err := p.parseBlock()
		if err != nil {
			return nil, err
		}
		ifNode.Add(NewSyntheticNode("Else", "", elseBlock))
	}
	return ifNode, nil
}

func (p *Parser) parseIdentifierStatement() (*ASTNode, error) {
	if p.peek(1).Type == Operator && p.peek(1).Lexeme == ":=" {
		name := p.advance()
		p.advance()
		expr, err := p.parseExpression(0)
		if err != nil {
			return nil, err
		}
		return NewNode("ShortVarDecl", name.Lexeme, name, expr), nil
	}
	if p.peek(1).Type == Operator && p.peek(1).Lexeme == "<-" {
		name := p.advance()
		p.advance()
		expr, err := p.parseExpression(0)
		if err != nil {
			return nil, err
		}
		return NewNode("SendStmt", "", name, NewNode("Channel", name.Lexeme, name), NewSyntheticNode("Value", "", expr)), nil
	}
	left, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}
	if p.match(Operator, "=") {
		op := p.advance()
		right, err := p.parseExpression(0)
		if err != nil {
			return nil, err
		}
		return NewNode("AssignStmt", "", op, left, right), nil
	}
	return NewNode("ExprStmt", "", Token{Line: left.Line, Column: left.Column}, left), nil
}

func (p *Parser) parseFuncLiteral() (*ASTNode, error) {
	kw, err := p.expect(Keyword, "func", "ключевое слово func")
	if err != nil {
		return nil, err
	}
	params, err := p.parseParameters()
	if err != nil {
		return nil, err
	}
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	return NewNode("FuncLiteral", "", kw, params, body), nil
}

var precedence = map[string]int{
	"||": 1,
	"&&": 2,
	"==": 3,
	"!=": 3,
	"<":  4,
	"<=": 4,
	">":  4,
	">=": 4,
	"+":  5,
	"-":  5,
	"*":  6,
	"/":  6,
	"%":  6,
}

func (p *Parser) parseExpression(minPrec int) (*ASTNode, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	for {
		tok := p.current()
		prec, ok := precedence[tok.Lexeme]
		if tok.Type != Operator || !ok || prec < minPrec {
			break
		}
		p.advance()
		right, err := p.parseExpression(prec + 1)
		if err != nil {
			return nil, err
		}
		left = NewNode("BinaryExpr", tok.Lexeme, tok, left, right)
	}
	return left, nil
}

func (p *Parser) parsePrimary() (*ASTNode, error) {
	tok := p.current()
	var node *ASTNode
	switch tok.Type {
	case Identifier:
		p.advance()
		node = NewNode("Identifier", tok.Lexeme, tok)
	case ConstInt:
		p.advance()
		node = NewNode("IntLiteral", tok.Lexeme, tok)
	case ConstFloat:
		p.advance()
		node = NewNode("FloatLiteral", tok.Lexeme, tok)
	case ConstString:
		p.advance()
		node = NewNode("StringLiteral", tok.Lexeme, tok)
	case ConstBool:
		p.advance()
		node = NewNode("BoolLiteral", tok.Lexeme, tok)
	case Keyword:
		if tok.Lexeme == "make" {
			p.advance()
			node = NewNode("Identifier", tok.Lexeme, tok)
		} else if tok.Lexeme == "chan" {
			typeNode, err := p.parseType()
			if err != nil {
				return nil, err
			}
			return typeNode, nil
		} else {
			return nil, ParseError{Line: tok.Line, Column: tok.Column, Expected: "выражение", Actual: tok, Message: "ключевое слово не может начинать выражение"}
		}
	case Delimiter:
		if tok.Lexeme == "(" {
			p.advance()
			expr, err := p.parseExpression(0)
			if err != nil {
				return nil, err
			}
			if _, err := p.expect(Delimiter, ")", "закрывающая скобка выражения )"); err != nil {
				return nil, err
			}
			node = NewNode("GroupedExpr", "", tok, expr)
		} else {
			return nil, ParseError{Line: tok.Line, Column: tok.Column, Expected: "выражение", Actual: tok, Message: "разделитель не может начинать выражение"}
		}
	default:
		return nil, ParseError{Line: tok.Line, Column: tok.Column, Expected: "выражение", Actual: tok, Message: "неожиданный токен в выражении"}
	}
	for {
		switch {
		case p.match(Delimiter, "."):
			p.advance()
			field, err := p.expect(Identifier, "", "идентификатор после точки")
			if err != nil {
				return nil, err
			}
			node = NewNode("SelectorExpr", "", field, node, NewNode("Field", field.Lexeme, field))
		case p.match(Delimiter, "("):
			args, err := p.parseArguments()
			if err != nil {
				return nil, err
			}
			node = NewNode("CallExpr", "", tok, node, args)
		default:
			return node, nil
		}
	}
}

func (p *Parser) parseArguments() (*ASTNode, error) {
	open, err := p.expect(Delimiter, "(", "открывающая скобка аргументов (")
	if err != nil {
		return nil, err
	}
	args := NewNode("Args", "", open)
	for !p.match(Delimiter, ")") {
		if p.match(EOFToken, "") {
			tok := p.current()
			return nil, ParseError{Line: tok.Line, Column: tok.Column, Expected: "закрывающая скобка аргументов )", Actual: tok, Message: "незакрытый список аргументов"}
		}
		expr, err := p.parseExpression(0)
		if err != nil {
			return nil, err
		}
		args.Add(expr)
		if p.match(Delimiter, ",") {
			p.advance()
			continue
		}
		if !p.match(Delimiter, ")") {
			tok := p.current()
			return nil, ParseError{Line: tok.Line, Column: tok.Column, Expected: "запятая или закрывающая скобка )", Actual: tok, Message: "ошибка в списке аргументов"}
		}
	}
	if _, err := p.expect(Delimiter, ")", "закрывающая скобка аргументов )"); err != nil {
		return nil, err
	}
	return args, nil
}

type Symbol struct {
	Name        string
	Kind        string
	Type        string
	Scope       string
	Declared    bool
	Initialized bool
	Line        int
}

type Triad struct {
	Number int
	Op     string
	Arg1   string
	Arg2   string
}

type SemanticError struct {
	ErrorType   string
	Explanation string
	Line        int
}

func (e SemanticError) String() string {
	if e.Line > 0 {
		return fmt.Sprintf("%s: %s (строка %d)", e.ErrorType, e.Explanation, e.Line)
	}
	return fmt.Sprintf("%s: %s", e.ErrorType, e.Explanation)
}

type FunctionSignature struct {
	Name       string
	ParamTypes []string
	ReturnType string
}

type ExprResult struct {
	Type    string
	Operand string
}

type ScopeFrame struct {
	Name    string
	Symbols map[string]*Symbol
}

type SemanticAnalyzer struct {
	scopes       []ScopeFrame
	symbols      []*Symbol
	functions    map[string]FunctionSignature
	imports      map[string]bool
	triads       []Triad
	errors       []SemanticError
	scopeCounter int
	currentFunc  FunctionSignature
}

func NewSemanticAnalyzer() *SemanticAnalyzer {
	a := &SemanticAnalyzer{
		functions: make(map[string]FunctionSignature),
		imports:   make(map[string]bool),
	}
	a.pushScope("global")
	return a
}

func (a *SemanticAnalyzer) pushScope(name string) {
	a.scopes = append(a.scopes, ScopeFrame{Name: name, Symbols: make(map[string]*Symbol)})
}

func (a *SemanticAnalyzer) popScope() {
	if len(a.scopes) > 1 {
		a.scopes = a.scopes[:len(a.scopes)-1]
	}
}

func (a *SemanticAnalyzer) currentScopeName() string {
	return a.scopes[len(a.scopes)-1].Name
}

func (a *SemanticAnalyzer) addSymbol(name, kind, typ string, initialized bool, line int) {
	frame := &a.scopes[len(a.scopes)-1]
	if _, exists := frame.Symbols[name]; exists {
		a.addError("ПОВТОРНОЕ ОБЪЯВЛЕНИЕ", fmt.Sprintf("идентификатор %q уже объявлен в области видимости %q", name, frame.Name), line)
		return
	}
	sym := &Symbol{Name: name, Kind: kind, Type: typ, Scope: frame.Name, Declared: true, Initialized: initialized, Line: line}
	frame.Symbols[name] = sym
	a.symbols = append(a.symbols, sym)
}

func (a *SemanticAnalyzer) lookup(name string) (*Symbol, bool) {
	for i := len(a.scopes) - 1; i >= 0; i-- {
		if sym, ok := a.scopes[i].Symbols[name]; ok {
			return sym, true
		}
	}
	return nil, false
}

func (a *SemanticAnalyzer) addError(errorType, explanation string, line int) {
	a.errors = append(a.errors, SemanticError{ErrorType: errorType, Explanation: explanation, Line: line})
}

func (a *SemanticAnalyzer) addTriad(op, arg1, arg2 string) string {
	tr := Triad{Number: len(a.triads) + 1, Op: op, Arg1: arg1, Arg2: arg2}
	a.triads = append(a.triads, tr)
	return fmt.Sprintf("^%d", tr.Number)
}

func (a *SemanticAnalyzer) Analyze(root *ASTNode) {
	a.registerImports(root)
	a.registerFunctions(root)
	a.analyzeFunctions(root)
}

func (a *SemanticAnalyzer) registerImports(root *ASTNode) {
	imports := findChild(root, "ImportSection")
	if imports == nil {
		return
	}
	for _, imp := range imports.Children {
		if imp.Kind == "Import" {
			alias := imp.Value
			if idx := strings.LastIndex(alias, "/"); idx >= 0 {
				alias = alias[idx+1:]
			}
			a.imports[alias] = true
			a.addSymbol(alias, "import", imp.Value, true, imp.Line)
		}
	}
}

func (a *SemanticAnalyzer) registerFunctions(root *ASTNode) {
	functions := findChild(root, "Functions")
	if functions == nil {
		return
	}
	for _, fn := range functions.Children {
		if fn.Kind != "FuncDecl" {
			continue
		}
		params := findChild(fn, "Params")
		paramTypes := make([]string, 0)
		if params != nil {
			for _, param := range params.Children {
				if len(param.Children) > 0 {
					paramTypes = append(paramTypes, typeToString(param.Children[0]))
				}
			}
		}
		returnType := "void"
		if ret := findChild(fn, "ReturnType"); ret != nil && len(ret.Children) > 0 {
			returnType = typeToString(ret.Children[0])
		}
		sig := FunctionSignature{Name: fn.Value, ParamTypes: paramTypes, ReturnType: returnType}
		if _, exists := a.functions[fn.Value]; exists {
			a.addError("ПОВТОРНОЕ ОБЪЯВЛЕНИЕ", fmt.Sprintf("функция %q уже объявлена", fn.Value), fn.Line)
			continue
		}
		a.functions[fn.Value] = sig
		a.addSymbol(fn.Value, "function", signatureToString(sig), true, fn.Line)
	}
}

func (a *SemanticAnalyzer) analyzeFunctions(root *ASTNode) {
	functions := findChild(root, "Functions")
	if functions == nil {
		return
	}
	for _, fn := range functions.Children {
		if fn.Kind == "FuncDecl" {
			a.analyzeFunction(fn)
		}
	}
}

func (a *SemanticAnalyzer) analyzeFunction(fn *ASTNode) {
	sig := a.functions[fn.Value]
	oldFunc := a.currentFunc
	a.currentFunc = sig
	a.pushScope("func:" + fn.Value)
	if params := findChild(fn, "Params"); params != nil {
		for _, param := range params.Children {
			typ := "unknown"
			if len(param.Children) > 0 {
				typ = typeToString(param.Children[0])
			}
			a.addSymbol(param.Value, "param", typ, true, param.Line)
		}
	}
	if block := findChild(fn, "Block"); block != nil {
		a.analyzeBlock(block, false)
	}
	a.popScope()
	a.currentFunc = oldFunc
}

func (a *SemanticAnalyzer) analyzeBlock(block *ASTNode, newScope bool) {
	if newScope {
		a.scopeCounter++
		a.pushScope(fmt.Sprintf("block:%d", a.scopeCounter))
		defer a.popScope()
	}
	for _, stmt := range block.Children {
		a.analyzeStatement(stmt)
	}
}

func (a *SemanticAnalyzer) analyzeStatement(stmt *ASTNode) {
	switch stmt.Kind {
	case "VarDecl":
		a.analyzeVarDecl(stmt)
	case "ShortVarDecl":
		a.analyzeShortVarDecl(stmt)
	case "AssignStmt":
		a.analyzeAssignStmt(stmt)
	case "SendStmt":
		a.analyzeSendStmt(stmt)
	case "ExprStmt":
		if len(stmt.Children) > 0 {
			a.analyzeExpr(stmt.Children[0])
		}
	case "ReturnStmt":
		a.analyzeReturnStmt(stmt)
	case "ForRangeStmt":
		a.analyzeForRangeStmt(stmt)
	case "GoStmt":
		a.analyzeGoStmt(stmt)
	case "DeferStmt":
		a.analyzeDeferStmt(stmt)
	case "IfStmt":
		a.analyzeIfStmt(stmt)
	}
}

func (a *SemanticAnalyzer) analyzeVarDecl(stmt *ASTNode) {
	typ := "unknown"
	if len(stmt.Children) > 0 && stmt.Children[0].Kind == "Type" {
		typ = typeToString(stmt.Children[0])
	}
	initialized := true // В Go var получает нулевое значение, поэтому переменная считается инициализированной.
	a.addSymbol(stmt.Value, "variable", typ, initialized, stmt.Line)
	if len(stmt.Children) > 1 {
		right := a.analyzeExpr(stmt.Children[1])
		if !typesCompatible(typ, right.Type) {
			a.addError("НЕСООТВЕТСТВИЕ ТИПОВ", fmt.Sprintf("переменной %q типа %s нельзя присвоить выражение типа %s", stmt.Value, typ, right.Type), stmt.Line)
		}
		a.addTriad("=", stmt.Value, right.Operand)
	}
}

func (a *SemanticAnalyzer) analyzeShortVarDecl(stmt *ASTNode) {
	if _, exists := a.scopes[len(a.scopes)-1].Symbols[stmt.Value]; exists {
		a.addError("ПОВТОРНОЕ ОБЪЯВЛЕНИЕ", fmt.Sprintf("переменная %q уже объявлена в области видимости %q", stmt.Value, a.currentScopeName()), stmt.Line)
		return
	}
	right := ExprResult{Type: "unknown", Operand: "?"}
	if len(stmt.Children) > 0 {
		right = a.analyzeExpr(stmt.Children[0])
	}
	a.addSymbol(stmt.Value, "variable", right.Type, true, stmt.Line)
	a.addTriad(":=", stmt.Value, right.Operand)
}

func (a *SemanticAnalyzer) analyzeAssignStmt(stmt *ASTNode) {
	if len(stmt.Children) < 2 {
		return
	}
	left := stmt.Children[0]
	right := a.analyzeExpr(stmt.Children[1])
	leftRes := a.analyzeExpr(left)
	if !typesCompatible(leftRes.Type, right.Type) {
		a.addError("НЕСООТВЕТСТВИЕ ТИПОВ", fmt.Sprintf("левой части типа %s нельзя присвоить выражение типа %s", leftRes.Type, right.Type), stmt.Line)
	}
	if left.Kind == "Identifier" {
		if sym, ok := a.lookup(left.Value); ok {
			sym.Initialized = true
		}
	}
	a.addTriad("=", leftRes.Operand, right.Operand)
}

func (a *SemanticAnalyzer) analyzeSendStmt(stmt *ASTNode) {
	if len(stmt.Children) < 2 {
		return
	}
	channelName := stmt.Children[0].Value
	chSym, ok := a.lookup(channelName)
	if !ok {
		a.addError("НЕОБЪЯВЛЕННАЯ ПЕРЕМЕННАЯ", fmt.Sprintf("канал %q используется до объявления", channelName), stmt.Line)
		return
	}
	value := a.analyzeExpr(stmt.Children[1].Children[0])
	if !strings.HasPrefix(chSym.Type, "chan ") {
		a.addError("НЕСООТВЕТСТВИЕ ТИПОВ", fmt.Sprintf("оператор <- применён к %q типа %s, ожидался канал", channelName, chSym.Type), stmt.Line)
	} else {
		elemType := strings.TrimSpace(strings.TrimPrefix(chSym.Type, "chan"))
		if !typesCompatible(elemType, value.Type) {
			a.addError("НЕСООТВЕТСТВИЕ ТИПОВ", fmt.Sprintf("в канал %q типа %s нельзя отправить значение типа %s", channelName, chSym.Type, value.Type), stmt.Line)
		}
	}
	a.addTriad("<-", channelName, value.Operand)
}

func (a *SemanticAnalyzer) analyzeReturnStmt(stmt *ASTNode) {
	if len(stmt.Children) == 0 {
		if a.currentFunc.ReturnType != "" && a.currentFunc.ReturnType != "void" {
			a.addError("НЕСООТВЕТСТВИЕ ТИПОВ", fmt.Sprintf("функция %q должна вернуть значение типа %s", a.currentFunc.Name, a.currentFunc.ReturnType), stmt.Line)
		}
		a.addTriad("return", "-", "-")
		return
	}
	value := a.analyzeExpr(stmt.Children[0])
	if a.currentFunc.ReturnType == "void" {
		a.addError("НЕСООТВЕТСТВИЕ ТИПОВ", fmt.Sprintf("функция %q не должна возвращать значение", a.currentFunc.Name), stmt.Line)
	} else if !typesCompatible(a.currentFunc.ReturnType, value.Type) {
		a.addError("НЕСООТВЕТСТВИЕ ТИПОВ", fmt.Sprintf("функция %q должна вернуть %s, но возвращает %s", a.currentFunc.Name, a.currentFunc.ReturnType, value.Type), stmt.Line)
	}
	a.addTriad("return", value.Operand, "-")
}

func (a *SemanticAnalyzer) analyzeForRangeStmt(stmt *ASTNode) {
	if len(stmt.Children) < 3 {
		return
	}
	iterator := stmt.Children[0]
	rangeExprNode := stmt.Children[1].Children[0]
	rangeValue := a.analyzeExpr(rangeExprNode)
	itemType := "int"
	if strings.HasPrefix(rangeValue.Type, "chan ") {
		itemType = strings.TrimSpace(strings.TrimPrefix(rangeValue.Type, "chan"))
	} else if rangeValue.Type != "int" {
		a.addError("НЕСООТВЕТСТВИЕ ТИПОВ", fmt.Sprintf("range поддержан только для int и chan T, получен тип %s", rangeValue.Type), stmt.Line)
	}
	a.addTriad("range", iterator.Value, rangeValue.Operand)
	a.scopeCounter++
	a.pushScope(fmt.Sprintf("for:%d", a.scopeCounter))
	a.addSymbol(iterator.Value, "variable", itemType, true, iterator.Line)
	a.analyzeBlock(stmt.Children[2], false)
	a.popScope()
}

func (a *SemanticAnalyzer) analyzeGoStmt(stmt *ASTNode) {
	if len(stmt.Children) == 0 {
		return
	}
	child := stmt.Children[0]
	if child.Kind == "FuncLiteralCall" {
		ref := a.analyzeFuncLiteralCall(child)
		a.addTriad("go", ref, "-")
		return
	}
	res := a.analyzeExpr(child)
	a.addTriad("go", res.Operand, "-")
}

func (a *SemanticAnalyzer) analyzeDeferStmt(stmt *ASTNode) {
	if len(stmt.Children) == 0 {
		return
	}
	res := a.analyzeExpr(stmt.Children[0])
	a.addTriad("defer", res.Operand, "-")
}

func (a *SemanticAnalyzer) analyzeIfStmt(stmt *ASTNode) {
	if len(stmt.Children) < 2 {
		return
	}
	conditionNode := stmt.Children[0].Children[0]
	condition := a.analyzeExpr(conditionNode)
	if condition.Type != "bool" && condition.Type != "unknown" {
		a.addError("НЕСООТВЕТСТВИЕ ТИПОВ", fmt.Sprintf("условие if должно иметь тип bool, получен тип %s", condition.Type), stmt.Line)
	}
	a.addTriad("if", condition.Operand, "then")
	if thenBlock := findChild(stmt.Children[1], "Block"); thenBlock != nil {
		a.analyzeBlock(thenBlock, true)
	}
	if len(stmt.Children) > 2 {
		a.addTriad("else", "-", "-")
		if elseBlock := findChild(stmt.Children[2], "Block"); elseBlock != nil {
			a.analyzeBlock(elseBlock, true)
		}
	}
	a.addTriad("endif", "-", "-")
}

func (a *SemanticAnalyzer) analyzeFuncLiteralCall(node *ASTNode) string {
	if len(node.Children) < 2 {
		return "?"
	}
	lit := node.Children[0]
	args := node.Children[1]
	paramTypes := make([]string, 0)
	if params := findChild(lit, "Params"); params != nil {
		for _, param := range params.Children {
			typ := "unknown"
			if len(param.Children) > 0 {
				typ = typeToString(param.Children[0])
			}
			paramTypes = append(paramTypes, typ)
		}
	}
	if len(args.Children) != len(paramTypes) {
		a.addError("ОШИБКА ВЫЗОВА", fmt.Sprintf("анонимная функция ожидает %d аргументов, передано %d", len(paramTypes), len(args.Children)), node.Line)
	}
	argOperands := make([]string, 0)
	argValues := make([]ExprResult, 0)
	for _, arg := range args.Children {
		res := a.analyzeExpr(arg)
		argValues = append(argValues, res)
		argOperands = append(argOperands, res.Operand)
	}
	for i := range paramTypes {
		if i < len(argValues) && !typesCompatible(paramTypes[i], argValues[i].Type) {
			a.addError("НЕСООТВЕТСТВИЕ ТИПОВ", fmt.Sprintf("аргумент %d анонимной функции: ожидался %s, получен %s", i+1, paramTypes[i], argValues[i].Type), node.Line)
		}
	}
	a.scopeCounter++
	a.pushScope(fmt.Sprintf("func_literal:%d", a.scopeCounter))
	if params := findChild(lit, "Params"); params != nil {
		for _, param := range params.Children {
			typ := "unknown"
			if len(param.Children) > 0 {
				typ = typeToString(param.Children[0])
			}
			a.addSymbol(param.Value, "param", typ, true, param.Line)
		}
	}
	if block := findChild(lit, "Block"); block != nil {
		a.analyzeBlock(block, false)
	}
	a.popScope()
	return a.addTriad("call", "func_literal", strings.Join(argOperands, ", "))
}

func (a *SemanticAnalyzer) analyzeExpr(expr *ASTNode) ExprResult {
	switch expr.Kind {
	case "IntLiteral":
		return ExprResult{Type: "int", Operand: expr.Value}
	case "FloatLiteral":
		return ExprResult{Type: "float", Operand: expr.Value}
	case "StringLiteral":
		return ExprResult{Type: "string", Operand: expr.Value}
	case "BoolLiteral":
		return ExprResult{Type: "bool", Operand: expr.Value}
	case "Identifier":
		if expr.Value == "make" || expr.Value == "close" {
			return ExprResult{Type: "builtin", Operand: expr.Value}
		}
		if sig, ok := a.functions[expr.Value]; ok {
			return ExprResult{Type: "func:" + signatureToString(sig), Operand: expr.Value}
		}
		sym, ok := a.lookup(expr.Value)
		if !ok {
			a.addError("НЕОБЪЯВЛЕННАЯ ПЕРЕМЕННАЯ", fmt.Sprintf("идентификатор %q используется до объявления", expr.Value), expr.Line)
			return ExprResult{Type: "unknown", Operand: expr.Value}
		}
		if !sym.Initialized {
			a.addError("НЕИНИЦИАЛИЗИРОВАННАЯ ПЕРЕМЕННАЯ", fmt.Sprintf("идентификатор %q используется до инициализации", expr.Value), expr.Line)
		}
		return ExprResult{Type: sym.Type, Operand: expr.Value}
	case "Type":
		return ExprResult{Type: "type", Operand: typeToString(expr)}
	case "GroupedExpr":
		if len(expr.Children) == 0 {
			return ExprResult{Type: "unknown", Operand: "?"}
		}
		return a.analyzeExpr(expr.Children[0])
	case "SelectorExpr":
		return a.analyzeSelector(expr)
	case "BinaryExpr":
		return a.analyzeBinaryExpr(expr)
	case "CallExpr":
		return a.analyzeCallExpr(expr)
	default:
		return ExprResult{Type: "unknown", Operand: expr.Kind}
	}
}

func (a *SemanticAnalyzer) analyzeSelector(expr *ASTNode) ExprResult {
	if len(expr.Children) < 2 {
		return ExprResult{Type: "unknown", Operand: "?"}
	}
	left := a.analyzeExpr(expr.Children[0])
	field := expr.Children[1].Value
	operand := left.Operand + "." + field
	if a.imports[left.Operand] {
		return ExprResult{Type: "selector", Operand: operand}
	}
	if left.Type == "sync.WaitGroup" {
		switch field {
		case "Add":
			return ExprResult{Type: "method:int->void", Operand: operand}
		case "Done", "Wait":
			return ExprResult{Type: "method:void", Operand: operand}
		}
	}
	if left.Type == "unknown" {
		return ExprResult{Type: "unknown", Operand: operand}
	}
	return ExprResult{Type: "selector", Operand: operand}
}

func (a *SemanticAnalyzer) analyzeBinaryExpr(expr *ASTNode) ExprResult {
	if len(expr.Children) < 2 {
		return ExprResult{Type: "unknown", Operand: "?"}
	}
	left := a.analyzeExpr(expr.Children[0])
	right := a.analyzeExpr(expr.Children[1])
	op := expr.Value
	switch op {
	case "+", "-", "*", "/", "%":
		if !isNumeric(left.Type) || !isNumeric(right.Type) {
			a.addError("НЕДОПУСТИМАЯ ОПЕРАЦИЯ", fmt.Sprintf("оператор %s требует числовые операнды, получены %s и %s", op, left.Type, right.Type), expr.Line)
			return ExprResult{Type: "unknown", Operand: a.addTriad(op, left.Operand, right.Operand)}
		}
		resultType := left.Type
		if left.Type == "float" || right.Type == "float" {
			resultType = "float"
		}
		return ExprResult{Type: resultType, Operand: a.addTriad(op, left.Operand, right.Operand)}
	case "==", "!=", "<", "<=", ">", ">=":
		if !typesCompatible(left.Type, right.Type) {
			a.addError("НЕСООТВЕТСТВИЕ ТИПОВ", fmt.Sprintf("оператор %s сравнивает несовместимые типы %s и %s", op, left.Type, right.Type), expr.Line)
		}
		return ExprResult{Type: "bool", Operand: a.addTriad(op, left.Operand, right.Operand)}
	case "&&", "||":
		if left.Type != "bool" || right.Type != "bool" {
			a.addError("НЕДОПУСТИМАЯ ОПЕРАЦИЯ", fmt.Sprintf("оператор %s требует bool и bool, получены %s и %s", op, left.Type, right.Type), expr.Line)
		}
		return ExprResult{Type: "bool", Operand: a.addTriad(op, left.Operand, right.Operand)}
	default:
		return ExprResult{Type: "unknown", Operand: a.addTriad(op, left.Operand, right.Operand)}
	}
}

func (a *SemanticAnalyzer) analyzeCallExpr(expr *ASTNode) ExprResult {
	if len(expr.Children) < 2 {
		return ExprResult{Type: "unknown", Operand: "call(?)"}
	}
	callee := a.analyzeExpr(expr.Children[0])
	argsNode := expr.Children[1]
	args := make([]ExprResult, 0)
	operands := make([]string, 0)
	for _, arg := range argsNode.Children {
		res := a.analyzeExpr(arg)
		args = append(args, res)
		operands = append(operands, res.Operand)
	}
	calleeName := callee.Operand
	switch {
	case calleeName == "make":
		if len(args) != 1 || args[0].Type != "type" {
			a.addError("ОШИБКА ВЫЗОВА", "make в тестовой программе должен принимать один аргумент типа chan T", expr.Line)
			return ExprResult{Type: "unknown", Operand: a.addTriad("call", "make", strings.Join(operands, ", "))}
		}
		ref := a.addTriad("call", "make", args[0].Operand)
		return ExprResult{Type: args[0].Operand, Operand: ref}
	case calleeName == "close":
		if len(args) != 1 || !strings.HasPrefix(args[0].Type, "chan ") {
			a.addError("ОШИБКА ВЫЗОВА", "close должен принимать один аргумент канального типа", expr.Line)
		}
		ref := a.addTriad("call", "close", strings.Join(operands, ", "))
		return ExprResult{Type: "void", Operand: ref}
	case calleeName == "fmt.Println":
		if !a.imports["fmt"] {
			a.addError("НЕОБЪЯВЛЕННЫЙ ИМПОРТ", "используется fmt.Println, но пакет fmt не импортирован", expr.Line)
		}
		ref := a.addTriad("call", "fmt.Println", strings.Join(operands, ", "))
		return ExprResult{Type: "void", Operand: ref}
	case calleeName == "wg.Add":
		if len(args) != 1 || !typesCompatible("int", args[0].Type) {
			a.addError("ОШИБКА ВЫЗОВА", "метод wg.Add ожидает один аргумент int", expr.Line)
		}
		ref := a.addTriad("call", "wg.Add", strings.Join(operands, ", "))
		return ExprResult{Type: "void", Operand: ref}
	case calleeName == "wg.Done" || calleeName == "wg.Wait":
		if len(args) != 0 {
			a.addError("ОШИБКА ВЫЗОВА", fmt.Sprintf("метод %s не принимает аргументы", calleeName), expr.Line)
		}
		ref := a.addTriad("call", calleeName, strings.Join(operands, ", "))
		return ExprResult{Type: "void", Operand: ref}
	default:
		if sig, ok := a.functions[calleeName]; ok {
			a.checkFunctionArguments(sig, args, expr.Line)
			ref := a.addTriad("call", calleeName, strings.Join(operands, ", "))
			return ExprResult{Type: sig.ReturnType, Operand: ref}
		}
		if strings.HasPrefix(callee.Type, "method") || strings.HasPrefix(callee.Type, "selector") {
			ref := a.addTriad("call", calleeName, strings.Join(operands, ", "))
			return ExprResult{Type: "void", Operand: ref}
		}
		a.addError("ОШИБКА ВЫЗОВА", fmt.Sprintf("неизвестная вызываемая функция %q", calleeName), expr.Line)
		return ExprResult{Type: "unknown", Operand: a.addTriad("call", calleeName, strings.Join(operands, ", "))}
	}
}

func (a *SemanticAnalyzer) checkFunctionArguments(sig FunctionSignature, args []ExprResult, line int) {
	if len(args) != len(sig.ParamTypes) {
		a.addError("ОШИБКА ВЫЗОВА", fmt.Sprintf("функция %q ожидает %d аргументов, передано %d", sig.Name, len(sig.ParamTypes), len(args)), line)
		return
	}
	for i := range args {
		if !typesCompatible(sig.ParamTypes[i], args[i].Type) {
			a.addError("НЕСООТВЕТСТВИЕ ТИПОВ", fmt.Sprintf("аргумент %d функции %q: ожидался %s, получен %s", i+1, sig.Name, sig.ParamTypes[i], args[i].Type), line)
		}
	}
}

func findChild(node *ASTNode, kind string) *ASTNode {
	if node == nil {
		return nil
	}
	for _, child := range node.Children {
		if child.Kind == kind {
			return child
		}
	}
	return nil
}

func typeToString(node *ASTNode) string {
	if node == nil {
		return "unknown"
	}
	if node.Kind != "Type" {
		return node.Value
	}
	if node.Value == "chan" && len(node.Children) > 0 {
		return "chan " + typeToString(node.Children[0])
	}
	return node.Value
}

func signatureToString(sig FunctionSignature) string {
	return fmt.Sprintf("func(%s) %s", strings.Join(sig.ParamTypes, ", "), sig.ReturnType)
}

func typesCompatible(expected, actual string) bool {
	if expected == actual || expected == "unknown" || actual == "unknown" {
		return true
	}
	if expected == "float" && actual == "int" {
		return true
	}
	return false
}

func isNumeric(typ string) bool {
	return typ == "int" || typ == "float" || typ == "unknown"
}

func loadSource() string {
	paths := []string{
		"../lab1/output.txt",
		"lab1/output.txt",
		"../lab1/test.go",
		"lab1/test.go",
		"test.go",
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err == nil {
			return string(data)
		}
	}
	fmt.Println("Ошибка: не удалось открыть исходный файл.")
	fmt.Println("Положи очищенный lab1/output.txt рядом с проектом или запусти из папки lab4 внутри репозитория.")
	os.Exit(1)
	return ""
}

func printAST(ast *ASTNode) {
	fmt.Println("AST:")
	fmt.Println(ast.Kind)
	for i, child := range ast.Children {
		child.Print("", i == len(ast.Children)-1)
	}
}

func printSymbolTable(symbols []*Symbol) {
	fmt.Println("Таблица символов:")
	fmt.Printf("%-14s | %-10s | %-22s | %-18s | %-11s | %-16s | %-6s\n", "Имя", "Вид", "Тип", "Область", "Объявлена", "Инициализирована", "Строка")
	fmt.Println(strings.Repeat("-", 118))
	sorted := append([]*Symbol(nil), symbols...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Scope == sorted[j].Scope {
			return sorted[i].Name < sorted[j].Name
		}
		return sorted[i].Scope < sorted[j].Scope
	})
	for _, s := range sorted {
		fmt.Printf("%-14s | %-10s | %-22s | %-18s | %-11t | %-16t | %-6d\n", s.Name, s.Kind, s.Type, s.Scope, s.Declared, s.Initialized, s.Line)
	}
}

func printTriads(triads []Triad) {
	fmt.Println("Триады:")
	for _, t := range triads {
		fmt.Printf("%d) (%s, %s, %s)\n", t.Number, t.Op, t.Arg1, t.Arg2)
	}
}

func printSemanticErrors(errors []SemanticError) {
	for _, err := range errors {
		fmt.Println(err.String())
	}
}

func printLexErrors(errors []LexError) {
	for _, err := range errors {
		fmt.Printf("Лексическая ошибка [%d:%d]. %s: %s\n", err.Line, err.Column, err.ErrorType, err.Explanation)
	}
}

func main() {
	source := loadSource()
	lexer := NewLexer(source)
	tokens, lexErrors := lexer.Scan()
	if len(lexErrors) > 0 {
		fmt.Printf("Лексический анализ завершён с ошибками: %d\n", len(lexErrors))
		printLexErrors(lexErrors)
		return
	}

	parser := NewParser(tokens)
	ast, err := parser.ParseProgram()
	if err != nil {
		fmt.Println(err)
		fmt.Println("Синтаксический анализ завершён с ошибками.")
		return
	}

	analyzer := NewSemanticAnalyzer()
	analyzer.Analyze(ast)

	printAST(ast)
	fmt.Println()
	printSymbolTable(analyzer.symbols)
	fmt.Println()

	if len(analyzer.errors) == 0 {
		fmt.Println("Семантический анализ завершён успешно. Ошибок не найдено.")
	} else {
		fmt.Printf("Семантический анализ завершён с ошибками. Найдено ошибок: %d.\n", len(analyzer.errors))
		printSemanticErrors(analyzer.errors)
	}
	fmt.Println()
	printTriads(analyzer.triads)
}
