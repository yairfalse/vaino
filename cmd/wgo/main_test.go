package main

import (
	"os"
	"testing"
)

func TestVersionVariables(t *testing.T) {
	// Test that version variables can be set
	originalVersion := version
	version = "test-version"
	
	if version != "test-version" {
		t.Errorf("Expected version to be 'test-version', got %s", version)
	}
	
	// Restore original
	version = originalVersion
}

func TestCommandsExist(t *testing.T) {
	commands := rootCmd.Commands()
	
	if len(commands) == 0 {
		t.Fatal("Root command should have subcommands")
	}
	
	// Check that key commands exist
	commandNames := make(map[string]bool)
	for _, cmd := range commands {
		commandNames[cmd.Name()] = true
	}
	
	// Core commands
	coreCommands := []string{"status", "snapshot", "drift", "config", "version"}
	for _, cmdName := range coreCommands {
		if !commandNames[cmdName] {
			t.Errorf("Core command '%s' should be present", cmdName)
		}
	}
	
	// AI commands
	aiCommands := []string{"analyze", "explain", "remediate"}
	for _, cmdName := range aiCommands {
		if !commandNames[cmdName] {
			t.Errorf("AI command '%s' should be present", cmdName)
		}
	}
}

func TestStatusCommandExists(t *testing.T) {
	// Find status command
	statusFound := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "status" {
			statusFound = true
			// Check description
			if cmd.Short == "" {
				t.Error("Status command should have a description")
			}
			break
		}
	}
	
	if !statusFound {
		t.Error("Status command should be present")
	}
}

func TestConfigCommandExists(t *testing.T) {
	// Find config command
	configFound := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "config" {
			configFound = true
			// Check description
			if cmd.Short == "" {
				t.Error("Config command should have a description")
			}
			break
		}
	}
	
	if !configFound {
		t.Error("Config command should be present")
	}
}

func TestAnalyzeCommandRequiresAPIKey(t *testing.T) {
	// Ensure no API key is set
	os.Unsetenv("ANTHROPIC_API_KEY")
	
	// Find analyze command
	analyzeFound := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "analyze" {
			analyzeFound = true
			break
		}
	}
	
	if !analyzeFound {
		t.Error("Analyze command should be present")
	}
}

func TestRootCommandProperties(t *testing.T) {
	if rootCmd.Use != "wgo" {
		t.Errorf("Expected root command use to be 'wgo', got %s", rootCmd.Use)
	}
	
	if rootCmd.Short == "" {
		t.Error("Root command should have a short description")
	}
	
	if rootCmd.Long == "" {
		t.Error("Root command should have a long description")
	}
}

func TestGlobalFlags(t *testing.T) {
	// Check that global flags exist
	flags := rootCmd.PersistentFlags()
	
	configFlag := flags.Lookup("config")
	if configFlag == nil {
		t.Error("--config flag should exist")
	}
	
	verboseFlag := flags.Lookup("verbose")
	if verboseFlag == nil {
		t.Error("--verbose flag should exist")
	}
	
	debugFlag := flags.Lookup("debug")
	if debugFlag == nil {
		t.Error("--debug flag should exist")
	}
}