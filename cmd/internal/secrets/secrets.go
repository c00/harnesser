package secrets

import (
	"context"
	"fmt"
	"os"

	"github.com/c00/harnesser/internal/inputscan"
	"github.com/zalando/go-keyring"
)

const (
	Service           = "harnesser"
	OpenrouterKeyName = "openrouter-api-key"
)

func HasSecret(name string) bool {
	val, err := keyring.Get(Service, name)
	if err != nil || val == "" {
		return false
	}

	return true
}

func SetSecret(name string, value string) error {
	err := keyring.Set(Service, name, value)
	if err != nil {
		return fmt.Errorf("cannot set secret %v: %w", name, err)
	}
	return nil
}

func GetSecret(name string) (string, error) {
	val, err := keyring.Get(Service, name)
	if err != nil {
		return "", fmt.Errorf("cannot get value: %w", err)
	}
	return val, nil
}

func ApiKeySetup(ctx context.Context) error {
	apiKeyFromKeyringExists := HasSecret(OpenrouterKeyName)
	apiKeyFromEnv := os.Getenv("OPENROUTER_API_KEY")

	fmt.Printf("The API key for openrouter will be stored in your keyring.\n")

	if apiKeyFromKeyringExists {
		fmt.Println("API key is already set in keyring.")
		doUpdate := inputscan.GetYesNo(ctx, "Do you want to update it?", true)
		if !doUpdate {
			return nil
		}
		// Fallthrough to the setup
	} else if apiKeyFromEnv != "" {
		keySnippet := apiKeyFromEnv[:5] + "***" + apiKeyFromEnv[len(apiKeyFromEnv)-4:]
		fmt.Printf("We found an API key in the env: %v. \n", keySnippet)
		use := inputscan.GetYesNo(ctx, "Would you like to use this?", true)
		if use {
			err := SetSecret(OpenrouterKeyName, apiKeyFromEnv)
			if err != nil {
				return fmt.Errorf("cannot store api key to keyring : %w", err)
			}
			return nil
		}
	}

	val := inputscan.GetInput(ctx, "Openrouter API key: ")
	if val == "" {
		return fmt.Errorf("API key cannot be empty")
	}

	err := SetSecret(OpenrouterKeyName, val)
	if err != nil {
		return fmt.Errorf("cannot store api key to keyring : %w", err)
	}
	return nil
}
