package main

import (
	"fmt"
	"sync"
)

/*
Test code for lab work
There some kind of code on Golang
*/

func division(num1 int, num2 int, results chan int) {
	result := num1 / num2 // Arithmetic operation
	results <- result
}

func main() {
	var wg sync.WaitGroup // Variable declaration
	number1 := 6
	number2 := 3
	results := make(chan int)
	for w := range 10 { // loop
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			if number2 == 0 {
				return
			} else {
				division(number1, number2, results) // function call
				return
			}
		}(w)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	for result := range results {
		fmt.Println(result)
	}
}
