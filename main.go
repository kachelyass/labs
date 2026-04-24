package main

import (
	"fmt"
	"lab/processor"
)

func main() {
	// Имя исходного файла, который нужно обработать.
	// В этом файле может быть свой package main и func main(),
	// потому что мы читаем его как обычный текст, а не компилируем.
	inputFile := "test.go"

	// Имя файла, куда будет сохранен очищенный код.
	outputFile := "output.txt"

	// Запускаем обработку файла через модуль processor.
	messages, err := processor.ProcessFile(inputFile, outputFile)

	// Выводим все информационные сообщения и ошибки,
	// которые были получены во время обработки.
	for _, msg := range messages {
		fmt.Println(msg)
	}

	// Если во время обработки произошла ошибка,
	// выводим ее и завершаем программу.
	if err != nil {
		fmt.Println("ERROR:", err)
		return
	}

	// Сообщаем, что результат успешно сохранен.
	fmt.Println("INFO: результат сохранен в файл", outputFile)
}
