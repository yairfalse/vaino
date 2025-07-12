package errors

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/viper"
)

// DisplayError formats and displays a VAINOError with enhanced formatting
func DisplayError(err error) {
	// Check if color should be disabled
	noColor := os.Getenv("NO_COLOR") != "" || os.Getenv("VAINO_NO_COLOR") != ""

	// Also check viper configuration (set by --no-color flag)
	if viperNoColor := getViperBool("output.no_color"); viperNoColor {
		noColor = true
	}

	color.NoColor = noColor

	vainoErr, ok := err.(*VAINOError)
	if !ok {
		// For non-VAINO errors, display a simple error message
		color.Red("Error: %v", err)
		return
	}

	// Choose color based on error type
	colorFunc := getErrorStyle(vainoErr.Type)

	// Error header
	fmt.Fprintf(os.Stderr, "\n%s\n", colorFunc(vainoErr.Message))

	// Cause with dimmed style
	if vainoErr.Cause != "" {
		fmt.Fprintf(os.Stderr, "   %s %s\n", color.YellowString("Cause:"), color.HiBlackString(vainoErr.Cause))
	}

	// Environment with dimmed style
	if vainoErr.Environment != "" {
		fmt.Fprintf(os.Stderr, "   %s %s\n", color.CyanString("Environment:"), color.HiBlackString(vainoErr.Environment))
	}

	// Solutions with numbered list
	if len(vainoErr.Solutions) > 0 {
		fmt.Fprintf(os.Stderr, "\n   %s\n", color.GreenString("Solutions:"))
		for i, solution := range vainoErr.Solutions {
			fmt.Fprintf(os.Stderr, "   %s %s\n", color.HiBlackString(fmt.Sprintf("%d.", i+1)), solution)
		}
	}

	// Verification command
	if vainoErr.Verify != "" {
		fmt.Fprintf(os.Stderr, "\n   %s %s\n", color.BlueString("Verify:"), color.HiWhiteString(vainoErr.Verify))
	}

	// Help command
	if vainoErr.Help != "" {
		fmt.Fprintf(os.Stderr, "   %s %s\n", color.MagentaString("Help:"), color.HiWhiteString(vainoErr.Help))
	}

	fmt.Fprintln(os.Stderr) // Final newline
}

// getErrorStyle returns the appropriate color function for an error type
func getErrorStyle(errType ErrorType) func(format string, a ...interface{}) string {
	switch errType {
	case ErrorTypeAuthentication:
		return color.RedString
	case ErrorTypeConfiguration:
		return color.YellowString
	case ErrorTypeProvider:
		return color.CyanString
	case ErrorTypeFileSystem:
		return color.MagentaString
	case ErrorTypeNetwork:
		return color.RedString
	case ErrorTypePermission:
		return color.RedString
	case ErrorTypeValidation:
		return color.YellowString
	default:
		return color.RedString
	}
}

// FormatErrorWithContext formats an error with additional context for CI/CD environments
func FormatErrorWithContext(err error, context map[string]string) string {
	var sb strings.Builder

	vainoErr, ok := err.(*VAINOError)
	if !ok {
		sb.WriteString(fmt.Sprintf("Error: %v\n", err))
		return sb.String()
	}

	// Main error without color for CI/CD
	sb.WriteString(fmt.Sprintf("Error: %s\n", vainoErr.Message))
	sb.WriteString(fmt.Sprintf("Type: %s/%s\n", vainoErr.Type, vainoErr.Provider))

	if vainoErr.Cause != "" {
		sb.WriteString(fmt.Sprintf("Cause: %s\n", vainoErr.Cause))
	}

	// Add context
	if len(context) > 0 {
		sb.WriteString("\nContext:\n")
		for k, v := range context {
			sb.WriteString(fmt.Sprintf("  %s: %s\n", k, v))
		}
	}

	// Solutions as plain text
	if len(vainoErr.Solutions) > 0 {
		sb.WriteString("\nSolutions:\n")
		for i, solution := range vainoErr.Solutions {
			sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, solution))
		}
	}

	if vainoErr.Verify != "" {
		sb.WriteString(fmt.Sprintf("\nVerify: %s\n", vainoErr.Verify))
	}

	if vainoErr.Help != "" {
		sb.WriteString(fmt.Sprintf("Help: %s\n", vainoErr.Help))
	}

	return sb.String()
}

// DisplayWarning shows a warning message with appropriate formatting
func DisplayWarning(message string) {
	noColor := os.Getenv("NO_COLOR") != "" || os.Getenv("VAINO_NO_COLOR") != ""
	color.NoColor = noColor

	fmt.Fprintf(os.Stderr, "Warning: %s\n", color.YellowString(message))
}

// DisplaySuccess shows a success message with appropriate formatting
func DisplaySuccess(message string) {
	noColor := os.Getenv("NO_COLOR") != "" || os.Getenv("VAINO_NO_COLOR") != ""
	color.NoColor = noColor

	fmt.Fprintf(os.Stderr, "Success: %s\n", color.GreenString(message))
}

// DisplayInfo shows an info message with appropriate formatting
func DisplayInfo(message string) {
	noColor := os.Getenv("NO_COLOR") != "" || os.Getenv("VAINO_NO_COLOR") != ""
	color.NoColor = noColor

	fmt.Fprintf(os.Stderr, "Info: %s\n", color.BlueString(message))
}

// getViperBool safely gets a boolean value from viper
func getViperBool(key string) bool {
	if viper.IsSet(key) {
		return viper.GetBool(key)
	}
	return false
}
