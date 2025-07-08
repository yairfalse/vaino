package gcp

import (
	"os"
	"testing"
)

func TestGCPCollector_Name(t *testing.T) {
	collector := NewGCPCollector()
	if collector.Name() != "gcp" {
		t.Errorf("Expected Name() to return 'gcp', got %s", collector.Name())
	}
}

func TestGCPCollector_Status(t *testing.T) {
	collector := NewGCPCollector()
	
	// Clear environment variables for clean test
	oldProject := os.Getenv("GOOGLE_CLOUD_PROJECT")
	oldCreds := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	os.Unsetenv("GOOGLE_CLOUD_PROJECT")
	os.Unsetenv("GCP_PROJECT")
	os.Unsetenv("GOOGLE_APPLICATION_CREDENTIALS")
	
	defer func() {
		if oldProject != "" {
			os.Setenv("GOOGLE_CLOUD_PROJECT", oldProject)
		}
		if oldCreds != "" {
			os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", oldCreds)
		}
	}()
	
	// Without project ID, should return error status
	status := collector.Status()
	if status == "ready" {
		t.Error("Expected Status() to return error without project ID")
	}
	
	// Set environment variable and test again
	os.Setenv("GOOGLE_CLOUD_PROJECT", "test-project")
	
	status = collector.Status()
	if status != "ready" {
		t.Errorf("Expected Status() to return 'ready' with project ID, got %s", status)
	}
}

func TestGCPCollector_SetProjectID(t *testing.T) {
	collector := NewGCPCollector()
	collector.SetProjectID("test-project")
	
	if collector.projectID != "test-project" {
		t.Errorf("Expected projectID to be 'test-project', got %s", collector.projectID)
	}
	
	// Status should now be ready
	status := collector.Status()
	if status != "ready" {
		t.Errorf("Expected Status() to return 'ready' after setting project ID, got %s", status)
	}
}

func TestGCPCollector_SetCredentialsFile(t *testing.T) {
	collector := NewGCPCollector()
	collector.SetCredentialsFile("/path/to/credentials.json")
	
	if collector.credentialsFile != "/path/to/credentials.json" {
		t.Errorf("Expected credentialsFile to be '/path/to/credentials.json', got %s", collector.credentialsFile)
	}
}