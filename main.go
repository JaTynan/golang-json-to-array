package main

import (
	"bufio"
	"fmt"
	"os"
)

func main() {

	// Opening the file
	fileName := "response.json"
	fmt.Printf("\nAttemtping to open file named:%v", fileName)
	file, err := os.Open(fileName)
	if err != nil {
		fmt.Printf("\nError opening file:%v", err)
		return
	}
	defer file.Close()

	// Reading the file
	reader := bufio.NewReader(file)
	for {
		char, _, err := reader.ReadRune()
		if err != nil {
			break
		}
		fmt.Printf("%c", char)
	}

	fmt.Printf("\nFinished reading the file named:%v", fileName)
}
