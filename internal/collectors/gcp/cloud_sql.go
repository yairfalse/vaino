package gcp

import (
	"context"
	"fmt"

	"github.com/yairfalse/vaino/pkg/types"
	sqladmin "google.golang.org/api/sqladmin/v1beta4"
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
	Tier                        string                       `json:"tier"`
	ActivationPolicy            string                       `json:"activationPolicy"`
	StorageAutoResize           bool                         `json:"storageAutoResize"`
	StorageAutoResizeLimit      int64                        `json:"storageAutoResizeLimit"`
	DataDiskSizeGb              int64                        `json:"dataDiskSizeGb"`
	DataDiskType                string                       `json:"dataDiskType"`
	LocationPreference          GCPLocationPreference        `json:"locationPreference"`
	BackupConfiguration         GCPBackupConfiguration       `json:"backupConfiguration"`
	MaintenanceWindow           GCPCloudSQLMaintenanceWindow `json:"maintenanceWindow"`
	DatabaseFlags               []GCPDatabaseFlag            `json:"databaseFlags"`
	UserLabels                  map[string]string            `json:"userLabels"`
	AvailabilityType            string                       `json:"availabilityType"`
	PricingPlan                 string                       `json:"pricingPlan"`
	ReplicationType             string                       `json:"replicationType"`
	CrashSafeReplicationEnabled bool                         `json:"crashSafeReplicationEnabled"`
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

type GCPCloudSQLMaintenanceWindow struct {
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

	// Get Cloud SQL instances
	instances, err := clientPool.GetCloudSQLInstances(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get Cloud SQL instances: %w", err)
	}

	// Convert each instance to a VAINO resource
	for _, instance := range instances {
		// Convert the API response to our GCPCloudSQLInstance type
		if sqlInstance, ok := instance.(*sqladmin.DatabaseInstance); ok {
			gcpInstance := convertSQLInstanceToGCP(sqlInstance)
			resource := c.normalizer.NormalizeCloudSQLInstance(gcpInstance)
			resources = append(resources, resource)
		}
	}

	return resources, nil
}

// convertSQLInstanceToGCP converts a sqladmin.DatabaseInstance to our GCPCloudSQLInstance type
func convertSQLInstanceToGCP(instance *sqladmin.DatabaseInstance) GCPCloudSQLInstance {
	gcpInstance := GCPCloudSQLInstance{
		Name:                instance.Name,
		ProjectID:           instance.Project,
		DatabaseVersion:     instance.DatabaseVersion,
		Region:              instance.Region,
		State:               instance.State,
		BackendType:         instance.BackendType,
		InstanceType:        instance.InstanceType,
		ConnectionName:      instance.ConnectionName,
		CurrentDiskSize:     instance.CurrentDiskSize,
		MaxDiskSize:         instance.MaxDiskSize,
		SelfLink:            instance.SelfLink,
		ServiceAccountEmail: instance.ServiceAccountEmailAddress,
		CreateTime:          instance.CreateTime,
	}

	// Convert IP addresses
	for _, ip := range instance.IpAddresses {
		gcpInstance.IPAddresses = append(gcpInstance.IPAddresses, GCPInstanceIPAddress{
			Type:      ip.Type,
			IPAddress: ip.IpAddress,
		})
	}

	// Convert settings
	if instance.Settings != nil {
		settings := GCPInstanceSettings{
			Tier:                        instance.Settings.Tier,
			ActivationPolicy:            instance.Settings.ActivationPolicy,
			StorageAutoResize:           instance.Settings.StorageAutoResize != nil && *instance.Settings.StorageAutoResize,
			StorageAutoResizeLimit:      instance.Settings.StorageAutoResizeLimit,
			DataDiskSizeGb:              instance.Settings.DataDiskSizeGb,
			DataDiskType:                instance.Settings.DataDiskType,
			AvailabilityType:            instance.Settings.AvailabilityType,
			PricingPlan:                 instance.Settings.PricingPlan,
			ReplicationType:             instance.Settings.ReplicationType,
			CrashSafeReplicationEnabled: instance.Settings.CrashSafeReplicationEnabled,
		}

		// Convert user labels
		if instance.Settings.UserLabels != nil {
			settings.UserLabels = make(map[string]string)
			for k, v := range instance.Settings.UserLabels {
				settings.UserLabels[k] = v
			}
		}

		// Convert location preference
		if instance.Settings.LocationPreference != nil {
			settings.LocationPreference = GCPLocationPreference{
				Zone:                 instance.Settings.LocationPreference.Zone,
				SecondaryZone:        instance.Settings.LocationPreference.SecondaryZone,
				FollowGaeApplication: instance.Settings.LocationPreference.FollowGaeApplication,
			}
		}

		// Convert backup configuration
		if instance.Settings.BackupConfiguration != nil {
			settings.BackupConfiguration = GCPBackupConfiguration{
				Enabled:                     instance.Settings.BackupConfiguration.Enabled,
				StartTime:                   instance.Settings.BackupConfiguration.StartTime,
				Location:                    instance.Settings.BackupConfiguration.Location,
				PointInTimeRecoveryEnabled:  instance.Settings.BackupConfiguration.PointInTimeRecoveryEnabled,
				TransactionLogRetentionDays: int32(instance.Settings.BackupConfiguration.TransactionLogRetentionDays),
			}

			// Convert backup retention settings
			if instance.Settings.BackupConfiguration.BackupRetentionSettings != nil {
				settings.BackupConfiguration.BackupRetentionSettings = GCPBackupRetentionSettings{
					RetentionUnit:   instance.Settings.BackupConfiguration.BackupRetentionSettings.RetentionUnit,
					RetainedBackups: int32(instance.Settings.BackupConfiguration.BackupRetentionSettings.RetainedBackups),
				}
			}
		}

		gcpInstance.Settings = settings
	}

	return gcpInstance
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
