package main

import (
	"fmt"
	"os/exec"

	"github.com/yairfalse/vaino/pkg/config"
)

func main() {
	fmt.Println("Testing kubectl detection")
	fmt.Println("========================\n")

	// First check if kubectl is actually available
	fmt.Println("1. Direct kubectl check:")
	cmd := exec.Command("kubectl", "version", "--client")
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("   kubectl command failed: %v\n", err)
	} else {
		fmt.Printf("   kubectl output: %s\n", string(output))
	}

	// Test the detector
	fmt.Println("\n2. VAINO detector test:")
	detector := config.NewProviderDetector()
	result := detector.DetectKubernetes()

	fmt.Printf("   Available: %v\n", result.Available)
	fmt.Printf("   Status: %s\n", result.Status)
	fmt.Printf("   Version: %s\n", result.Version)

	if result.Available {
		fmt.Println("\n✓ kubectl detection is working correctly!")
	} else {
		fmt.Println("\n✗ kubectl detection failed")
	}
}
