package gcp

import (
	"os"
)

// Collector implements the collectors.Collector interface for GCP
type Collector struct {
	projectID       string
	credentialsFile string
}

// NewGCPCollector creates a new GCP collector
func NewGCPCollector() *Collector {
	return &Collector{}
}

// Name returns the collector name
func (c *Collector) Name() string {
	return "gcp"
}

// Status returns the current status of the collector
func (c *Collector) Status() string {
	// Check for project ID first
	projectID := c.projectID
	if projectID == "" {
		projectID = os.Getenv("GOOGLE_CLOUD_PROJECT")
		if projectID == "" {
			projectID = os.Getenv("GCP_PROJECT")
		}
	}
	
	if projectID == "" {
		return "no project ID configured (set GOOGLE_CLOUD_PROJECT)"
	}
	
	// Check for credentials
	credentialsFile := c.credentialsFile
	if credentialsFile == "" {
		credentialsFile = os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	}
	
	if credentialsFile != "" {
		if _, err := os.Stat(credentialsFile); err != nil {
			return "credentials file not found"
		}
	}
	
	return "ready"
}

// SetProjectID sets the project ID for the collector
func (c *Collector) SetProjectID(projectID string) {
	c.projectID = projectID
}

// SetCredentialsFile sets the credentials file path
func (c *Collector) SetCredentialsFile(credentialsFile string) {
	c.credentialsFile = credentialsFile
}