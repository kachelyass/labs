package processor

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// ProcessFile читает исходный файл, очищает его содержимое
// и сохраняет результат в новый файл.
func ProcessFile(inputFile string, outputFile string) ([]string, error) {
	// Читаем содержимое исходного файла.
	sourceBytes, err := os.ReadFile(inputFile)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать файл %s: %w", inputFile, err)
	}

	// Очищаем исходный код от комментариев и лишних пробелов.
	cleaned, messages, hasErrors := CleanSource(string(sourceBytes))

	// Если во время обработки были найдены ошибки,
	// например незакрытый многострочный комментарий,
	// то файл с результатом не сохраняем.
	if hasErrors {
		return messages, fmt.Errorf("обработка остановлена из-за ошибок")
	}

	// Записываем очищенный код в выходной файл.
	err = os.WriteFile(outputFile, []byte(cleaned), 0644)
	if err != nil {
		return messages, fmt.Errorf("не удалось записать файл %s: %w", outputFile, err)
	}

	return messages, nil
}

// CleanSource выполняет основную обработку исходного кода:
// удаляет комментарии, лишние пробелы, табуляции и пустые строки.
func CleanSource(source string) (string, []string, bool) {
	var messages []string
	hasErrors := false

	// Приводим переводы строк к единому формату.
	source = normalizeNewlines(source)

	// Временно заменяем строковые литералы на специальные метки.
	// Это нужно, чтобы случайно не удалить комментарии внутри строк.
	maskedSource, literals := maskStringLiterals(source)

	// Проверяем многострочные комментарии на ошибки.
	blockErrors := checkBlockComments(maskedSource)
	if len(blockErrors) > 0 {
		messages = append(messages, blockErrors...)
		return "", messages, true
	}

	// Регулярное выражение для многострочных комментариев вида /* ... */.
	blockCommentRe := regexp.MustCompile(`(?s)/\*.*?\*/`)

	// Регулярное выражение для однострочных комментариев вида // ...
	lineCommentRe := regexp.MustCompile(`(?m)//[^\n]*`)

	// Считаем количество найденных комментариев.
	blockCommentCount := len(blockCommentRe.FindAllStringIndex(maskedSource, -1))
	lineCommentCount := len(lineCommentRe.FindAllStringIndex(maskedSource, -1))

	// Удаляем многострочные комментарии.
	// Если комментарий занимал несколько строк, оставляем один перевод строки,
	// чтобы части кода случайно не склеились.
	cleaned := blockCommentRe.ReplaceAllStringFunc(maskedSource, func(comment string) string {
		if strings.Contains(comment, "\n") {
			return "\n"
		}

		return " "
	})

	// Удаляем однострочные комментарии.
	cleaned = lineCommentRe.ReplaceAllString(cleaned, "")

	// Проверяем наличие недопустимых символов.
	invalidMessages := findInvalidCharacters(cleaned)
	if len(invalidMessages) > 0 {
		messages = append(messages, invalidMessages...)
		hasErrors = true
	}

	// Удаляем лишние пробельные символы и пустые строки.
	cleaned = cleanWhitespace(cleaned)

	// Возвращаем строковые литералы обратно.
	cleaned = restoreStringLiterals(cleaned, literals)

	// Добавляем информационные сообщения.
	messages = append(messages,
		fmt.Sprintf("INFO: удалено многострочных комментариев: %d", blockCommentCount),
		fmt.Sprintf("INFO: удалено однострочных комментариев: %d", lineCommentCount),
		"INFO: удалены лишние пробельные символы и пустые строки",
	)

	return cleaned, messages, hasErrors
}

// normalizeNewlines заменяет разные варианты перевода строк
// на стандартный символ \n.
func normalizeNewlines(source string) string {
	source = strings.ReplaceAll(source, "\r\n", "\n")
	source = strings.ReplaceAll(source, "\r", "\n")

	return source
}

// maskStringLiterals временно заменяет строковые и символьные литералы.
// Например, строка "hello // comment" не должна восприниматься как комментарий.
func maskStringLiterals(source string) (string, []string) {
	// Регулярное выражение находит:
	// 1. raw-строки в обратных кавычках `...`
	// 2. обычные строки "..."
	// 3. символьные литералы 'a'
	literalRe := regexp.MustCompile("(?s)`[^`]*`|\"(?:\\\\.|[^\"\\\\])*\"|'(?:\\\\.|[^'\\\\])+'")

	var literals []string

	// Каждую найденную строку заменяем на метку вида __LITERAL_0__.
	masked := literalRe.ReplaceAllStringFunc(source, func(literal string) string {
		placeholder := fmt.Sprintf("__LITERAL_%d__", len(literals))
		literals = append(literals, literal)

		return placeholder
	})

	return masked, literals
}

// restoreStringLiterals возвращает строковые литералы обратно
// вместо временных меток.
func restoreStringLiterals(source string, literals []string) string {
	for i, literal := range literals {
		placeholder := fmt.Sprintf("__LITERAL_%d__", i)
		source = strings.ReplaceAll(source, placeholder, literal)
	}

	return source
}

// checkBlockComments проверяет корректность многострочных комментариев.
// Например, сообщает об ошибке, если комментарий /* открыт, но не закрыт.
func checkBlockComments(source string) []string {
	var messages []string

	// Ищем только начало и конец многострочного комментария.
	tokenRe := regexp.MustCompile(`/\*|\*/`)
	tokens := tokenRe.FindAllStringIndex(source, -1)

	inBlockComment := false
	startIndex := -1

	for _, token := range tokens {
		value := source[token[0]:token[1]]

		// Найдено начало многострочного комментария.
		if value == "/*" && !inBlockComment {
			inBlockComment = true
			startIndex = token[0]
			continue
		}

		// Найден конец многострочного комментария.
		if value == "*/" && inBlockComment {
			inBlockComment = false
			startIndex = -1
			continue
		}

		// Найдено закрытие комментария без открытия.
		if value == "*/" && !inBlockComment {
			line, column := position(source, token[0])
			messages = append(messages,
				fmt.Sprintf("ERROR [%d:%d]: найдено закрытие многострочного комментария без открытия", line, column),
			)
		}
	}

	// Если после проверки мы все еще внутри комментария,
	// значит многострочный комментарий не был закрыт.
	if inBlockComment {
		line, column := position(source, startIndex)
		messages = append(messages,
			fmt.Sprintf("ERROR [%d:%d]: незакрытый многострочный комментарий", line, column),
		)
	}

	return messages
}

// cleanWhitespace удаляет лишние пробельные символы:
// пробелы и табуляции в начале и конце строк,
// повторяющиеся пробелы и пустые строки.
func cleanWhitespace(source string) string {
	// Удаляем пробелы и табуляции в начале и конце каждой строки.
	trimLineRe := regexp.MustCompile(`(?m)^[ \t]+|[ \t]+$`)

	// Заменяем несколько пробелов или табуляций подряд на один пробел.
	multipleSpacesRe := regexp.MustCompile(`[ \t]{2,}`)

	// Удаляем пустые строки.
	emptyLinesRe := regexp.MustCompile(`(?m)^\s*\n`)

	source = trimLineRe.ReplaceAllString(source, "")
	source = multipleSpacesRe.ReplaceAllString(source, " ")
	source = emptyLinesRe.ReplaceAllString(source, "")

	// Удаляем лишние пробелы и переводы строк в начале и конце всего текста.
	source = strings.TrimSpace(source)

	// Добавляем перевод строки в конец файла.
	if source != "" {
		source += "\n"
	}

	return source
}

// findInvalidCharacters ищет символы, которые не входят
// в допустимый набор символов для простого Go-кода.
func findInvalidCharacters(source string) []string {
	var messages []string

	// Разрешены:
	// буквы, цифры, пробельные символы, знаки операций,
	// скобки, точки, запятые, двоеточия и другие базовые символы Go.
	invalidCharRe := regexp.MustCompile(`[^\pL\pN\s_+\-*/%=&|!<>:;.,(){}\[\]^]`)
	invalids := invalidCharRe.FindAllStringIndex(source, -1)

	for _, invalid := range invalids {
		line, column := position(source, invalid[0])
		character := source[invalid[0]:invalid[1]]

		messages = append(messages,
			fmt.Sprintf("ERROR [%d:%d]: недопустимый символ: %q", line, column, character),
		)
	}

	return messages
}

// position вычисляет номер строки и столбца по индексу символа.
// Это нужно для понятного вывода ошибок.
func position(source string, byteIndex int) (int, int) {
	line := 1
	column := 1

	for i, r := range source {
		if i >= byteIndex {
			break
		}

		if r == '\n' {
			line++
			column = 1
		} else {
			column++
		}
	}

	return line, column
}
