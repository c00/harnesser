package inputscan

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
)

func GetInput(ctx context.Context, prompt string) string {
	fmt.Print(prompt)

	inputCh := make(chan string)

	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		inputCh <- scanner.Text()
	}()

	select {
	case input := <-inputCh:
		return input
	case <-ctx.Done():
		return ""
	}
}

func GetYesNo(ctx context.Context, prompt string, defaultOption bool) bool {
	yesChar := "yes"
	noChar := "No"

	if defaultOption {
		yesChar = "Yes"
		noChar = "no"
	}

	for {
		prompt := fmt.Sprintf("%v [%v/%v]", prompt, yesChar, noChar)
		input := strings.ToLower(GetInput(ctx, prompt))
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
