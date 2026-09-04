package main

import (
	"fmt"

	"github.com/c00/harnesser/internal/inputscan"
)

// This package is just a random test thing. None of this is important. It is where we throw in some test code
// when we want to quickly prototype a thing.
func main() {
	fmt.Print("You: ")
	foo := inputscan.GetYesNo("Do you like beans?", true)

	foo2 := inputscan.GetYesNo("Do you like cheese?", false)

	fmt.Println("Answers", foo, foo2)
}
