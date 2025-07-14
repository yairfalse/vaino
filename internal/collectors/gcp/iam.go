package gcp

import (
	"context"

	"github.com/yairfalse/vaino/pkg/types"
)

// GCP IAM Policy
type GCPIAMPolicy struct {
	Version  int                   `json:"version"`
	Bindings []GCPIAMPolicyBinding `json:"bindings"`
	Etag     string                `json:"etag"`
}

type GCPIAMPolicyBinding struct {
	Role      string                 `json:"role"`
	Members   []string               `json:"members"`
	Condition *GCPIAMPolicyCondition `json:"condition,omitempty"`
}

type GCPIAMPolicyCondition struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Expression  string `json:"expression"`
}

// GCP Service Account
type GCPServiceAccount struct {
	Name           string `json:"name"`
	ProjectID      string `json:"projectId"`
	UniqueID       string `json:"uniqueId"`
	Email          string `json:"email"`
	DisplayName    string `json:"displayName"`
	Description    string `json:"description"`
	OAuth2ClientID string `json:"oauth2ClientId"`
	Disabled       bool   `json:"disabled"`
	Etag           string `json:"etag"`
}

// GCP Service Account Key
type GCPServiceAccountKey struct {
	Name                string `json:"name"`
	PrivateKeyType      string `json:"privateKeyType"`
	KeyAlgorithm        string `json:"keyAlgorithm"`
	PrivateKeyData      string `json:"privateKeyData"`
	PublicKeyData       string `json:"publicKeyData"`
	ValidAfterTime      string `json:"validAfterTime"`
	ValidBeforeTime     string `json:"validBeforeTime"`
	KeyOrigin           string `json:"keyOrigin"`
	KeyType             string `json:"keyType"`
	ServiceAccountEmail string `json:"serviceAccountEmail"`
}

// GCP Custom Role
type GCPCustomRole struct {
	Name                string   `json:"name"`
	Title               string   `json:"title"`
	Description         string   `json:"description"`
	IncludedPermissions []string `json:"includedPermissions"`
	Stage               string   `json:"stage"`
	Etag                string   `json:"etag"`
	Deleted             bool     `json:"deleted"`
}

// collectIAMResources collects GCP IAM resources
func (c *GCPCollector) collectIAMResources(ctx context.Context, clientPool *GCPServicePool, projectID string, regions []string) ([]types.Resource, error) {
	var resources []types.Resource

	// Get project IAM policy (placeholder implementation)
	policy, err := clientPool.GetProjectIAMPolicy(ctx, projectID)
	if err == nil {
		// For now, return empty resources since the GCP collector is not fully implemented
		_ = policy
	}

	// Get service accounts (placeholder implementation)
	serviceAccounts, err := clientPool.GetServiceAccounts(ctx, projectID)
	if err == nil {
		// For now, return empty resources since the GCP collector is not fully implemented
		_ = serviceAccounts
	}

	// Get custom roles (placeholder implementation)
	customRoles, err := clientPool.GetCustomRoles(ctx, projectID)
	if err == nil {
		// For now, return empty resources since the GCP collector is not fully implemented
		_ = customRoles
	}

	return resources, nil
}
