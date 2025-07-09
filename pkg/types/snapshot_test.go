package types

import (
	"testing"
	"time"
)

func TestSnapshot_Validate(t *testing.T) {
	baseTime := time.Now()

	tests := []struct {
		name     string
		snapshot Snapshot
		wantErr  bool
	}{
		{
			name: "valid snapshot",
			snapshot: Snapshot{
				ID:        "snap-123",
				Timestamp: baseTime,
				Provider:  "aws",
				Resources: []Resource{
					{
						ID:       "i-123",
						Type:     "ec2:instance",
						Provider: "aws",
						Name:     "test",
						Region:   "us-west-2",
					},
				},
				Metadata: SnapshotMetadata{
					CollectorVersion: "1.0.0",
					CollectionTime:   time.Second * 5,
					ResourceCount:    1,
				},
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			snapshot: Snapshot{
				Timestamp: baseTime,
				Provider:  "aws",
				Resources: []Resource{},
				Metadata:  SnapshotMetadata{},
			},
			wantErr: true,
		},
		{
			name: "missing provider",
			snapshot: Snapshot{
				ID:        "snap-123",
				Timestamp: baseTime,
				Resources: []Resource{},
				Metadata:  SnapshotMetadata{},
			},
			wantErr: true,
		},
		{
			name: "zero timestamp",
			snapshot: Snapshot{
				ID:        "snap-123",
				Timestamp: time.Time{},
				Provider:  "aws",
				Resources: []Resource{},
				Metadata:  SnapshotMetadata{},
			},
			wantErr: true,
		},
		{
			name: "invalid resource",
			snapshot: Snapshot{
				ID:        "snap-123",
				Timestamp: baseTime,
				Provider:  "aws",
				Resources: []Resource{
					{
						// Missing required fields
						Type: "ec2:instance",
					},
				},
				Metadata: SnapshotMetadata{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.snapshot.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Snapshot.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSnapshot_ResourceCount(t *testing.T) {
	snapshot := Snapshot{
		Resources: []Resource{
			{ID: "1", Type: "ec2:instance", Provider: "aws"},
			{ID: "2", Type: "ec2:volume", Provider: "aws"},
			{ID: "3", Type: "pod", Provider: "kubernetes"},
		},
	}

	count := snapshot.ResourceCount()
	if count != 3 {
		t.Errorf("Expected resource count 3, got %d", count)
	}

	// Test empty snapshot
	emptySnapshot := Snapshot{}
	count = emptySnapshot.ResourceCount()
	if count != 0 {
		t.Errorf("Expected resource count 0 for empty snapshot, got %d", count)
	}
}

func TestSnapshot_ResourcesByProvider(t *testing.T) {
	snapshot := Snapshot{
		Resources: []Resource{
			{ID: "1", Type: "ec2:instance", Provider: "aws"},
			{ID: "2", Type: "ec2:volume", Provider: "aws"},
			{ID: "3", Type: "pod", Provider: "kubernetes"},
			{ID: "4", Type: "service", Provider: "kubernetes"},
		},
	}

	byProvider := snapshot.ResourcesByProvider()

	if len(byProvider) != 2 {
		t.Errorf("Expected 2 providers, got %d", len(byProvider))
	}

	awsResources := byProvider["aws"]
	if len(awsResources) != 2 {
		t.Errorf("Expected 2 AWS resources, got %d", len(awsResources))
	}

	k8sResources := byProvider["kubernetes"]
	if len(k8sResources) != 2 {
		t.Errorf("Expected 2 Kubernetes resources, got %d", len(k8sResources))
	}
}

func TestSnapshot_ResourcesByType(t *testing.T) {
	snapshot := Snapshot{
		Resources: []Resource{
			{ID: "1", Type: "ec2:instance", Provider: "aws"},
			{ID: "2", Type: "ec2:instance", Provider: "aws"},
			{ID: "3", Type: "ec2:volume", Provider: "aws"},
			{ID: "4", Type: "pod", Provider: "kubernetes"},
		},
	}

	byType := snapshot.ResourcesByType()

	if len(byType) != 3 {
		t.Errorf("Expected 3 resource types, got %d", len(byType))
	}

	instances := byType["ec2:instance"]
	if len(instances) != 2 {
		t.Errorf("Expected 2 EC2 instances, got %d", len(instances))
	}

	volumes := byType["ec2:volume"]
	if len(volumes) != 1 {
		t.Errorf("Expected 1 EC2 volume, got %d", len(volumes))
	}

	pods := byType["pod"]
	if len(pods) != 1 {
		t.Errorf("Expected 1 pod, got %d", len(pods))
	}
}

func TestSnapshot_FindResource(t *testing.T) {
	snapshot := Snapshot{
		Resources: []Resource{
			{ID: "i-123", Type: "ec2:instance", Provider: "aws", Name: "web-server"},
			{ID: "vol-456", Type: "ec2:volume", Provider: "aws", Name: "data-volume"},
			{ID: "pod-789", Type: "pod", Provider: "kubernetes", Name: "api-pod"},
		},
	}

	// Test finding by ID
	resource, found := snapshot.FindResource("i-123")
	if !found {
		t.Error("Expected to find resource i-123")
	}
	if resource.Name != "web-server" {
		t.Errorf("Expected name 'web-server', got %s", resource.Name)
	}

	// Test not found
	_, found = snapshot.FindResource("nonexistent")
	if found {
		t.Error("Expected not to find nonexistent resource")
	}
}

func TestBaseline_Validate(t *testing.T) {
	baseTime := time.Now()

	tests := []struct {
		name     string
		baseline Baseline
		wantErr  bool
	}{
		{
			name: "valid baseline",
			baseline: Baseline{
				ID:          "baseline-123",
				Name:        "Production Baseline",
				Description: "Baseline for production environment",
				SnapshotID:  "snap-456",
				CreatedAt:   baseTime,
				Version:     "1.0.0",
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			baseline: Baseline{
				Name:       "Production Baseline",
				SnapshotID: "snap-456",
				CreatedAt:  baseTime,
				Version:    "1.0.0",
			},
			wantErr: true,
		},
		{
			name: "missing name",
			baseline: Baseline{
				ID:         "baseline-123",
				SnapshotID: "snap-456",
				CreatedAt:  baseTime,
				Version:    "1.0.0",
			},
			wantErr: true,
		},
		{
			name: "missing snapshot ID",
			baseline: Baseline{
				ID:        "baseline-123",
				Name:      "Production Baseline",
				CreatedAt: baseTime,
				Version:   "1.0.0",
			},
			wantErr: true,
		},
		{
			name: "zero created time",
			baseline: Baseline{
				ID:         "baseline-123",
				Name:       "Production Baseline",
				SnapshotID: "snap-456",
				CreatedAt:  time.Time{},
				Version:    "1.0.0",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.baseline.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Baseline.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSnapshotMetadata_Validate(t *testing.T) {
	tests := []struct {
		name     string
		metadata SnapshotMetadata
		wantErr  bool
	}{
		{
			name: "valid metadata",
			metadata: SnapshotMetadata{
				CollectorVersion: "1.0.0",
				CollectionTime:   time.Second * 5,
				ResourceCount:    10,
				Regions:          []string{"us-west-2", "us-east-1"},
			},
			wantErr: false,
		},
		{
			name: "missing collector version",
			metadata: SnapshotMetadata{
				CollectionTime: time.Second * 5,
				ResourceCount:  10,
			},
			wantErr: true,
		},
		{
			name: "negative resource count",
			metadata: SnapshotMetadata{
				CollectorVersion: "1.0.0",
				CollectionTime:   time.Second * 5,
				ResourceCount:    -1,
			},
			wantErr: true,
		},
		{
			name: "negative collection time",
			metadata: SnapshotMetadata{
				CollectorVersion: "1.0.0",
				CollectionTime:   -time.Second,
				ResourceCount:    10,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.metadata.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("SnapshotMetadata.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
