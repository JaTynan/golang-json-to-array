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
	//delimiterCharacters := "{}[],:\""
	//delimiterRunes := []rune(delimiterCharacters)
	//fmt.Printf("\nDelimiter Characters: %v\n", delimiterCharacters)

	// Create the slice of slices, counting rows/columns.
	items := [3][20][20][20]string{}
	itemsOuter := -1
	itemsMid := 0
	itemsInn := 0
	itemsSubInn := 0
	bracketStackSize := 0
	var previousChar rune
	for {
		// check for when we hit the end of the file, run out of characters to process
		char, _, err := reader.ReadRune()
		if err != nil {
			break
		}
		// This is where we process the characters and set the array
		// Start by handling the depth
		// 123=={ 91==[
		if char == 123 || char == 91 {
			bracketStack.Push(char)
			bracketStackSize = len(bracketStack.items)
			switch bracketStackSize {
			case 1:
				itemsOuter++
			case 2:
				itemsMid++
			case 3:
				itemsInn++
			case 4:
				itemsSubInn++
			}
			// 125==} 93==]
		} else if char == 125 || char == 93 {
			bracketStack.Pop()
			bracketStackSize = len(bracketStack.items)
			switch bracketStackSize {
			case 1:
				itemsOuter--
			case 2:
				itemsMid--
			case 3:
				itemsInn--
			case 4:
				itemsSubInn--
			}
			// 10=="\n" 13=="\r" 34=="\"" 44=="," 58==":" 91=="[" 93=="]" 123==" "
		} else if char == 58 && previousChar != 123 && previousChar != 91 {
			switch bracketStackSize {
			case 1:
				itemsMid++
			case 2:
				itemsInn++
			case 3:
				itemsSubInn++
			}
		} else if char == 44 {
			switch bracketStackSize {
			case 1:
				itemsMid--
				itemsInn++
			case 2:
				itemsInn--
				itemsSubInn++
			case 3:
				itemsSubInn--
			}
		} else if char != 34 && char != 13 && char != 10 && char != 32 {
			items[itemsOuter][itemsMid][itemsInn][itemsSubInn] += string(char)
		}
		previousChar = char
	}

	fmt.Printf("\nFinished reading the file named:%v", fileName)
	fmt.Printf("\nArray from JSON built:\n%v", items)

}
