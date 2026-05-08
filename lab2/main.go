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
)

type Token struct {
	Type   TokenType
	Lexeme string
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

// Таблица ключевых слов.
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
}

// Таблица булевых констант.
var boolConstants = map[string]bool{
	"true":  true,
	"false": true,
}

// Таблица операторов.
var operators = map[string]bool{
	"=":  true,
	":=": true,
	"+":  true,
	"-":  true,
	"*":  true,
	"/":  true,
	"==": true,
	"<-": true,
	"<":  true,
	">":  true,
}

// Таблица разделителей.
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
	return &Lexer{
		input: []rune(input),
		line:  1,
		col:   1,
	}
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

// Пропуск пробелов, табуляций и переносов строк.
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

// Чтение идентификатора, ключевого слова или булевой константы.
func (l *Lexer) readIdentifier() Token {
	var builder strings.Builder

	for isIdentifierPart(l.current()) {
		builder.WriteRune(l.advance())
	}

	lexeme := builder.String()

	switch {
	case keywords[lexeme]:
		return Token{Type: Keyword, Lexeme: lexeme}
	case boolConstants[lexeme]:
		return Token{Type: ConstBool, Lexeme: lexeme}
	default:
		return Token{Type: Identifier, Lexeme: lexeme}
	}
}

// Чтение целочисленных и вещественных констант.
func (l *Lexer) readNumber() (Token, *LexError) {
	startLine := l.line
	startCol := l.col

	var builder strings.Builder
	hasDot := false

	for unicode.IsDigit(l.current()) {
		builder.WriteRune(l.advance())
	}

	if l.current() == '.' {
		if l.peek(1) == '.' {
			builder.WriteRune(l.advance())
			builder.WriteRune(l.advance())

			for !unicode.IsSpace(l.current()) && l.current() != 0 {
				builder.WriteRune(l.advance())
			}

			return Token{}, &LexError{
				Line:        startLine,
				Column:      startCol,
				ErrorType:   "Некорректное число",
				Explanation: fmt.Sprintf("числовая константа %q содержит две точки подряд", builder.String()),
			}
		}

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
		return Token{Type: ConstFloat, Lexeme: builder.String()}, nil
	}

	return Token{Type: ConstInt, Lexeme: builder.String()}, nil
}

// Чтение строковой константы.
func (l *Lexer) readString() (Token, *LexError) {
	startLine := l.line
	startCol := l.col

	quote := l.advance()

	var builder strings.Builder
	builder.WriteRune(quote)

	for l.current() != 0 {
		ch := l.current()

		if quote == '"' && ch == '\n' {
			return Token{}, &LexError{
				Line:        startLine,
				Column:      startCol,
				ErrorType:   "Незакрытая строка",
				Explanation: "строковый литерал не закрыт до конца строки",
			}
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
			return Token{Type: ConstString, Lexeme: builder.String()}, nil
		}
	}

	return Token{}, &LexError{
		Line:        startLine,
		Column:      startCol,
		ErrorType:   "Незакрытая строка",
		Explanation: "строковый литерал не закрыт до конца файла",
	}
}

// Чтение операторов и разделителей.
func (l *Lexer) readOperatorOrDelimiter() (Token, *LexError) {
	startLine := l.line
	startCol := l.col
	ch := l.current()

	if delimiters[ch] {
		l.advance()
		return Token{Type: Delimiter, Lexeme: string(ch)}, nil
	}

	two := string([]rune{ch, l.peek(1)})

	if operators[two] {
		l.advance()
		l.advance()
		return Token{Type: Operator, Lexeme: two}, nil
	}

	one := string(ch)

	if operators[one] {
		l.advance()
		return Token{Type: Operator, Lexeme: one}, nil
	}

	knownOperatorStart := strings.ContainsRune("+-*/%=!<>&|:^", ch)

	l.advance()

	if knownOperatorStart {
		return Token{}, &LexError{
			Line:        startLine,
			Column:      startCol,
			ErrorType:   "Неизвестный оператор",
			Explanation: fmt.Sprintf("оператор %q отсутствует в таблице операторов", one),
		}
	}

	return Token{}, &LexError{
		Line:        startLine,
		Column:      startCol,
		ErrorType:   "Недопустимый символ",
		Explanation: fmt.Sprintf("символ %q не может быть частью лексемы", one),
	}
}

// Главный метод лексического анализа.
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

	return tokens, errors
}

// Вывод таблицы лексем.
func printTokenTable(tokens []Token) {
	fmt.Println("Лексема              | Тип")
	fmt.Println("---------------------+----------------")

	for _, token := range tokens {
		fmt.Printf("%-20s | %s\n", token.Lexeme, token.Type)
	}
}

// Вывод последовательности токенов.
func printTokenSequence(tokens []Token) {
	fmt.Print("[")

	for i, token := range tokens {
		if i > 0 {
			fmt.Print(", ")
		}

		fmt.Printf("(%s, %q)", token.Type, token.Lexeme)
	}

	fmt.Println("]")
}

// Вывод ошибок.
func printErrors(errors []LexError) {
	for _, err := range errors {
		fmt.Printf(
			"Строка %d, позиция %d. %s: %s\n",
			err.Line,
			err.Column,
			err.ErrorType,
			err.Explanation,
		)
	}
}

// Загрузка очищенного кода из лабораторной работы №1.
func loadSource() string {
	paths := []string{
		"../lab1/output.txt",
		"lab1/output.txt",
	}

	for _, path := range paths {
		data, err := os.ReadFile(path)

		if err == nil {
			return string(data)
		}
	}

	fmt.Println("Ошибка: не удалось открыть файл lab1/output.txt")
	fmt.Println("Проверь, что файл output.txt лежит в папке lab1.")
	os.Exit(1)

	return ""
}

func main() {
	source := loadSource()

	lexer := NewLexer(source)

	tokens, errors := lexer.Scan()

	printTokenTable(tokens)

	fmt.Println()

	printTokenSequence(tokens)

	fmt.Println()

	if len(errors) == 0 {
		fmt.Printf(
			"Лексический анализ завершён успешно. Обнаружено %d токенов. Ошибок не найдено.\n",
			len(tokens),
		)
		return
	}

	fmt.Printf(
		"Лексический анализ завершён с ошибками. Обнаружено %d токенов и %d ошибок.\n",
		len(tokens),
		len(errors),
	)

	printErrors(errors)
}
