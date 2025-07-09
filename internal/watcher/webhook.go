package watcher

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// WebhookPayload represents the payload sent to webhooks
type WebhookPayload struct {
	Timestamp  time.Time       `json:"timestamp"`
	Source     string          `json:"source"`
	Summary    WebhookSummary  `json:"summary"`
	Groups     []WebhookGroup  `json:"groups,omitempty"`
	RawChanges []WebhookChange `json:"raw_changes,omitempty"`
	Metadata   WebhookMetadata `json:"metadata"`
}

// WebhookSummary provides a summary of changes
type WebhookSummary struct {
	Total      int `json:"total"`
	Added      int `json:"added"`
	Modified   int `json:"modified"`
	Removed    int `json:"removed"`
	HighConf   int `json:"high_confidence_groups"`
	MediumConf int `json:"medium_confidence_groups"`
	LowConf    int `json:"low_confidence_groups"`
}

// WebhookGroup represents a correlated group for webhooks
type WebhookGroup struct {
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Confidence  string          `json:"confidence"`
	ChangeCount int             `json:"change_count"`
	Changes     []WebhookChange `json:"changes"`
	Reason      string          `json:"reason"`
}

// WebhookChange represents an individual change for webhooks
type WebhookChange struct {
	Type         string    `json:"type"`
	ResourceType string    `json:"resource_type"`
	ResourceName string    `json:"resource_name"`
	Namespace    string    `json:"namespace,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
}

// WebhookMetadata provides additional context
type WebhookMetadata struct {
	WatchInterval string   `json:"watch_interval"`
	Providers     []string `json:"providers"`
	Version       string   `json:"version"`
}

// sendWebhook sends change notification to configured webhook URL
func (w *Watcher) sendWebhook(event *WatchEvent) error {
	if w.webhookURL == "" {
		return nil
	}

	payload := w.buildWebhookPayload(event)

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	// Send with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Post(w.webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned error status: %d", resp.StatusCode)
	}

	if !w.quiet {
		fmt.Printf("📡 Webhook sent successfully (%d)\n", resp.StatusCode)
	}

	return nil
}

// buildWebhookPayload converts a WatchEvent to webhook format
func (w *Watcher) buildWebhookPayload(event *WatchEvent) WebhookPayload {
	// Build summary with confidence counts
	summary := WebhookSummary{
		Total:    event.Summary.Total,
		Added:    event.Summary.Added,
		Modified: event.Summary.Modified,
		Removed:  event.Summary.Removed,
	}

	// Count confidence levels
	for _, group := range event.CorrelatedGroups {
		switch group.Confidence {
		case "high":
			summary.HighConf++
		case "medium":
			summary.MediumConf++
		case "low":
			summary.LowConf++
		}
	}

	// Convert correlated groups
	var webhookGroups []WebhookGroup
	for _, group := range event.CorrelatedGroups {
		webhookGroup := WebhookGroup{
			Title:       group.Title,
			Description: group.Description,
			Confidence:  group.Confidence,
			ChangeCount: len(group.Changes),
			Reason:      group.Reason,
		}

		// Convert changes in the group
		for _, change := range group.Changes {
			webhookGroup.Changes = append(webhookGroup.Changes, WebhookChange{
				Type:         change.Type,
				ResourceType: change.ResourceType,
				ResourceName: change.ResourceName,
				Namespace:    change.Namespace,
				Timestamp:    change.Timestamp,
			})
		}

		webhookGroups = append(webhookGroups, webhookGroup)
	}

	// Convert raw changes (for ungrouped changes)
	var rawChanges []WebhookChange
	for _, change := range event.RawChanges {
		rawChanges = append(rawChanges, WebhookChange{
			Type:         change.Type,
			ResourceType: change.ResourceType,
			ResourceName: change.ResourceName,
			Namespace:    change.Namespace,
			Timestamp:    change.Timestamp,
		})
	}

	return WebhookPayload{
		Timestamp:  event.Timestamp,
		Source:     event.Source,
		Summary:    summary,
		Groups:     webhookGroups,
		RawChanges: rawChanges,
		Metadata: WebhookMetadata{
			WatchInterval: w.interval.String(),
			Providers:     w.providers,
			Version:       "1.0.0", // This would come from build info
		},
	}
}

// sendSlackWebhook sends a Slack-formatted webhook
func (w *Watcher) sendSlackWebhook(event *WatchEvent) error {
	if w.webhookURL == "" {
		return nil
	}

	// Build Slack-specific payload
	payload := map[string]interface{}{
		"text": fmt.Sprintf("🔍 Infrastructure changes detected at %s",
			event.Timestamp.Format("15:04:05")),
		"attachments": []map[string]interface{}{
			{
				"color": w.getSlackColor(event),
				"fields": []map[string]interface{}{
					{
						"title": "Summary",
						"value": fmt.Sprintf("%d total changes (%d added, %d modified, %d removed)",
							event.Summary.Total,
							event.Summary.Added,
							event.Summary.Modified,
							event.Summary.Removed),
						"short": false,
					},
				},
			},
		},
	}

	// Add correlated groups as fields
	if len(event.CorrelatedGroups) > 0 {
		var groupTexts []string
		for _, group := range event.CorrelatedGroups {
			confidence := w.getConfidenceIndicator(group.Confidence)
			groupTexts = append(groupTexts, fmt.Sprintf("%s %s (%d changes)",
				confidence, group.Title, len(group.Changes)))
		}

		attachment := payload["attachments"].([]map[string]interface{})[0]
		fields := attachment["fields"].([]map[string]interface{})
		fields = append(fields, map[string]interface{}{
			"title": "Correlated Changes",
			"value": fmt.Sprintf("```\n%s\n```", strings.Join(groupTexts, "\n")),
			"short": false,
		})
		attachment["fields"] = fields
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Slack payload: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(w.webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send Slack webhook: %w", err)
	}
	defer resp.Body.Close()

	return nil
}

// getSlackColor returns appropriate color for Slack attachments
func (w *Watcher) getSlackColor(event *WatchEvent) string {
	// Determine color based on change types and confidence
	hasHighConf := false
	for _, group := range event.CorrelatedGroups {
		if group.Confidence == "high" {
			hasHighConf = true
			break
		}
	}

	if hasHighConf || event.Summary.Removed > 0 {
		return "danger" // Red for high confidence or deletions
	} else if event.Summary.Added > 0 {
		return "good" // Green for additions
	} else {
		return "warning" // Yellow for modifications
	}
}
