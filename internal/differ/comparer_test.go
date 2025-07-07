package differ

import (
	"testing"

	"github.com/yairfalse/wgo/pkg/types"
)

func TestDefaultComparer_CompareResources(t *testing.T) {
	comparer := &DefaultComparer{}
	
	tests := []struct {
		name           string
		baseline       types.Resource
		current        types.Resource
		expectedChanges int
		expectSpecificChange string
	}{
		{
			name: "identical resources",
			baseline: types.Resource{
				ID:       "resource-1",
				Type:     "instance",
				Name:     "test-server",
				Provider: "aws",
				Configuration: map[string]interface{}{
					"instance_type": "t3.micro",
					"state":         "running",
				},
			},
			current: types.Resource{
				ID:       "resource-1",
				Type:     "instance",
				Name:     "test-server", 
				Provider: "aws",
				Configuration: map[string]interface{}{
					"instance_type": "t3.micro",
					"state":         "running",
				},
			},
			expectedChanges: 0,
		},
		{
			name: "changed instance type",
			baseline: types.Resource{
				ID:       "resource-1",
				Type:     "instance",
				Name:     "test-server",
				Provider: "aws",
				Configuration: map[string]interface{}{
					"instance_type": "t3.micro",
					"state":         "running",
				},
			},
			current: types.Resource{
				ID:       "resource-1", 
				Type:     "instance",
				Name:     "test-server",
				Provider: "aws",
				Configuration: map[string]interface{}{
					"instance_type": "t3.medium", // Changed
					"state":         "running",
				},
			},
			expectedChanges:      1,
			expectSpecificChange: "instance_type",
		},
		{
			name: "multiple changes",
			baseline: types.Resource{
				ID:       "resource-1",
				Type:     "instance",
				Name:     "test-server",
				Provider: "aws", 
				Configuration: map[string]interface{}{
					"instance_type": "t3.micro",
					"state":         "running",
					"monitoring":    false,
				},
			},
			current: types.Resource{
				ID:       "resource-1",
				Type:     "instance", 
				Name:     "test-server",
				Provider: "aws",
				Configuration: map[string]interface{}{
					"instance_type": "t3.medium", // Changed
					"state":         "stopped",   // Changed
					"monitoring":    false,
				},
			},
			expectedChanges: 2,
		},
		{
			name: "added field",
			baseline: types.Resource{
				ID:       "resource-1",
				Type:     "instance",
				Name:     "test-server",
				Provider: "aws",
				Configuration: map[string]interface{}{
					"instance_type": "t3.micro",
				},
			},
			current: types.Resource{
				ID:       "resource-1",
				Type:     "instance",
				Name:     "test-server",
				Provider: "aws", 
				Configuration: map[string]interface{}{
					"instance_type": "t3.micro",
					"monitoring":    true, // Added field
				},
			},
			expectedChanges:      1,
			expectSpecificChange: "monitoring",
		},
		{
			name: "removed field",
			baseline: types.Resource{
				ID:       "resource-1",
				Type:     "instance", 
				Name:     "test-server",
				Provider: "aws",
				Configuration: map[string]interface{}{
					"instance_type": "t3.micro",
					"monitoring":    true,
				},
			},
			current: types.Resource{
				ID:       "resource-1",
				Type:     "instance",
				Name:     "test-server",
				Provider: "aws",
				Configuration: map[string]interface{}{
					"instance_type": "t3.micro",
					// monitoring field removed
				},
			},
			expectedChanges:      1,
			expectSpecificChange: "monitoring",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes := comparer.CompareResources(tt.baseline, tt.current)
			
			if len(changes) != tt.expectedChanges {
				t.Errorf("expected %d changes, got %d", tt.expectedChanges, len(changes))
				for i, change := range changes {
					t.Logf("Change %d: %s: %v -> %v", i, change.Field, change.OldValue, change.NewValue)
				}
			}
			
			if tt.expectSpecificChange != "" {
				found := false
				for _, change := range changes {
					if change.Field == tt.expectSpecificChange {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected change in field %s, but not found", tt.expectSpecificChange)
				}
			}
		})
	}
}

func TestDefaultComparer_CompareConfiguration(t *testing.T) {
	comparer := &DefaultComparer{}
	
	tests := []struct {
		name            string
		baseline        map[string]interface{}
		current         map[string]interface{}
		expectedChanges int
	}{
		{
			name: "nested object changes",
			baseline: map[string]interface{}{
				"security_groups": map[string]interface{}{
					"ingress": []interface{}{
						map[string]interface{}{
							"port":     80,
							"protocol": "tcp",
						},
					},
				},
			},
			current: map[string]interface{}{
				"security_groups": map[string]interface{}{
					"ingress": []interface{}{
						map[string]interface{}{
							"port":     443, // Changed port
							"protocol": "tcp",
						},
					},
				},
			},
			expectedChanges: 1,
		},
		{
			name: "array changes",
			baseline: map[string]interface{}{
				"tags": []interface{}{"prod", "web"},
			},
			current: map[string]interface{}{
				"tags": []interface{}{"prod", "web", "critical"}, // Added element
			},
			expectedChanges: 1,
		},
		{
			name: "deep nested changes",
			baseline: map[string]interface{}{
				"network": map[string]interface{}{
					"vpc": map[string]interface{}{
						"subnets": map[string]interface{}{
							"private": map[string]interface{}{
								"cidr": "10.0.1.0/24",
							},
						},
					},
				},
			},
			current: map[string]interface{}{
				"network": map[string]interface{}{
					"vpc": map[string]interface{}{
						"subnets": map[string]interface{}{
							"private": map[string]interface{}{
								"cidr": "10.0.2.0/24", // Changed deep nested value
							},
						},
					},
				},
			},
			expectedChanges: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes := comparer.CompareConfiguration("", tt.baseline, tt.current)
			
			if len(changes) != tt.expectedChanges {
				t.Errorf("expected %d changes, got %d", tt.expectedChanges, len(changes))
				for i, change := range changes {
					t.Logf("Change %d: %s: %v -> %v", i, change.Path, change.OldValue, change.NewValue)
				}
			}
		})
	}
}

func TestDefaultComparer_SecurityRelatedChanges(t *testing.T) {
	comparer := &DefaultComparer{}
	
	baseline := types.Resource{
		ID:       "i-1234567890",
		Type:     "instance",
		Name:     "web-server",
		Provider: "aws",
		Configuration: map[string]interface{}{
			"instance_type":        "t3.micro",
			"state":               "running",
			"security_group_ids":   []interface{}{"sg-123", "sg-456"},
			"subnet_id":           "subnet-abc",
			"public_ip":           "1.2.3.4",
			"private_ip":          "10.0.1.100",
		},
	}
	
	current := types.Resource{
		ID:       "i-1234567890",
		Type:     "instance",
		Name:     "web-server",
		Provider: "aws",
		Configuration: map[string]interface{}{
			"instance_type":        "t3.medium", // instance type change
			"state":               "running",
			"security_group_ids":   []interface{}{"sg-123", "sg-789"}, // security groups changed
			"subnet_id":           "subnet-abc",
			"public_ip":           "1.2.3.5", // IP changed
			"private_ip":          "10.0.1.100",
		},
	}
	
	changes := comparer.CompareResources(baseline, current)
	
	// Should detect all changes
	if len(changes) == 0 {
		t.Error("expected changes to be detected")
	}
	
	// Verify specific changes are detected
	changeFields := make(map[string]bool)
	for _, change := range changes {
		changeFields[change.Field] = true
	}
	
	expectedFields := []string{"instance_type", "security_group_ids", "public_ip"}
	for _, field := range expectedFields {
		if !changeFields[field] {
			t.Errorf("expected %s change to be detected", field)
		}
	}
}

func TestDefaultComparer_SecurityGroupChanges(t *testing.T) {
	comparer := &DefaultComparer{}
	
	baseline := types.Resource{
		ID:       "sg-123456",
		Type:     "security_group",
		Name:     "web-sg",
		Provider: "aws",
		Configuration: map[string]interface{}{
			"ingress_rules": []interface{}{
				map[string]interface{}{
					"port":        80,
					"protocol":    "tcp",
					"cidr_blocks": []interface{}{"0.0.0.0/0"},
				},
			},
		},
	}
	
	current := types.Resource{
		ID:       "sg-123456",
		Type:     "security_group",
		Name:     "web-sg",
		Provider: "aws",
		Configuration: map[string]interface{}{
			"ingress_rules": []interface{}{
				map[string]interface{}{
					"port":        80,
					"protocol":    "tcp",
					"cidr_blocks": []interface{}{"0.0.0.0/0"},
				},
				map[string]interface{}{
					"port":        22, // SSH added
					"protocol":    "tcp",
					"cidr_blocks": []interface{}{"0.0.0.0/0"},
				},
			},
		},
	}
	
	changes := comparer.CompareResources(baseline, current)
	
	// Should detect the addition of SSH rule
	if len(changes) == 0 {
		t.Error("expected changes to be detected")
	}
	
	// Should find ingress_rules change
	foundIngressChange := false
	for _, change := range changes {
		if change.Field == "ingress_rules" {
			foundIngressChange = true
		}
	}
	
	if !foundIngressChange {
		t.Error("expected ingress_rules change to be detected")
	}
}