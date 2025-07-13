package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yairfalse/vaino/internal/analyzer"
	"github.com/yairfalse/vaino/internal/storage"
	"github.com/yairfalse/vaino/internal/visualization"
	"golang.org/x/term"
)

// TimelineOptions holds filtering and display options for timeline analysis
type TimelineOptions struct {
	EventTypes         []string
	Severities         []string
	MinConfidence      float64
	IncidentsOnly      bool
	Compact            bool
	IncludePredictions bool
}

func newTimelineCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "timeline",
		Short: "Browse infrastructure snapshots chronologically",
		Long: `Browse stored infrastructure snapshots in chronological order.
This provides a simple view of when snapshots were taken and basic statistics.

For advanced change timeline with correlation analysis, use: vaino changes --timeline
For change comparison between snapshots, use: vaino diff`,
		Example: `  # Show snapshot timeline
  vaino timeline

  # Show snapshots from last 2 weeks
  vaino timeline --since "2 weeks ago"

  # Show timeline between two snapshots
  vaino timeline --between snap1 snap2

  # Show snapshots for specific provider
  vaino timeline --provider kubernetes

  # Export timeline as JSON
  vaino timeline --output json`,
		RunE: runTimeline,
	}

	// Date/time filters
	cmd.Flags().StringP("since", "s", "", "show snapshots since date/duration (e.g., '2 weeks ago', '2024-01-01')")
	cmd.Flags().StringP("until", "u", "", "show snapshots until date (e.g., '2024-01-31')")
	cmd.Flags().StringSlice("between", nil, "show snapshots between two specific snapshots (e.g., --between snap1,snap2)")

	// Provider filters
	cmd.Flags().StringSlice("provider", nil, "filter by provider (aws, gcp, kubernetes, terraform)")

	// Tag filters
	// Removed baselines-only flag - use --tags instead
	cmd.Flags().StringSlice("tags", nil, "filter by tags (key=value)")

	// Output options
	cmd.Flags().BoolP("stats", "", false, "show snapshot statistics")
	cmd.Flags().BoolP("quiet", "q", false, "quiet mode - show timestamps only")
	cmd.Flags().IntP("limit", "l", 50, "limit number of snapshots shown")

	// Analysis options
	cmd.Flags().BoolP("analyze", "a", false, "perform advanced timeline analysis with correlations and trends")
	cmd.Flags().Bool("events", false, "show detected events in timeline")
	cmd.Flags().Bool("trends", false, "show trend analysis")
	cmd.Flags().Bool("correlations", false, "show correlation analysis between providers")

	// Enhanced filtering options
	cmd.Flags().StringSlice("event-types", nil, "filter events by type (resource_addition, resource_removal, deployment, etc.)")
	cmd.Flags().StringSlice("severities", nil, "filter events by severity (critical, warning, info)")
	cmd.Flags().String("min-confidence", "", "minimum confidence for trends/correlations (0.0-1.0)")
	cmd.Flags().Bool("incidents-only", false, "show only critical incidents and large-scale changes")
	cmd.Flags().Bool("compact", false, "compact timeline view with less detail")
	cmd.Flags().Bool("include-predictions", false, "include future predictions in trend analysis")

	return cmd
}

func runTimeline(cmd *cobra.Command, args []string) error {
	cfg := GetConfig()

	// Initialize storage
	localStorage := storage.NewLocal(cfg.Storage.BasePath)

	// Get all snapshots
	snapshots, err := localStorage.ListSnapshots()
	if err != nil {
		return fmt.Errorf("failed to list snapshots: %w", err)
	}

	if len(snapshots) == 0 {
		fmt.Println("No snapshots found. Run 'vaino scan' to create your first snapshot.")
		return nil
	}

	// Check for --between flag
	betweenSnapshots, _ := cmd.Flags().GetStringSlice("between")
	if len(betweenSnapshots) == 2 {
		return handleTimelineBetween(localStorage, snapshots, betweenSnapshots[0], betweenSnapshots[1], cmd)
	} else if len(betweenSnapshots) == 1 {
		return fmt.Errorf("--between requires two snapshot identifiers")
	}

	// Parse filter options
	sinceTime, err := parseTimeFilter(cmd, "since")
	if err != nil {
		return fmt.Errorf("invalid --since value: %w", err)
	}

	untilTime, err := parseTimeFilter(cmd, "until")
	if err != nil {
		return fmt.Errorf("invalid --until value: %w", err)
	}

	providers, _ := cmd.Flags().GetStringSlice("provider")
	tags, _ := cmd.Flags().GetStringSlice("tags")
	showStats, _ := cmd.Flags().GetBool("stats")
	quiet, _ := cmd.Flags().GetBool("quiet")
	limit, _ := cmd.Flags().GetInt("limit")

	// Analysis options
	performAnalysis, _ := cmd.Flags().GetBool("analyze")
	showEvents, _ := cmd.Flags().GetBool("events")
	showTrends, _ := cmd.Flags().GetBool("trends")
	showCorrelations, _ := cmd.Flags().GetBool("correlations")

	// Enhanced filtering options
	eventTypes, _ := cmd.Flags().GetStringSlice("event-types")
	severities, _ := cmd.Flags().GetStringSlice("severities")
	minConfidenceStr, _ := cmd.Flags().GetString("min-confidence")
	incidentsOnly, _ := cmd.Flags().GetBool("incidents-only")
	compact, _ := cmd.Flags().GetBool("compact")
	includePredictions, _ := cmd.Flags().GetBool("include-predictions")

	// Parse minimum confidence
	var minConfidence float64
	if minConfidenceStr != "" {
		if conf, err := strconv.ParseFloat(minConfidenceStr, 64); err == nil {
			minConfidence = conf
		}
	}

	// Filter snapshots
	filteredSnapshots := filterSnapshots(snapshots, sinceTime, untilTime, "", "", providers)

	// Removed baseline filter - users should use --tags baseline=value instead

	// Apply tag filters
	if len(tags) > 0 {
		tagFilter := make(map[string]string)
		for _, tag := range tags {
			parts := strings.SplitN(tag, "=", 2)
			if len(parts) == 2 {
				tagFilter[parts[0]] = parts[1]
			}
		}

		var taggedSnapshots []storage.SnapshotInfo
		for _, snapshot := range filteredSnapshots {
			match := true
			for k, v := range tagFilter {
				if snapshot.Tags[k] != v {
					match = false
					break
				}
			}
			if match {
				taggedSnapshots = append(taggedSnapshots, snapshot)
			}
		}
		filteredSnapshots = taggedSnapshots
	}

	// Limit results
	if limit > 0 && len(filteredSnapshots) > limit {
		filteredSnapshots = filteredSnapshots[:limit]
	}

	// Perform advanced analysis if requested
	var timelineAnalyzer *analyzer.TimelineAnalyzer
	if performAnalysis || showEvents || showTrends || showCorrelations {
		timelineAnalyzer = analyzer.NewTimelineAnalyzer(filteredSnapshots)
		if err := timelineAnalyzer.AnalyzeTimeline(); err != nil {
			fmt.Printf("Warning: Analysis failed: %v\n", err)
		}
	}

	// Get output format
	outputFormat, _ := cmd.Flags().GetString("output")

	// Create timeline options
	timelineOptions := TimelineOptions{
		EventTypes:         eventTypes,
		Severities:         severities,
		MinConfidence:      minConfidence,
		IncidentsOnly:      incidentsOnly,
		Compact:            compact,
		IncludePredictions: includePredictions,
	}

	// Display timeline with analysis
	return displaySnapshotTimelineWithAnalysis(filteredSnapshots, outputFormat, showStats, quiet,
		timelineAnalyzer, showEvents, showTrends, showCorrelations, timelineOptions)
}

func handleTimelineBetween(localStorage storage.Storage, allSnapshots []storage.SnapshotInfo, snap1, snap2 string, cmd *cobra.Command) error {
	// Find the two snapshots
	var snapshot1, snapshot2 *storage.SnapshotInfo

	for _, snapshot := range allSnapshots {
		if snapshot.ID == snap1 || matchesSnapshotTag(snapshot, snap1) {
			snapshot1 = &snapshot
		}
		if snapshot.ID == snap2 || matchesSnapshotTag(snapshot, snap2) {
			snapshot2 = &snapshot
		}
	}

	if snapshot1 == nil {
		return fmt.Errorf("snapshot not found: %s", snap1)
	}
	if snapshot2 == nil {
		return fmt.Errorf("snapshot not found: %s", snap2)
	}

	// Ensure snapshot1 is before snapshot2
	if snapshot1.Timestamp.After(snapshot2.Timestamp) {
		snapshot1, snapshot2 = snapshot2, snapshot1
	}

	// Filter snapshots between the two snapshots
	var filteredSnapshots []storage.SnapshotInfo
	for _, snapshot := range allSnapshots {
		if snapshot.Timestamp.After(snapshot1.Timestamp) && snapshot.Timestamp.Before(snapshot2.Timestamp) {
			filteredSnapshots = append(filteredSnapshots, snapshot)
		} else if snapshot.Timestamp.Equal(snapshot1.Timestamp) || snapshot.Timestamp.Equal(snapshot2.Timestamp) {
			filteredSnapshots = append(filteredSnapshots, snapshot)
		}
	}

	// Get output options
	outputFormat, _ := cmd.Flags().GetString("output")
	showStats, _ := cmd.Flags().GetBool("stats")
	quiet, _ := cmd.Flags().GetBool("quiet")

	// Display timeline
	emptyOptions := TimelineOptions{}
	return displaySnapshotTimelineWithAnalysis(filteredSnapshots, outputFormat, showStats, quiet, nil, false, false, false, emptyOptions)
}

func parseTimeFilter(cmd *cobra.Command, flagName string) (*time.Time, error) {
	value, _ := cmd.Flags().GetString(flagName)
	if value == "" {
		return nil, nil
	}

	// Try parsing as duration first (e.g., "2 weeks ago", "3 days ago")
	if strings.Contains(value, "ago") {
		return parseDurationAgo(value)
	}

	// Try parsing as date
	formats := []string{
		"2006-01-02",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04:05Z",
		time.RFC3339,
	}

	for _, format := range formats {
		if t, err := time.Parse(format, value); err == nil {
			return &t, nil
		}
	}

	return nil, fmt.Errorf("unable to parse time: %s", value)
}

func parseDurationAgo(value string) (*time.Time, error) {
	// Simple parser for durations like "2 weeks ago", "3 days ago"
	parts := strings.Fields(value)
	if len(parts) < 3 || parts[len(parts)-1] != "ago" {
		return nil, fmt.Errorf("invalid duration format: %s", value)
	}

	amountStr := parts[0]
	unit := parts[1]

	amount, err := strconv.Atoi(amountStr)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %s", amountStr)
	}

	var duration time.Duration
	switch strings.ToLower(unit) {
	case "minute", "minutes":
		duration = time.Duration(amount) * time.Minute
	case "hour", "hours":
		duration = time.Duration(amount) * time.Hour
	case "day", "days":
		duration = time.Duration(amount) * 24 * time.Hour
	case "week", "weeks":
		duration = time.Duration(amount) * 7 * 24 * time.Hour
	case "month", "months":
		duration = time.Duration(amount) * 30 * 24 * time.Hour
	default:
		return nil, fmt.Errorf("unsupported time unit: %s", unit)
	}

	t := time.Now().Add(-duration)
	return &t, nil
}

func filterSnapshots(snapshots []storage.SnapshotInfo, since, until *time.Time, fromID, toID string, providers []string) []storage.SnapshotInfo {
	var filtered []storage.SnapshotInfo

	// Create provider filter map
	providerFilter := make(map[string]bool)
	for _, p := range providers {
		providerFilter[strings.ToLower(p)] = true
	}

	for _, snapshot := range snapshots {
		// Time filters
		if since != nil && snapshot.Timestamp.Before(*since) {
			continue
		}
		if until != nil && snapshot.Timestamp.After(*until) {
			continue
		}

		// Provider filter
		if len(providerFilter) > 0 && !providerFilter[strings.ToLower(snapshot.Provider)] {
			continue
		}

		filtered = append(filtered, snapshot)
	}

	// Sort by timestamp (oldest first for timeline view)
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Timestamp.Before(filtered[j].Timestamp)
	})

	return filtered
}

func displaySnapshotTimelineWithAnalysis(snapshots []storage.SnapshotInfo, outputFormat string, showStats, quiet bool,
	timelineAnalyzer *analyzer.TimelineAnalyzer, showEvents, showTrends, showCorrelations bool, options TimelineOptions) error {

	// First show events if requested
	if timelineAnalyzer != nil && showEvents {
		events := filterEvents(timelineAnalyzer.GetEvents(), options)
		displayTimelineEvents(events)
		fmt.Println()
	}

	// Show trends if requested
	if timelineAnalyzer != nil && showTrends {
		trends := filterTrends(timelineAnalyzer.GetTrends(), options)
		displayTimelineTrends(trends, options.IncludePredictions)
		fmt.Println()
	}

	// Show correlations if requested
	if timelineAnalyzer != nil && showCorrelations {
		correlations := filterCorrelations(timelineAnalyzer.GetCorrelations(), options)
		displayTimelineCorrelations(correlations)
		fmt.Println()
	}

	// Then show the regular timeline
	return displaySnapshotTimeline(snapshots, outputFormat, showStats, quiet, options.Compact)
}

func displaySnapshotTimeline(snapshots []storage.SnapshotInfo, outputFormat string, showStats, quiet bool, compact bool) error {
	if outputFormat == "json" {
		return timelineOutputJSON(snapshots)
	}

	if quiet {
		return displayTimelineQuiet(snapshots)
	}

	// Get terminal width
	termWidth := 80
	if width, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && width > 0 {
		termWidth = width
	}

	// Display beautiful graph timeline
	if len(snapshots) > 0 {
		graph := visualization.CreateSimpleTimeline(snapshots, termWidth)
		fmt.Println(graph)
		fmt.Println()
	}

	// Show detailed list if fewer than 10 snapshots
	if len(snapshots) <= 10 {
		fmt.Println("Snapshot Details:")
		fmt.Println(strings.Repeat("-", 60))
		for _, snapshot := range snapshots {
			fmt.Printf("📅 %s\n", snapshot.Timestamp.Format("2006-01-02 15:04:05"))
			fmt.Printf("   Provider: %s\n", snapshot.Provider)

			// Load the actual snapshot to get resource breakdown
			if snapshot.FilePath != "" {
				// Try to read the snapshot file directly
				var actualSnapshot struct {
					Resources []struct {
						ID        string            `json:"id"`
						Name      string            `json:"name"`
						Type      string            `json:"type"`
						Namespace string            `json:"namespace"`
						Provider  string            `json:"provider"`
						Labels    map[string]string `json:"labels"`
					} `json:"resources"`
				}

				file, err := os.Open(snapshot.FilePath)
				if err == nil {
					decoder := json.NewDecoder(file)
					if decoder.Decode(&actualSnapshot) == nil {
						fmt.Printf("   Resources: %d total\n", len(actualSnapshot.Resources))

						// Group resources by namespace and application
						type resourceInfo struct {
							name   string
							rtype  string
							labels map[string]string
						}

						namespaceGroups := make(map[string][]resourceInfo)

						for _, resource := range actualSnapshot.Resources {
							ns := resource.Namespace
							if ns == "" {
								ns = "cluster-wide"
							}

							namespaceGroups[ns] = append(namespaceGroups[ns], resourceInfo{
								name:   resource.Name,
								rtype:  resource.Type,
								labels: resource.Labels,
							})
						}

						// Separate system namespaces from user namespaces
						systemNamespaces := []string{"kube-system", "kube-public", "kube-node-lease"}
						userNamespaces := []string{}

						for ns := range namespaceGroups {
							isSystem := false
							for _, sysNs := range systemNamespaces {
								if ns == sysNs {
									isSystem = true
									break
								}
							}
							if !isSystem && ns != "cluster-wide" {
								userNamespaces = append(userNamespaces, ns)
							}
						}

						// Sort user namespaces
						sort.Strings(userNamespaces)

						// Display user workloads first
						if len(userNamespaces) > 0 {
							fmt.Printf("   Your Workloads:\n")
							for _, ns := range userNamespaces {
								resources := namespaceGroups[ns]
								fmt.Printf("     • %s namespace: %d resources\n", ns, len(resources))

								// Group resources by application using labels
								type appGroup struct {
									appType   string
									resources []resourceInfo
								}

								appGroups := make(map[string]*appGroup)
								var ungrouped []resourceInfo

								// Try to group by common app labels
								for _, res := range resources {
									appName := ""

									// Try common label patterns
									if res.labels != nil {
										if name := res.labels["app"]; name != "" {
											appName = name
										} else if name := res.labels["app.kubernetes.io/name"]; name != "" {
											appName = name
										} else if name := res.labels["k8s-app"]; name != "" {
											appName = name
										}
									}

									// If no app label, try to extract from name
									if appName == "" {
										// Common pattern: deployment/service names often match
										if res.rtype == "deployment" || res.rtype == "service" || res.rtype == "statefulset" {
											appName = res.name
										}
									}

									if appName != "" {
										if appGroups[appName] == nil {
											appGroups[appName] = &appGroup{
												appType: detectAppType(appName),
											}
										}
										appGroups[appName].resources = append(appGroups[appName].resources, res)
									} else {
										ungrouped = append(ungrouped, res)
									}
								}

								// Display grouped applications
								if len(appGroups) > 0 {
									for appName, group := range appGroups {
										// Count resource types in this app
										appResources := make(map[string][]string)
										for _, res := range group.resources {
											appResources[res.rtype] = append(appResources[res.rtype], res.name)
										}

										// Display app with type hint
										typeHint := ""
										if group.appType != "service" {
											typeHint = fmt.Sprintf(" (%s)", group.appType)
										}
										fmt.Printf("       \"%s\"%s:\n", appName, typeHint)

										// Show main workload types
										if deps := appResources["deployment"]; len(deps) > 0 {
											fmt.Printf("         - Deployment: %s\n", deps[0])
										}
										if sts := appResources["statefulset"]; len(sts) > 0 {
											fmt.Printf("         - StatefulSet: %s\n", sts[0])
										}
										if svcs := appResources["service"]; len(svcs) > 0 {
											fmt.Printf("         - Service: %s", svcs[0])
											if len(svcs) > 1 {
												fmt.Printf(" +%d more", len(svcs)-1)
											}
											fmt.Println()
										}

										// Summarize other resources
										var others []string
										for rtype, items := range appResources {
											if rtype != "deployment" && rtype != "statefulset" && rtype != "service" && rtype != "pod" {
												others = append(others, fmt.Sprintf("%d %s", len(items), rtype))
											}
										}
										if len(others) > 0 {
											fmt.Printf("         - Resources: %s\n", strings.Join(others, ", "))
										}
									}
								}

								// Display ungrouped resources as fallback
								if len(ungrouped) > 0 {
									// Group ungrouped by type
									ungroupedByType := make(map[string][]string)
									for _, res := range ungrouped {
										ungroupedByType[res.rtype] = append(ungroupedByType[res.rtype], res.name)
									}

									if len(ungroupedByType) > 0 {
										fmt.Println("       Standalone resources:")

										// Show important types first
										if pods := ungroupedByType["pod"]; len(pods) > 0 {
											fmt.Printf("         - Pods: %s", pods[0])
											for i := 1; i < len(pods) && i < 3; i++ {
												fmt.Printf(", %s", pods[i])
											}
											if len(pods) > 3 {
												fmt.Printf(" +%d more", len(pods)-3)
											}
											fmt.Println()
										}

										// Other types
										var otherSummary []string
										for rtype, names := range ungroupedByType {
											if rtype != "pod" {
												otherSummary = append(otherSummary, fmt.Sprintf("%d %s", len(names), rtype))
											}
										}
										if len(otherSummary) > 0 {
											fmt.Printf("         - Other: %s\n", strings.Join(otherSummary, ", "))
										}
									}
								}
							}
						} else {
							fmt.Printf("   Your Workloads: None detected\n")
						}

						// Show system summary
						systemTotal := 0
						for _, ns := range systemNamespaces {
							if resources, exists := namespaceGroups[ns]; exists {
								systemTotal += len(resources)
							}
						}

						if clusterWide, exists := namespaceGroups["cluster-wide"]; exists {
							systemTotal += len(clusterWide)
						}

						if systemTotal > 0 {
							fmt.Printf("   System Resources: %d Kubernetes internals\n", systemTotal)
						}

					} else {
						fmt.Printf("   Resources: %d\n", snapshot.ResourceCount)
					}
					file.Close()
				} else {
					fmt.Printf("   Resources: %d\n", snapshot.ResourceCount)
				}
			} else {
				fmt.Printf("   Resources: %d\n", snapshot.ResourceCount)
			}

			fmt.Printf("   Created: %s\n", snapshot.Timestamp.Format("2006-01-02 15:04:05"))
			fmt.Printf("   ID: %s\n", snapshot.ID)

			if len(snapshot.Tags) > 0 {
				fmt.Print("   Tags: ")
				var tags []string
				for k, v := range snapshot.Tags {
					tags = append(tags, fmt.Sprintf("%s=%s", k, v))
				}
				fmt.Println(strings.Join(tags, ", "))
			}

			fmt.Println()
		}
	}

	if showStats {
		displaySnapshotStats(snapshots)
	}

	if !quiet {
		fmt.Println("For advanced change timeline with correlation analysis, use:")
		fmt.Println("   vaino changes --timeline")
	}

	return nil
}

func timelineOutputJSON(snapshots []storage.SnapshotInfo) error {
	// Convert to a simple JSON structure
	type TimelineEntry struct {
		Timestamp     time.Time         `json:"timestamp"`
		Provider      string            `json:"provider"`
		ResourceCount int               `json:"resource_count"`
		ID            string            `json:"id"`
		Tags          map[string]string `json:"tags,omitempty"`
	}

	var entries []TimelineEntry
	for _, snapshot := range snapshots {
		entries = append(entries, TimelineEntry{
			Timestamp:     snapshot.Timestamp,
			Provider:      snapshot.Provider,
			ResourceCount: snapshot.ResourceCount,
			ID:            snapshot.ID,
			Tags:          snapshot.Tags,
		})
	}

	jsonData, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(jsonData))
	return nil
}

func displayTimelineQuiet(snapshots []storage.SnapshotInfo) error {
	for _, snapshot := range snapshots {
		fmt.Printf("%s %s %d\n",
			snapshot.Timestamp.Format("2006-01-02T15:04:05"),
			snapshot.Provider,
			snapshot.ResourceCount)
	}
	return nil
}

func displaySnapshotStats(snapshots []storage.SnapshotInfo) {
	if len(snapshots) == 0 {
		return
	}

	fmt.Println("Snapshot Statistics")
	fmt.Println(strings.Repeat("-", 30))

	// Provider distribution
	providers := make(map[string]int)
	totalResources := 0

	for _, snapshot := range snapshots {
		providers[snapshot.Provider]++
		totalResources += snapshot.ResourceCount
	}

	fmt.Printf("Total snapshots: %d\n", len(snapshots))
	fmt.Printf("Date range: %s to %s\n",
		snapshots[0].Timestamp.Format("2006-01-02"),
		snapshots[len(snapshots)-1].Timestamp.Format("2006-01-02"))
	fmt.Printf("Total resources: %d\n", totalResources)
	fmt.Printf("Average resources per snapshot: %.1f\n", float64(totalResources)/float64(len(snapshots)))

	fmt.Println("\nProvider distribution:")
	for provider, count := range providers {
		fmt.Printf("  %s: %d snapshots\n", provider, count)
	}
	fmt.Println()
}

// matchesSnapshotTag checks if a snapshot has any tag value matching the given string
func matchesSnapshotTag(snapshot storage.SnapshotInfo, value string) bool {
	for _, tagValue := range snapshot.Tags {
		if tagValue == value {
			return true
		}
	}
	return false
}

// detectAppType tries to identify the type of application based on its name
func detectAppType(name string) string {
	lowerName := strings.ToLower(name)

	// Database patterns
	if strings.Contains(lowerName, "postgres") ||
		strings.Contains(lowerName, "mysql") ||
		strings.Contains(lowerName, "mongo") ||
		strings.Contains(lowerName, "mariadb") ||
		strings.Contains(lowerName, "cassandra") {
		return "database"
	}

	// Cache/Queue patterns
	if strings.Contains(lowerName, "redis") ||
		strings.Contains(lowerName, "memcache") ||
		strings.Contains(lowerName, "rabbitmq") ||
		strings.Contains(lowerName, "kafka") ||
		strings.Contains(lowerName, "queue") {
		return "cache/queue"
	}

	// Frontend patterns
	if strings.Contains(lowerName, "frontend") ||
		strings.Contains(lowerName, "ui") ||
		strings.Contains(lowerName, "web") ||
		strings.Contains(lowerName, "nginx") ||
		strings.Contains(lowerName, "apache") {
		return "frontend"
	}

	// API/Backend patterns
	if strings.Contains(lowerName, "api") ||
		strings.Contains(lowerName, "backend") ||
		strings.Contains(lowerName, "server") {
		return "backend"
	}

	// Monitoring/Logging
	if strings.Contains(lowerName, "prometheus") ||
		strings.Contains(lowerName, "grafana") ||
		strings.Contains(lowerName, "elastic") ||
		strings.Contains(lowerName, "logstash") ||
		strings.Contains(lowerName, "fluentd") {
		return "monitoring"
	}

	return "service" // default
}

// getHumanReadableResourceName converts technical resource types to contextual descriptions
func getHumanReadableResourceName(resourceType string) string {
	// Map of technical names to contextual descriptions
	contextualNames := map[string]string{
		// Kubernetes workloads
		"pod":         "application pods running containers",
		"deployment":  "managed application deployments",
		"replicaset":  "pod replica controllers",
		"daemonset":   "system services on each node",
		"statefulset": "stateful applications with persistent identity",
		"job":         "batch jobs",
		"cronjob":     "scheduled recurring jobs",

		// Kubernetes networking
		"service":       "network services for pod communication",
		"ingress":       "external traffic routing rules",
		"networkpolicy": "network security policies",
		"endpointslice": "network endpoint mappings",
		"endpoints":     "service endpoint configurations",

		// Kubernetes security & identity
		"serviceaccount":     "identity accounts for pods & services",
		"role":               "permission sets for namespace access",
		"rolebinding":        "assignments linking users to roles",
		"clusterrole":        "cluster-wide permission templates",
		"clusterrolebinding": "cluster-wide role assignments",
		"secret":             "encrypted credential & certificate storage",

		// Kubernetes configuration & storage
		"configmap":               "application configuration data",
		"persistentvolume":        "cluster storage volumes",
		"persistentvolumeclaim":   "storage requests from applications",
		"namespace":               "isolated resource groups",
		"node":                    "worker machines in the cluster",
		"horizontalpodautoscaler": "automatic scaling rules for workloads",

		// AWS compute & networking
		"aws_instance":          "virtual machines in EC2",
		"aws_vpc":               "private network environments",
		"aws_subnet":            "network segments within VPCs",
		"aws_security_group":    "firewall rules for instances",
		"aws_load_balancer":     "traffic distribution services",
		"aws_autoscaling_group": "automatically scaling server groups",

		// AWS storage & databases
		"aws_s3_bucket":       "object storage containers",
		"aws_rds_instance":    "managed database instances",
		"aws_ebs_volume":      "block storage for EC2 instances",
		"aws_efs_file_system": "shared network file systems",

		// AWS identity & access
		"aws_iam_role":   "service permission sets",
		"aws_iam_user":   "human user accounts",
		"aws_iam_policy": "detailed permission documents",
		"aws_iam_group":  "user collections for permission management",

		// GCP compute & networking
		"gcp_compute_instance": "virtual machines in Compute Engine",
		"gcp_vpc_network":      "private cloud networks",
		"gcp_compute_firewall": "network access control rules",
		"gcp_compute_address":  "static IP address reservations",

		// GCP storage & databases
		"gcp_storage_bucket": "object storage containers",
		"gcp_sql_instance":   "managed database instances",
		"gcp_compute_disk":   "persistent storage disks",

		// GCP identity & access
		"gcp_iam_binding":     "permission assignments to resources",
		"gcp_service_account": "service identity accounts",
		"gcp_project":         "resource organization containers",

		// Terraform management
		"terraform_state": "infrastructure state tracking",
		"null_resource":   "execution triggers & dependencies",
		"local_file":      "generated configuration files",
		"random_id":       "unique identifiers for resources",
		"data":            "read-only information sources",
	}

	// Return contextual description if available
	if contextualName, exists := contextualNames[strings.ToLower(resourceType)]; exists {
		return contextualName
	}

	// Default: convert snake_case to Title Case with context
	words := strings.Split(strings.ReplaceAll(resourceType, "_", " "), " ")
	for i, word := range words {
		words[i] = strings.Title(strings.ToLower(word))
	}
	result := strings.Join(words, " ")

	// Add generic context based on common patterns
	if strings.Contains(strings.ToLower(resourceType), "policy") {
		return result + " (security policies)"
	} else if strings.Contains(strings.ToLower(resourceType), "network") {
		return result + " (networking components)"
	} else if strings.Contains(strings.ToLower(resourceType), "storage") || strings.Contains(strings.ToLower(resourceType), "volume") {
		return result + " (storage resources)"
	} else if strings.Contains(strings.ToLower(resourceType), "instance") || strings.Contains(strings.ToLower(resourceType), "vm") {
		return result + " (compute instances)"
	}

	return result + " (infrastructure components)"
}

// filterEvents filters events based on the provided options
func filterEvents(events []analyzer.TimelineEvent, options TimelineOptions) []analyzer.TimelineEvent {
	if len(options.EventTypes) == 0 && len(options.Severities) == 0 && !options.IncidentsOnly {
		return events // No filtering needed
	}

	var filtered []analyzer.TimelineEvent

	// Create filter maps for quick lookup
	eventTypeFilter := make(map[string]bool)
	for _, eventType := range options.EventTypes {
		eventTypeFilter[eventType] = true
	}

	severityFilter := make(map[string]bool)
	for _, severity := range options.Severities {
		severityFilter[severity] = true
	}

	for _, event := range events {
		// Skip if event type doesn't match filter
		if len(eventTypeFilter) > 0 && !eventTypeFilter[event.Type] {
			continue
		}

		// Skip if severity doesn't match filter
		if len(severityFilter) > 0 && !severityFilter[event.Severity] {
			continue
		}

		// If incidents-only is set, only show critical events or large-scale changes
		if options.IncidentsOnly {
			if event.Severity != "critical" && event.Type != "infrastructure_change" && event.Type != "sustained_decline" {
				continue
			}
		}

		filtered = append(filtered, event)
	}

	return filtered
}

// filterTrends filters trends based on the provided options
func filterTrends(trends []analyzer.TimelineTrend, options TimelineOptions) []analyzer.TimelineTrend {
	if options.MinConfidence == 0 {
		return trends // No confidence filtering needed
	}

	var filtered []analyzer.TimelineTrend

	for _, trend := range trends {
		if trend.Confidence >= options.MinConfidence {
			filtered = append(filtered, trend)
		}
	}

	return filtered
}

// filterCorrelations filters correlations based on the provided options
func filterCorrelations(correlations []analyzer.CorrelationAnalysis, options TimelineOptions) []analyzer.CorrelationAnalysis {
	if options.MinConfidence == 0 {
		return correlations // No confidence filtering needed
	}

	var filtered []analyzer.CorrelationAnalysis

	for _, correlation := range correlations {
		if correlation.Confidence >= options.MinConfidence {
			filtered = append(filtered, correlation)
		}
	}

	return filtered
}

// displayTimelineEvents displays detected timeline events
func displayTimelineEvents(events []analyzer.TimelineEvent) {
	if len(events) == 0 {
		fmt.Println("Timeline Events: None detected")
		return
	}

	fmt.Printf("Timeline Events (%d detected):\n", len(events))
	fmt.Println(strings.Repeat("-", 50))

	// Group events by severity
	severityGroups := map[string][]analyzer.TimelineEvent{
		"critical": {},
		"warning":  {},
		"info":     {},
	}

	for _, event := range events {
		severityGroups[event.Severity] = append(severityGroups[event.Severity], event)
	}

	// Display by severity (critical first)
	for _, severity := range []string{"critical", "warning", "info"} {
		if len(severityGroups[severity]) == 0 {
			continue
		}

		var icon string
		switch severity {
		case "critical":
			icon = "🔴"
		case "warning":
			icon = "🟡"
		case "info":
			icon = "🔵"
		}

		fmt.Printf("\n%s %s Events:\n", icon, strings.Title(severity))
		for _, event := range severityGroups[severity] {
			fmt.Printf("  %s - %s [%s]\n",
				event.Timestamp.Format("2006-01-02 15:04:05"),
				event.Description,
				event.Provider)

			if event.Context != nil {
				if countChange, ok := event.Context["count_change"].(int); ok {
					fmt.Printf("    Resource change: %+d\n", countChange)
				}
				if timeDiff, ok := event.Context["time_diff"].(time.Duration); ok {
					fmt.Printf("    Time since last: %v\n", timeDiff)
				}
			}
		}
	}
}

// displayTimelineTrends displays trend analysis results
func displayTimelineTrends(trends []analyzer.TimelineTrend, includePredictions bool) {
	if len(trends) == 0 {
		fmt.Println("Timeline Trends: None detected")
		return
	}

	fmt.Printf("Timeline Trends (%d detected):\n", len(trends))
	fmt.Println(strings.Repeat("-", 50))

	for _, trend := range trends {
		var icon string
		switch trend.Trend {
		case "increasing":
			icon = "📈"
		case "decreasing":
			icon = "📉"
		case "stable":
			icon = "➖"
		case "volatile":
			icon = "📊"
		default:
			icon = "📋"
		}

		fmt.Printf("\n%s %s - %s trend (%.1f%% confidence)\n",
			icon, trend.Provider, trend.Trend, trend.Confidence*100)

		fmt.Printf("  Time period: %s to %s\n",
			trend.StartTime.Format("2006-01-02"),
			trend.EndTime.Format("2006-01-02"))

		if len(trend.DataPoints) > 0 {
			first := trend.DataPoints[0]
			last := trend.DataPoints[len(trend.DataPoints)-1]
			change := last.Value - first.Value
			fmt.Printf("  Resource change: %d → %d (%+d)\n", first.Value, last.Value, change)
		}

		// Show predictions if available and requested
		if includePredictions && len(trend.Predictions) > 0 {
			fmt.Printf("  Predictions:\n")
			for i, pred := range trend.Predictions {
				if i >= 2 { // Show max 2 predictions
					break
				}
				fmt.Printf("    %s: %d resources (%.1f%% confidence)\n",
					pred.FutureTime.Format("2006-01-02"),
					pred.PredictedValue,
					pred.Confidence*100)
			}
		}
	}
}

// displayTimelineCorrelations displays correlation analysis results
func displayTimelineCorrelations(correlations []analyzer.CorrelationAnalysis) {
	if len(correlations) == 0 {
		fmt.Println("Timeline Correlations: None detected")
		return
	}

	fmt.Printf("Timeline Correlations (%d detected):\n", len(correlations))
	fmt.Println(strings.Repeat("-", 50))

	for _, corr := range correlations {
		var icon string
		var strength string
		absCorr := corr.Correlation
		if absCorr < 0 {
			absCorr = -absCorr
		}

		if absCorr > 0.7 {
			strength = "Strong"
			icon = "🔗"
		} else if absCorr > 0.5 {
			strength = "Moderate"
			icon = "🔗"
		} else {
			strength = "Weak"
			icon = "🔗"
		}

		var direction string
		if corr.Correlation > 0 {
			direction = "positive"
		} else {
			direction = "negative"
		}

		fmt.Printf("\n%s %s %s correlation between %s and %s\n",
			icon, strength, direction, corr.Provider1, corr.Provider2)

		fmt.Printf("  Correlation coefficient: %.3f (%.1f%% confidence)\n",
			corr.Correlation, corr.Confidence*100)

		// Explain what this means
		if corr.Correlation > 0.3 {
			fmt.Printf("  → When %s resources increase, %s resources tend to increase\n",
				corr.Provider1, corr.Provider2)
		} else if corr.Correlation < -0.3 {
			fmt.Printf("  → When %s resources increase, %s resources tend to decrease\n",
				corr.Provider1, corr.Provider2)
		}

		if len(corr.Examples) > 0 {
			fmt.Printf("  Recent examples:\n")
			for i, example := range corr.Examples {
				if i >= 2 { // Show max 2 examples
					break
				}
				fmt.Printf("    %s: %s\n",
					example.Timestamp.Format("2006-01-02"),
					example.Description)
			}
		}
	}
}
