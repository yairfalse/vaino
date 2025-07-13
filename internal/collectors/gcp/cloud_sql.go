package gcp

import (
	"context"
	"fmt"

	"github.com/yairfalse/vaino/pkg/types"
)

// GCPCloudSQLInstance represents a Cloud SQL instance from the GCP API
type GCPCloudSQLInstance struct {
	Name                string                 `json:"name"`
	ProjectID           string                 `json:"project"`
	DatabaseVersion     string                 `json:"databaseVersion"`
	Region              string                 `json:"region"`
	State               string                 `json:"state"`
	BackendType         string                 `json:"backendType"`
	InstanceType        string                 `json:"instanceType"`
	ConnectionName      string                 `json:"connectionName"`
	IPAddresses         []GCPInstanceIPAddress `json:"ipAddresses"`
	Settings            GCPInstanceSettings    `json:"settings"`
	CurrentDiskSize     int64                  `json:"currentDiskSize"`
	MaxDiskSize         int64                  `json:"maxDiskSize"`
	SelfLink            string                 `json:"selfLink"`
	ServiceAccountEmail string                 `json:"serviceAccountEmailAddress"`
	CreateTime          string                 `json:"createTime"`
}

type GCPInstanceIPAddress struct {
	Type      string `json:"type"`
	IPAddress string `json:"ipAddress"`
}

type GCPInstanceSettings struct {
	Tier                        string                 `json:"tier"`
	ActivationPolicy            string                 `json:"activationPolicy"`
	StorageAutoResize           bool                   `json:"storageAutoResize"`
	StorageAutoResizeLimit      int64                  `json:"storageAutoResizeLimit"`
	DataDiskSizeGb              int64                  `json:"dataDiskSizeGb"`
	DataDiskType                string                 `json:"dataDiskType"`
	LocationPreference          GCPLocationPreference  `json:"locationPreference"`
	BackupConfiguration         GCPBackupConfiguration `json:"backupConfiguration"`
	MaintenanceWindow           GCPMaintenanceWindow   `json:"maintenanceWindow"`
	DatabaseFlags               []GCPDatabaseFlag      `json:"databaseFlags"`
	UserLabels                  map[string]string      `json:"userLabels"`
	AvailabilityType            string                 `json:"availabilityType"`
	PricingPlan                 string                 `json:"pricingPlan"`
	ReplicationType             string                 `json:"replicationType"`
	CrashSafeReplicationEnabled bool                   `json:"crashSafeReplicationEnabled"`
}

type GCPLocationPreference struct {
	Zone                 string `json:"zone"`
	SecondaryZone        string `json:"secondaryZone"`
	FollowGaeApplication string `json:"followGaeApplication"`
}

type GCPBackupConfiguration struct {
	Enabled                     bool                       `json:"enabled"`
	StartTime                   string                     `json:"startTime"`
	Location                    string                     `json:"location"`
	PointInTimeRecoveryEnabled  bool                       `json:"pointInTimeRecoveryEnabled"`
	TransactionLogRetentionDays int32                      `json:"transactionLogRetentionDays"`
	BackupRetentionSettings     GCPBackupRetentionSettings `json:"backupRetentionSettings"`
}

type GCPBackupRetentionSettings struct {
	RetentionUnit   string `json:"retentionUnit"`
	RetainedBackups int32  `json:"retainedBackups"`
}

type GCPMaintenanceWindow struct {
	Hour        int32  `json:"hour"`
	Day         int32  `json:"day"`
	UpdateTrack string `json:"updateTrack"`
}

type GCPDatabaseFlag struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// collectCloudSQLResources collects GCP Cloud SQL instances and databases
func (c *GCPCollector) collectCloudSQLResources(ctx context.Context, clientPool *GCPServicePool, projectID string, regions []string) ([]types.Resource, error) {
	var resources []types.Resource

	// Get Cloud SQL instances (global)
	instances, err := clientPool.GetCloudSQLInstances(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get Cloud SQL instances: %w", err)
	}

	for _, instance := range instances {
		resource := c.normalizer.NormalizeCloudSQLInstance(instance)
		resources = append(resources, resource)

		// Get databases for this instance
		databases, err := clientPool.GetCloudSQLDatabases(ctx, projectID, instance.Name)
		if err == nil {
			for _, database := range databases {
				dbResource := c.normalizer.NormalizeCloudSQLDatabase(database, instance.Name)
				resources = append(resources, dbResource)
			}
		}

		// Get users for this instance
		users, err := clientPool.GetCloudSQLUsers(ctx, projectID, instance.Name)
		if err == nil {
			for _, user := range users {
				userResource := c.normalizer.NormalizeCloudSQLUser(user, instance.Name)
				resources = append(resources, userResource)
			}
		}
	}

	return resources, nil
}

// GCP Cloud SQL Database
type GCPCloudSQLDatabase struct {
	Name      string `json:"name"`
	Instance  string `json:"instance"`
	Project   string `json:"project"`
	Charset   string `json:"charset"`
	Collation string `json:"collation"`
	SelfLink  string `json:"selfLink"`
}

// GCP Cloud SQL User
type GCPCloudSQLUser struct {
	Name     string `json:"name"`
	Instance string `json:"instance"`
	Project  string `json:"project"`
	Host     string `json:"host"`
	Type     string `json:"type"`
	SelfLink string `json:"selfLink"`
}
