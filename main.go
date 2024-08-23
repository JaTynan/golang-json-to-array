package main

import (
	"bufio"
	"fmt"
	"os"
)

type stack struct {
	items []interface{}
}

func (s *stack) Push(v interface{}) {
	s.items = append(s.items, v)
}
func (s *stack) Pop() interface{} {
	if len(s.items) == 0 {
		return nil
	}
	last := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return last
}
func (s *stack) Peek() interface{} {
	if len(s.items) == 0 {
		return nil
	}
	return s.items[len(s.items)-1]
}
func (s *stack) IsEmpty() bool {
	return len(s.items) == 0
}

func main() {

	// Array to hold json file contents
	var items [100][100][100]any
	itemsX := 0
	itemsY := 0
	itemsZ := 0

	// Opening the file
	fileName := "response.json"
	fmt.Printf("\nAttempting to open file named:%v\n", fileName)
	file, err := os.Open(fileName)
	if err != nil {
		fmt.Printf("\nError opening file:%v\n", err)
		return
	}
	defer file.Close()

	// Reading the file
	reader := bufio.NewReader(file)
	var bracketStack stack
	//  {==0  }==1  [==2  ]==3 ,==4  :==5  "==6
	delimiterCharacters := "{}[],:\""
	delimiterRunes := []rune(delimiterCharacters)
	fmt.Printf("\nDelimiter Characters: %v\n", delimiterCharacters)

	quotationOuterCount := 0
	quotationInnerCount := 0
	quotationInnerInnerCount := 0
	for {
		// check for when we hit the end of the file, run out of characters to process
		char, _, err := reader.ReadRune()
		if err != nil {
			break
		}
		// This is where we process the characters and set the array
		// Start by handling the depth
		if char == delimiterRunes[0] || char == delimiterRunes[2] {
			bracketStack.Push(char)
		} else if char == delimiterRunes[1] || char == delimiterRunes[3] {
			bracketStack.Pop()
		}
		if char == delimiterRunes[6] {
			quotationOuterCount += 1
			quotationInnerCount += 1
			quotationInnerInnerCount += 1
		}
		itemsX = len(bracketStack.items)
		itemsY = 0
		itemsZ = 0
		items[itemsX][itemsY][itemsZ] = string(char)
	}
	fmt.Printf("\nFinished reading the file named:%v", fileName)
	fmt.Printf("\nArray from JSON built:\n%v", items)
}
