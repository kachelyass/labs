package main

import (
	"fmt"
	"os"
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
		return Token{}, &LexError{
			Line:        startLine,
			Column:      startCol,
			ErrorType:   "Некорректное число",
			Explanation: fmt.Sprintf("лексема %q начинается с цифры или содержит буквы в числовой константе", builder.String()),
		}
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
	Children []*ASTNode
}

func NewNode(kind string, value string, children ...*ASTNode) *ASTNode {
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
	return Token{}, ParseError{
		Line:     tok.Line,
		Column:   tok.Column,
		Expected: expected,
		Actual:   tok,
		Message:  "нарушена ожидаемая структура программы",
	}
}

func (p *Parser) optionalDelimiter(lexeme string) {
	if p.match(Delimiter, lexeme) {
		p.advance()
	}
}

func (p *Parser) ParseProgram() (*ASTNode, error) {
	root := NewNode("Program", "")

	if _, err := p.expect(Keyword, "package", "ключевое слово package"); err != nil {
		return nil, err
	}
	pkg, err := p.expect(Identifier, "", "имя пакета")
	if err != nil {
		return nil, err
	}
	root.Add(NewNode("Package", pkg.Lexeme))
	p.optionalDelimiter(";")

	if p.match(Keyword, "import") {
		imports, err := p.parseImportSection()
		if err != nil {
			return nil, err
		}
		root.Add(imports)
	}

	functions := NewNode("Functions", "")
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
	if _, err := p.expect(Keyword, "import", "ключевое слово import"); err != nil {
		return nil, err
	}
	section := NewNode("ImportSection", "")

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
			section.Add(NewNode("Import", strings.Trim(imp.Lexeme, "\"`")))
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
	section.Add(NewNode("Import", strings.Trim(imp.Lexeme, "\"`")))
	p.optionalDelimiter(";")
	return section, nil
}

func (p *Parser) parseFunctionDecl() (*ASTNode, error) {
	if _, err := p.expect(Keyword, "func", "ключевое слово func"); err != nil {
		return nil, err
	}
	name, err := p.expect(Identifier, "", "имя функции")
	if err != nil {
		return nil, err
	}

	fn := NewNode("FuncDecl", name.Lexeme)
	params, err := p.parseParameters()
	if err != nil {
		return nil, err
	}
	fn.Add(params)

	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	fn.Add(body)
	return fn, nil
}

func (p *Parser) parseParameters() (*ASTNode, error) {
	if _, err := p.expect(Delimiter, "(", "открывающая скобка параметров ("); err != nil {
		return nil, err
	}
	params := NewNode("Params", "")

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
		params.Add(NewNode("Param", name.Lexeme, typeNode))

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

func (p *Parser) parseType() (*ASTNode, error) {
	if p.match(Keyword, "chan") {
		p.advance()
		inner, err := p.parseType()
		if err != nil {
			return nil, err
		}
		return NewNode("Type", "chan", inner), nil
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
	return NewNode("Type", value), nil
}

func (p *Parser) parseBlock() (*ASTNode, error) {
	if _, err := p.expect(Delimiter, "{", "открывающая операторная скобка {"); err != nil {
		return nil, err
	}
	block := NewNode("Block", "")

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
	if _, err := p.expect(Keyword, "var", "ключевое слово var"); err != nil {
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
	return NewNode("VarDecl", name.Lexeme, typeNode), nil
}

func (p *Parser) parseReturnStmt() (*ASTNode, error) {
	if _, err := p.expect(Keyword, "return", "ключевое слово return"); err != nil {
		return nil, err
	}

	if p.match(Delimiter, "}") || p.match(Delimiter, ";") || p.match(EOFToken, "") {
		return NewNode("ReturnStmt", ""), nil
	}
	expr, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}
	return NewNode("ReturnStmt", "", expr), nil
}

func (p *Parser) parseForRangeStmt() (*ASTNode, error) {
	if _, err := p.expect(Keyword, "for", "ключевое слово for"); err != nil {
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
	return NewNode("ForRangeStmt", "", NewNode("Iterator", name.Lexeme), NewNode("RangeExpr", "", rangeExpr), body), nil
}

func (p *Parser) parseGoStmt() (*ASTNode, error) {
	if _, err := p.expect(Keyword, "go", "ключевое слово go"); err != nil {
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
			return NewNode("GoStmt", "", NewNode("FuncLiteralCall", "", lit, args)), nil
		}
		return NewNode("GoStmt", "", lit), nil
	}

	expr, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}
	return NewNode("GoStmt", "", expr), nil
}

func (p *Parser) parseDeferStmt() (*ASTNode, error) {
	if _, err := p.expect(Keyword, "defer", "ключевое слово defer"); err != nil {
		return nil, err
	}
	expr, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}
	return NewNode("DeferStmt", "", expr), nil
}

func (p *Parser) parseIfStmt() (*ASTNode, error) {
	if _, err := p.expect(Keyword, "if", "ключевое слово if"); err != nil {
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
	ifNode := NewNode("IfStmt", "", NewNode("Condition", "", condition), NewNode("Then", "", thenBlock))
	if p.match(Keyword, "else") {
		p.advance()
		elseBlock, err := p.parseBlock()
		if err != nil {
			return nil, err
		}
		ifNode.Add(NewNode("Else", "", elseBlock))
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
		return NewNode("ShortVarDecl", name.Lexeme, expr), nil
	}

	if p.peek(1).Type == Operator && p.peek(1).Lexeme == "<-" {
		name := p.advance()
		p.advance()
		expr, err := p.parseExpression(0)
		if err != nil {
			return nil, err
		}
		return NewNode("SendStmt", "", NewNode("Channel", name.Lexeme), NewNode("Value", "", expr)), nil
	}

	left, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}
	if p.match(Operator, "=") {
		p.advance()
		right, err := p.parseExpression(0)
		if err != nil {
			return nil, err
		}
		return NewNode("AssignStmt", "", left, right), nil
	}
	return NewNode("ExprStmt", "", left), nil
}

func (p *Parser) parseFuncLiteral() (*ASTNode, error) {
	if _, err := p.expect(Keyword, "func", "ключевое слово func"); err != nil {
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
	return NewNode("FuncLiteral", "", params, body), nil
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
		left = NewNode("BinaryExpr", tok.Lexeme, left, right)
	}
	return left, nil
}

func (p *Parser) parsePrimary() (*ASTNode, error) {
	tok := p.current()
	var node *ASTNode

	switch tok.Type {
	case Identifier:
		p.advance()
		node = NewNode("Identifier", tok.Lexeme)
	case ConstInt, ConstFloat, ConstString, ConstBool:
		p.advance()
		node = NewNode("Literal", tok.Lexeme)
	case Keyword:
		if tok.Lexeme == "make" {
			p.advance()
			node = NewNode("Identifier", tok.Lexeme)
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
			node = NewNode("GroupedExpr", "", expr)
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
			node = NewNode("SelectorExpr", "", node, NewNode("Field", field.Lexeme))
		case p.match(Delimiter, "("):
			args, err := p.parseArguments()
			if err != nil {
				return nil, err
			}
			node = NewNode("CallExpr", "", node, args)
		default:
			return node, nil
		}
	}
}

func (p *Parser) parseArguments() (*ASTNode, error) {
	if _, err := p.expect(Delimiter, "(", "открывающая скобка аргументов ("); err != nil {
		return nil, err
	}
	args := NewNode("Args", "")

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
	fmt.Println("Положи очищенный lab1/output.txt рядом с проектом или запусти из папки lab3 внутри репозитория.")
	os.Exit(1)
	return ""
}

func printTokenSequence(tokens []Token) {
	fmt.Println("Поток токенов:")
	for _, token := range tokens {
		if token.Type == EOFToken {
			continue
		}
		fmt.Printf("(%s, %q, %d:%d)\n", token.Type, token.Lexeme, token.Line, token.Column)
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

	printTokenSequence(tokens)
	fmt.Println()

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

	fmt.Println("AST:")
	fmt.Println(ast.Kind)
	for i, child := range ast.Children {
		child.Print("", i == len(ast.Children)-1)
	}
	fmt.Println()
	fmt.Println("Синтаксический анализ завершён успешно. Ошибок не найдено.")
}
