package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newExplainCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "explain",
		Short: "Get AI analysis of infrastructure drift",
		Long: `Get detailed AI-powered analysis and recommendations for infrastructure
drift, changes, or specific resources. Uses Claude AI to provide insights
about root causes, risks, and remediation steps.`,
		Example: `  # Explain latest drift report
  vaino explain

  # Explain specific drift report
  vaino explain --drift-report drift-report-123.json

  # Explain changes in a resource
  vaino explain --resource ec2-instance-123 --provider aws

  # Get security-focused analysis
  vaino explain --focus security --drift-report latest

  # Save analysis to file
  vaino explain --drift-report latest --output-file analysis.md --format markdown`,
		RunE: runExplain,
	}

	// Flags
	cmd.Flags().String("drift-report", "", "drift report file to analyze")
	cmd.Flags().String("resource", "", "specific resource ID to analyze")
	cmd.Flags().StringP("provider", "p", "", "resource provider (required with --resource)")
	cmd.Flags().String("focus", "", "analysis focus (security, cost, performance, compliance)")
	cmd.Flags().String("output-file", "", "save analysis to file")
	cmd.Flags().String("format", "text", "output format (text, markdown, json)")
	cmd.Flags().Bool("include-remediation", true, "include remediation suggestions")
	cmd.Flags().Bool("include-risk-assessment", true, "include risk assessment")
	cmd.Flags().String("context", "", "additional context for analysis")

	return cmd
}

func runExplain(cmd *cobra.Command, args []string) error {
	fmt.Println("🤖 AI Infrastructure Analysis")
	fmt.Println("=============================")

	driftReport, _ := cmd.Flags().GetString("drift-report")
	resource, _ := cmd.Flags().GetString("resource")
	provider, _ := cmd.Flags().GetString("provider")
	focus, _ := cmd.Flags().GetString("focus")
	outputFile, _ := cmd.Flags().GetString("output-file")
	format, _ := cmd.Flags().GetString("format")
	includeRemediation, _ := cmd.Flags().GetBool("include-remediation")
	includeRisk, _ := cmd.Flags().GetBool("include-risk-assessment")

	if resource != "" && provider == "" {
		return fmt.Errorf("--provider is required when using --resource")
	}

	if driftReport == "" && resource == "" {
		fmt.Println("Analyzing latest drift report")
	} else if driftReport != "" {
		fmt.Printf("Drift report: %s\n", driftReport)
	} else {
		fmt.Printf("🔧 Resource: %s (%s)\n", resource, provider)
	}

	if focus != "" {
		fmt.Printf("Analysis focus: %s\n", focus)
	}

	if outputFile != "" {
		fmt.Printf("Output file: %s (%s)\n", outputFile, format)
	}

	fmt.Printf("Include remediation: %v\n", includeRemediation)
	fmt.Printf("Include risk assessment: %v\n", includeRisk)

	fmt.Println("\nAI explanation not yet implemented")
	fmt.Println("This command will:")
	fmt.Println("  • Analyze drift data with Claude AI")
	fmt.Println("  • Explain changes in plain language")
	fmt.Println("  • Assess risks and impact")
	fmt.Println("  • Suggest remediation steps")
	fmt.Println("  • Provide compliance insights")

	// Check if Claude API key is configured
	config := GetConfig()
	if config != nil && config.Claude.APIKey == "" {
		fmt.Println("\nTo enable AI features:")
		fmt.Println("   Set CLAUDE_API_KEY or ANTHROPIC_API_KEY environment variable")
		fmt.Println("   Or configure it in ~/.vaino/config.yaml")
	}

	return nil
}
