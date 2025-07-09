package types

import (
	"testing"
)

func TestResource_Validate(t *testing.T) {
	tests := []struct {
		name     string
		resource Resource
		wantErr  bool
	}{
		{
			name: "valid resource",
			resource: Resource{
				ID:            "i-1234567890abcdef0",
				Type:          "ec2:instance",
				Provider:      "aws",
				Name:          "web-server",
				Region:        "us-west-2",
				Configuration: map[string]interface{}{"instance_type": "t3.micro"},
				Tags:          map[string]string{"Environment": "production"},
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			resource: Resource{
				Type:     "ec2:instance",
				Provider: "aws",
				Name:     "web-server",
				Region:   "us-west-2",
			},
			wantErr: true,
		},
		{
			name: "missing type",
			resource: Resource{
				ID:       "i-1234567890abcdef0",
				Provider: "aws",
				Name:     "web-server",
				Region:   "us-west-2",
			},
			wantErr: true,
		},
		{
			name: "missing provider",
			resource: Resource{
				ID:     "i-1234567890abcdef0",
				Type:   "ec2:instance",
				Name:   "web-server",
				Region: "us-west-2",
			},
			wantErr: true,
		},
		{
			name: "empty ID",
			resource: Resource{
				ID:       "",
				Type:     "ec2:instance",
				Provider: "aws",
				Name:     "web-server",
				Region:   "us-west-2",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.resource.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Resource.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestResource_String(t *testing.T) {
	resource := Resource{
		ID:       "i-1234567890abcdef0",
		Type:     "ec2:instance",
		Provider: "aws",
		Name:     "web-server",
		Region:   "us-west-2",
	}

	str := resource.String()
	expected := "aws:ec2:instance:i-1234567890abcdef0 (web-server) in us-west-2"

	if str != expected {
		t.Errorf("Resource.String() = %s, want %s", str, expected)
	}
}

func TestResource_StringWithoutName(t *testing.T) {
	resource := Resource{
		ID:       "i-1234567890abcdef0",
		Type:     "ec2:instance",
		Provider: "aws",
		Region:   "us-west-2",
	}

	str := resource.String()
	expected := "aws:ec2:instance:i-1234567890abcdef0 in us-west-2"

	if str != expected {
		t.Errorf("Resource.String() = %s, want %s", str, expected)
	}
}

func TestResource_StringWithNamespace(t *testing.T) {
	resource := Resource{
		ID:        "pod-1234",
		Type:      "pod",
		Provider:  "kubernetes",
		Name:      "api-server",
		Namespace: "production",
	}

	str := resource.String()
	expected := "kubernetes:pod:pod-1234 (api-server) in production"

	if str != expected {
		t.Errorf("Resource.String() = %s, want %s", str, expected)
	}
}

func TestResource_Equals(t *testing.T) {
	resource1 := Resource{
		ID:            "i-1234567890abcdef0",
		Type:          "ec2:instance",
		Provider:      "aws",
		Name:          "web-server",
		Region:        "us-west-2",
		Configuration: map[string]interface{}{"instance_type": "t3.micro"},
		Tags:          map[string]string{"Environment": "production"},
	}

	resource2 := Resource{
		ID:            "i-1234567890abcdef0",
		Type:          "ec2:instance",
		Provider:      "aws",
		Name:          "web-server",
		Region:        "us-west-2",
		Configuration: map[string]interface{}{"instance_type": "t3.micro"},
		Tags:          map[string]string{"Environment": "production"},
	}

	if !resource1.Equals(resource2) {
		t.Error("Identical resources should be equal")
	}

	// Test different ID
	resource3 := resource1
	resource3.ID = "i-different"
	if resource1.Equals(resource3) {
		t.Error("Resources with different IDs should not be equal")
	}

	// Test different configuration
	resource4 := resource1
	resource4.Configuration = map[string]interface{}{"instance_type": "t3.small"}
	if resource1.Equals(resource4) {
		t.Error("Resources with different configurations should not be equal")
	}

	// Test different tags
	resource5 := resource1
	resource5.Tags = map[string]string{"Environment": "staging"}
	if resource1.Equals(resource5) {
		t.Error("Resources with different tags should not be equal")
	}
}

func TestResource_Hash(t *testing.T) {
	resource1 := Resource{
		ID:            "i-1234567890abcdef0",
		Type:          "ec2:instance",
		Provider:      "aws",
		Name:          "web-server",
		Region:        "us-west-2",
		Configuration: map[string]interface{}{"instance_type": "t3.micro"},
		Tags:          map[string]string{"Environment": "production"},
	}

	resource2 := Resource{
		ID:            "i-1234567890abcdef0",
		Type:          "ec2:instance",
		Provider:      "aws",
		Name:          "web-server",
		Region:        "us-west-2",
		Configuration: map[string]interface{}{"instance_type": "t3.micro"},
		Tags:          map[string]string{"Environment": "production"},
	}

	hash1 := resource1.Hash()
	hash2 := resource2.Hash()

	if hash1 != hash2 {
		t.Error("Identical resources should have the same hash")
	}

	// Test that different resources have different hashes
	resource3 := resource1
	resource3.Configuration = map[string]interface{}{"instance_type": "t3.small"}
	hash3 := resource3.Hash()

	if hash1 == hash3 {
		t.Error("Different resources should have different hashes")
	}

	// Hash should be deterministic
	hash4 := resource1.Hash()
	if hash1 != hash4 {
		t.Error("Hash should be deterministic")
	}
}
