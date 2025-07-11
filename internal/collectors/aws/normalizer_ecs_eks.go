package aws

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	ecsTypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	eksTypes "github.com/aws/aws-sdk-go-v2/service/eks/types"
	"github.com/yairfalse/vaino/pkg/types"
	"time"
)

// NormalizeECSCluster converts an ECS cluster to VAINO format
func (n *Normalizer) NormalizeECSCluster(cluster ecsTypes.Cluster) types.Resource {
	return types.Resource{
		ID:       aws.ToString(cluster.ClusterArn),
		Type:     "aws_ecs_cluster",
		Provider: "aws",
		Name:     aws.ToString(cluster.ClusterName),
		Region:   n.region,
		Configuration: map[string]interface{}{
			"cluster_name":                       aws.ToString(cluster.ClusterName),
			"status":                             aws.ToString(cluster.Status),
			"running_tasks_count":                cluster.RunningTasksCount,
			"pending_tasks_count":                cluster.PendingTasksCount,
			"active_services_count":              cluster.ActiveServicesCount,
			"statistics":                         normalizeECSStatistics(cluster.Statistics),
			"capacity_providers":                 cluster.CapacityProviders,
			"default_capacity_provider_strategy": normalizeCapacityProviderStrategy(cluster.DefaultCapacityProviderStrategy),
		},
		Tags: normalizeECSTags(cluster.Tags),
		Metadata: types.ResourceMetadata{
			CreatedAt: time.Now(), // ECS clusters don't have creation timestamp
		},
	}
}

// NormalizeECSService converts an ECS service to VAINO format
func (n *Normalizer) NormalizeECSService(service ecsTypes.Service, clusterArn string) types.Resource {
	return types.Resource{
		ID:       aws.ToString(service.ServiceArn),
		Type:     "aws_ecs_service",
		Provider: "aws",
		Name:     aws.ToString(service.ServiceName),
		Region:   n.region,
		Configuration: map[string]interface{}{
			"service_name":               aws.ToString(service.ServiceName),
			"cluster_arn":                clusterArn,
			"task_definition":            aws.ToString(service.TaskDefinition),
			"desired_count":              service.DesiredCount,
			"running_count":              service.RunningCount,
			"pending_count":              service.PendingCount,
			"status":                     aws.ToString(service.Status),
			"launch_type":                string(service.LaunchType),
			"platform_version":           aws.ToString(service.PlatformVersion),
			"capacity_provider_strategy": normalizeCapacityProviderStrategy(service.CapacityProviderStrategy),
			"load_balancers":             normalizeLoadBalancers(service.LoadBalancers),
			"service_registries":         normalizeServiceRegistries(service.ServiceRegistries),
		},
		Tags: normalizeECSTags(service.Tags),
		Metadata: types.ResourceMetadata{
			CreatedAt: aws.ToTime(service.CreatedAt),
			UpdatedAt: time.Now(), // ECS services don't have UpdatedAt field
		},
	}
}

// NormalizeECSTask converts an ECS task to VAINO format
func (n *Normalizer) NormalizeECSTask(task ecsTypes.Task, clusterArn string) types.Resource {
	return types.Resource{
		ID:       aws.ToString(task.TaskArn),
		Type:     "aws_ecs_task",
		Provider: "aws",
		Name:     aws.ToString(task.TaskDefinitionArn), // Use task definition as name
		Region:   n.region,
		Configuration: map[string]interface{}{
			"cluster_arn":            clusterArn,
			"task_definition_arn":    aws.ToString(task.TaskDefinitionArn),
			"desired_status":         aws.ToString(task.DesiredStatus),
			"last_status":            aws.ToString(task.LastStatus),
			"health_status":          string(task.HealthStatus),
			"launch_type":            string(task.LaunchType),
			"platform_version":       aws.ToString(task.PlatformVersion),
			"cpu":                    aws.ToString(task.Cpu),
			"memory":                 aws.ToString(task.Memory),
			"availability_zone":      aws.ToString(task.AvailabilityZone),
			"connectivity":           string(task.Connectivity),
			"connectivity_at":        formatTimePtr(task.ConnectivityAt),
			"pull_started_at":        formatTimePtr(task.PullStartedAt),
			"pull_stopped_at":        formatTimePtr(task.PullStoppedAt),
			"execution_stopped_at":   formatTimePtr(task.ExecutionStoppedAt),
			"capacity_provider_name": aws.ToString(task.CapacityProviderName),
		},
		Tags: normalizeECSTags(task.Tags),
		Metadata: types.ResourceMetadata{
			CreatedAt: aws.ToTime(task.CreatedAt),
			UpdatedAt: time.Now(), // ECS tasks don't have UpdatedAt field
		},
	}
}

// NormalizeEKSCluster converts an EKS cluster to VAINO format
func (n *Normalizer) NormalizeEKSCluster(cluster eksTypes.Cluster) types.Resource {
	return types.Resource{
		ID:       aws.ToString(cluster.Arn),
		Type:     "aws_eks_cluster",
		Provider: "aws",
		Name:     aws.ToString(cluster.Name),
		Region:   n.region,
		Configuration: map[string]interface{}{
			"name":                      aws.ToString(cluster.Name),
			"status":                    string(cluster.Status),
			"version":                   aws.ToString(cluster.Version),
			"endpoint":                  aws.ToString(cluster.Endpoint),
			"role_arn":                  aws.ToString(cluster.RoleArn),
			"platform_version":          aws.ToString(cluster.PlatformVersion),
			"kubernetes_network_config": normalizeKubernetesNetworkConfig(cluster.KubernetesNetworkConfig),
			"logging":                   normalizeLogging(cluster.Logging),
			"identity":                  normalizeIdentity(cluster.Identity),
			"encryption_config":         normalizeEncryptionConfig(cluster.EncryptionConfig),
		},
		Tags: cluster.Tags,
		Metadata: types.ResourceMetadata{
			CreatedAt: aws.ToTime(cluster.CreatedAt),
		},
	}
}

// NormalizeEKSNodeGroup converts an EKS node group to VAINO format
func (n *Normalizer) NormalizeEKSNodeGroup(nodeGroup eksTypes.Nodegroup, clusterName string) types.Resource {
	return types.Resource{
		ID:       aws.ToString(nodeGroup.NodegroupArn),
		Type:     "aws_eks_nodegroup",
		Provider: "aws",
		Name:     aws.ToString(nodeGroup.NodegroupName),
		Region:   n.region,
		Configuration: map[string]interface{}{
			"nodegroup_name":  aws.ToString(nodeGroup.NodegroupName),
			"cluster_name":    clusterName,
			"status":          string(nodeGroup.Status),
			"capacity_type":   string(nodeGroup.CapacityType),
			"scaling_config":  normalizeNodeGroupScalingConfig(nodeGroup.ScalingConfig),
			"instance_types":  nodeGroup.InstanceTypes,
			"ami_type":        string(nodeGroup.AmiType),
			"node_role":       aws.ToString(nodeGroup.NodeRole),
			"labels":          nodeGroup.Labels,
			"taints":          normalizeNodeGroupTaints(nodeGroup.Taints),
			"resources":       normalizeNodeGroupResources(nodeGroup.Resources),
			"disk_size":       aws.ToInt32(nodeGroup.DiskSize),
			"version":         aws.ToString(nodeGroup.Version),
			"launch_template": normalizeLaunchTemplateSpecification(nodeGroup.LaunchTemplate),
			"update_config":   normalizeNodeGroupUpdateConfig(nodeGroup.UpdateConfig),
		},
		Tags: nodeGroup.Tags,
		Metadata: types.ResourceMetadata{
			CreatedAt: aws.ToTime(nodeGroup.CreatedAt),
			UpdatedAt: aws.ToTime(nodeGroup.ModifiedAt),
		},
	}
}

// NormalizeEKSFargateProfile converts an EKS Fargate profile to VAINO format
func (n *Normalizer) NormalizeEKSFargateProfile(profile eksTypes.FargateProfile, clusterName string) types.Resource {
	return types.Resource{
		ID:       aws.ToString(profile.FargateProfileArn),
		Type:     "aws_eks_fargate_profile",
		Provider: "aws",
		Name:     aws.ToString(profile.FargateProfileName),
		Region:   n.region,
		Configuration: map[string]interface{}{
			"fargate_profile_name":   aws.ToString(profile.FargateProfileName),
			"cluster_name":           clusterName,
			"status":                 string(profile.Status),
			"pod_execution_role_arn": aws.ToString(profile.PodExecutionRoleArn),
			"subnets":                profile.Subnets,
			"selectors":              normalizeFargateProfileSelectors(profile.Selectors),
		},
		Tags: profile.Tags,
		Metadata: types.ResourceMetadata{
			CreatedAt: aws.ToTime(profile.CreatedAt),
		},
	}
}

// Helper functions for ECS normalization

func normalizeECSStatistics(stats []ecsTypes.KeyValuePair) map[string]string {
	result := make(map[string]string)
	for _, stat := range stats {
		if stat.Name != nil && stat.Value != nil {
			result[*stat.Name] = *stat.Value
		}
	}
	return result
}

func normalizeCapacityProviderStrategy(strategy []ecsTypes.CapacityProviderStrategyItem) []map[string]interface{} {
	var result []map[string]interface{}
	for _, item := range strategy {
		result = append(result, map[string]interface{}{
			"capacity_provider": aws.ToString(item.CapacityProvider),
			"weight":            item.Weight,
			"base":              item.Base,
		})
	}
	return result
}

func normalizeLoadBalancers(lbs []ecsTypes.LoadBalancer) []map[string]interface{} {
	var result []map[string]interface{}
	for _, lb := range lbs {
		result = append(result, map[string]interface{}{
			"target_group_arn":   aws.ToString(lb.TargetGroupArn),
			"load_balancer_name": aws.ToString(lb.LoadBalancerName),
			"container_name":     aws.ToString(lb.ContainerName),
			"container_port":     aws.ToInt32(lb.ContainerPort),
		})
	}
	return result
}

func normalizeServiceRegistries(registries []ecsTypes.ServiceRegistry) []map[string]interface{} {
	var result []map[string]interface{}
	for _, registry := range registries {
		result = append(result, map[string]interface{}{
			"registry_arn":   aws.ToString(registry.RegistryArn),
			"port":           aws.ToInt32(registry.Port),
			"container_name": aws.ToString(registry.ContainerName),
			"container_port": aws.ToInt32(registry.ContainerPort),
		})
	}
	return result
}

func normalizeECSTags(tags []ecsTypes.Tag) map[string]string {
	result := make(map[string]string)
	for _, tag := range tags {
		if tag.Key != nil && tag.Value != nil {
			result[*tag.Key] = *tag.Value
		}
	}
	return result
}

// Helper functions for EKS normalization

func normalizeKubernetesNetworkConfig(config *eksTypes.KubernetesNetworkConfigResponse) map[string]interface{} {
	if config == nil {
		return nil
	}
	return map[string]interface{}{
		"service_ipv4_cidr": aws.ToString(config.ServiceIpv4Cidr),
		"service_ipv6_cidr": aws.ToString(config.ServiceIpv6Cidr),
		"ip_family":         string(config.IpFamily),
	}
}

func normalizeLogging(logging *eksTypes.Logging) map[string]interface{} {
	if logging == nil {
		return nil
	}
	return map[string]interface{}{
		"enabled": normalizeLogSetup(logging.ClusterLogging),
	}
}

func normalizeLogSetup(logSetups []eksTypes.LogSetup) []map[string]interface{} {
	var result []map[string]interface{}
	for _, setup := range logSetups {
		result = append(result, map[string]interface{}{
			"types":   setup.Types,
			"enabled": aws.ToBool(setup.Enabled),
		})
	}
	return result
}

func normalizeIdentity(identity *eksTypes.Identity) map[string]interface{} {
	if identity == nil {
		return nil
	}
	return map[string]interface{}{
		"oidc": normalizeOIDC(identity.Oidc),
	}
}

func normalizeOIDC(oidc *eksTypes.OIDC) map[string]interface{} {
	if oidc == nil {
		return nil
	}
	return map[string]interface{}{
		"issuer": aws.ToString(oidc.Issuer),
	}
}

func normalizeEncryptionConfig(configs []eksTypes.EncryptionConfig) []map[string]interface{} {
	var result []map[string]interface{}
	for _, config := range configs {
		result = append(result, map[string]interface{}{
			"resources": config.Resources,
			"provider":  normalizeProvider(config.Provider),
		})
	}
	return result
}

func normalizeProvider(provider *eksTypes.Provider) map[string]interface{} {
	if provider == nil {
		return nil
	}
	return map[string]interface{}{
		"key_arn": aws.ToString(provider.KeyArn),
	}
}

func normalizeNodeGroupScalingConfig(config *eksTypes.NodegroupScalingConfig) map[string]interface{} {
	if config == nil {
		return nil
	}
	return map[string]interface{}{
		"min_size":     aws.ToInt32(config.MinSize),
		"max_size":     aws.ToInt32(config.MaxSize),
		"desired_size": aws.ToInt32(config.DesiredSize),
	}
}

func normalizeNodeGroupTaints(taints []eksTypes.Taint) []map[string]interface{} {
	var result []map[string]interface{}
	for _, taint := range taints {
		result = append(result, map[string]interface{}{
			"key":    aws.ToString(taint.Key),
			"value":  aws.ToString(taint.Value),
			"effect": string(taint.Effect),
		})
	}
	return result
}

func normalizeNodeGroupResources(resources *eksTypes.NodegroupResources) map[string]interface{} {
	if resources == nil {
		return nil
	}
	return map[string]interface{}{
		"auto_scaling_groups":          normalizeAutoScalingGroups(resources.AutoScalingGroups),
		"remote_access_security_group": aws.ToString(resources.RemoteAccessSecurityGroup),
	}
}

func normalizeAutoScalingGroups(asgs []eksTypes.AutoScalingGroup) []map[string]interface{} {
	var result []map[string]interface{}
	for _, asg := range asgs {
		result = append(result, map[string]interface{}{
			"name": aws.ToString(asg.Name),
		})
	}
	return result
}

func normalizeLaunchTemplateSpecification(template *eksTypes.LaunchTemplateSpecification) map[string]interface{} {
	if template == nil {
		return nil
	}
	return map[string]interface{}{
		"name":    aws.ToString(template.Name),
		"version": aws.ToString(template.Version),
		"id":      aws.ToString(template.Id),
	}
}

func normalizeNodeGroupUpdateConfig(config *eksTypes.NodegroupUpdateConfig) map[string]interface{} {
	if config == nil {
		return nil
	}
	return map[string]interface{}{
		"max_unavailable":            aws.ToInt32(config.MaxUnavailable),
		"max_unavailable_percentage": aws.ToInt32(config.MaxUnavailablePercentage),
	}
}

func normalizeFargateProfileSelectors(selectors []eksTypes.FargateProfileSelector) []map[string]interface{} {
	var result []map[string]interface{}
	for _, selector := range selectors {
		result = append(result, map[string]interface{}{
			"namespace": aws.ToString(selector.Namespace),
			"labels":    selector.Labels,
		})
	}
	return result
}

// formatTimePtr formats a time pointer to string
func formatTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}
