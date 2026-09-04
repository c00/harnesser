package inputscan

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func GetInput(prompt string) string {
	fmt.Print(prompt)

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	input := scanner.Text()

	return input
}

func GetYesNo(prompt string, defaultOption bool) bool {
	yesChar := "yes"
	noChar := "No"

	if defaultOption {
		yesChar = "Yes"
		noChar = "no"
	}

	for {
		prompt := fmt.Sprintf("%v [%v/%v]", prompt, yesChar, noChar)
		input := strings.ToLower(GetInput(prompt))
		if input == "" {
			return defaultOption
		}

		if input == "y" || input == "yes" {
			return true
		}
		if input == "n" || input == "no" {
			return false
		}

		fmt.Println("Please enter either yes or no")
	}

}
