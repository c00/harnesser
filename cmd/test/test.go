package main

import (
	"fmt"

	"github.com/landlock-lsm/go-landlock/landlock"
)

// This package is just a random test thing. None of this is important. It is where we throw in some test code
// when we want to quickly prototype a thing.
func main() {
	err := landlock.V10.BestEffort().RestrictPaths(
		landlock.RODirs(),
	)

	if err != nil {
		fmt.Println("not fine: ", err)
	}
	fmt.Println("this is fine.")
}
