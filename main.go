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
	for {
		char, _, err := reader.ReadRune()
		if err != nil {
			break
		}
		fmt.Printf("%c", char)
		if char == delimiterRunes[0] {
			bracketStack.Push(char)
		} else if char == delimiterRunes[1] {
			bracketStack.Pop()
		}
	}

	fmt.Printf("\nFinished reading the file named:%v", fileName)
}
