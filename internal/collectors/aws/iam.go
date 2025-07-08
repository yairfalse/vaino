package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamTypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/yairfalse/wgo/pkg/types"
)

// CollectIAMResources collects basic IAM resources (roles, policies, users)
func (c *AWSCollector) CollectIAMResources(ctx context.Context) ([]types.Resource, error) {
	var resources []types.Resource
	
	// Collect IAM roles
	roles, err := c.collectIAMRoles(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect IAM roles: %w", err)
	}
	resources = append(resources, roles...)
	
	// Collect IAM users (basic)
	users, err := c.collectIAMUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect IAM users: %w", err)
	}
	resources = append(resources, users...)
	
	return resources, nil
}

// collectIAMRoles fetches IAM roles
func (c *AWSCollector) collectIAMRoles(ctx context.Context) ([]types.Resource, error) {
	var resources []types.Resource
	var marker *string
	
	for {
		input := &iam.ListRolesInput{}
		if marker != nil {
			input.Marker = marker
		}
		
		result, err := c.clients.IAM.ListRoles(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to list IAM roles: %w", err)
		}
		
		// Process IAM roles
		for _, role := range result.Roles {
			resource := c.normalizeIAMRole(role)
			
			// Try to get role tags
			if tags, err := c.getIAMRoleTags(ctx, *role.RoleName); err == nil {
				resource.Tags = tags
			}
			
			resources = append(resources, resource)
		}
		
		// Check if there are more results
		marker = result.Marker
		if marker == nil {
			break
		}
	}
	
	return resources, nil
}

// collectIAMUsers fetches IAM users (basic information only)
func (c *AWSCollector) collectIAMUsers(ctx context.Context) ([]types.Resource, error) {
	var resources []types.Resource
	var marker *string
	
	for {
		input := &iam.ListUsersInput{}
		if marker != nil {
			input.Marker = marker
		}
		
		result, err := c.clients.IAM.ListUsers(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to list IAM users: %w", err)
		}
		
		// Process IAM users
		for _, user := range result.Users {
			resource := c.normalizeIAMUser(user)
			
			// Try to get user tags
			if tags, err := c.getIAMUserTags(ctx, *user.UserName); err == nil {
				resource.Tags = tags
			}
			
			resources = append(resources, resource)
		}
		
		// Check if there are more results
		marker = result.Marker
		if marker == nil {
			break
		}
	}
	
	return resources, nil
}

// normalizeIAMRole converts an IAM role to WGO format
func (c *AWSCollector) normalizeIAMRole(role iamTypes.Role) types.Resource {
	return types.Resource{
		ID:       aws.ToString(role.Arn),
		Type:     "aws_iam_role",
		Provider: "aws",
		Name:     aws.ToString(role.RoleName),
		Region:   "global", // IAM is global
		Configuration: map[string]interface{}{
			"name":                      aws.ToString(role.RoleName),
			"path":                      aws.ToString(role.Path),
			"assume_role_policy":        aws.ToString(role.AssumeRolePolicyDocument),
			"description":               aws.ToString(role.Description),
			"max_session_duration":      aws.ToInt32(role.MaxSessionDuration),
			"permissions_boundary_arn":  aws.ToString(role.PermissionsBoundary.PermissionsBoundaryArn),
		},
		Tags: make(map[string]string), // Tags will be fetched separately
		Metadata: types.ResourceMetadata{
			CreatedAt: aws.ToTime(role.CreateDate),
		},
	}
}

// normalizeIAMUser converts an IAM user to WGO format
func (c *AWSCollector) normalizeIAMUser(user iamTypes.User) types.Resource {
	return types.Resource{
		ID:       aws.ToString(user.Arn),
		Type:     "aws_iam_user",
		Provider: "aws",
		Name:     aws.ToString(user.UserName),
		Region:   "global", // IAM is global
		Configuration: map[string]interface{}{
			"name":                     aws.ToString(user.UserName),
			"path":                     aws.ToString(user.Path),
			"permissions_boundary_arn": aws.ToString(user.PermissionsBoundary.PermissionsBoundaryArn),
		},
		Tags: make(map[string]string), // Tags will be fetched separately
		Metadata: types.ResourceMetadata{
			CreatedAt: aws.ToTime(user.CreateDate),
		},
	}
}

// getIAMRoleTags fetches tags for an IAM role
func (c *AWSCollector) getIAMRoleTags(ctx context.Context, roleName string) (map[string]string, error) {
	result, err := c.clients.IAM.ListRoleTags(ctx, &iam.ListRoleTagsInput{
		RoleName: &roleName,
	})
	if err != nil {
		return make(map[string]string), nil
	}
	
	tags := make(map[string]string)
	for _, tag := range result.Tags {
		if tag.Key != nil && tag.Value != nil {
			tags[*tag.Key] = *tag.Value
		}
	}
	
	return tags, nil
}

// getIAMUserTags fetches tags for an IAM user
func (c *AWSCollector) getIAMUserTags(ctx context.Context, userName string) (map[string]string, error) {
	result, err := c.clients.IAM.ListUserTags(ctx, &iam.ListUserTagsInput{
		UserName: &userName,
	})
	if err != nil {
		return make(map[string]string), nil
	}
	
	tags := make(map[string]string)
	for _, tag := range result.Tags {
		if tag.Key != nil && tag.Value != nil {
			tags[*tag.Key] = *tag.Value
		}
	}
	
	return tags, nil
}