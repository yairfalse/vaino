package main

import (
	"fmt"

	"github.com/yairfalse/vaino/internal/errors"
	"github.com/yairfalse/vaino/internal/output"
)

func main() {
	fmt.Println("VAINO Professional Output Test")
	fmt.Println("=============================\n")

	// Test 1: Error Display (should be clean, no emojis)
	fmt.Println("1. Error Display Test:")
	authErr := errors.GCPAuthenticationError(fmt.Errorf("could not find default credentials"))
	errors.DisplayError(authErr)

	// Test 2: Message Helpers (should use clean prefixes)
	fmt.Println("\n2. Message Helper Test:")
	errors.DisplayWarning("This is a warning message")
	errors.DisplaySuccess("This is a success message")
	errors.DisplayInfo("This is an info message")

	// Test 3: Renderer Output (should be professional)
	fmt.Println("\n3. Renderer Test:")
	config := output.Config{
		EnableColors: false,
		TimeFormat:   "2006-01-02 15:04:05",
	}
	renderer := output.NewRenderer(config)
	renderer.DisplayError(fmt.Errorf("sample error"))
	renderer.DisplaySuccess("Operation completed")
	renderer.DisplayWarning("Configuration incomplete")
	renderer.DisplayInfo("Additional information available")

	// Test 4: Exit Code Test
	fmt.Println("\n4. Exit Code Test:")
	fmt.Printf("Authentication Error Exit Code: %d\n", errors.GetExitCode(authErr))
	fmt.Printf("Network Error Exit Code: %d\n", errors.GetExitCode(errors.NetworkError(errors.ProviderAWS, "Connection failed")))
	fmt.Printf("Generic Error Exit Code: %d\n", errors.GetExitCode(fmt.Errorf("generic error")))

	fmt.Println("\n=============================")
	fmt.Println("Professional Output Test Complete")
	fmt.Println("All messages should be emoji-free")
	fmt.Println("and use clean text prefixes only.")
}
