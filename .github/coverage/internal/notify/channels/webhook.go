// Package channels provides generic webhook notification channel implementation
package channels

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/mrz1836/go-broadcast/coverage/internal/types"
)

// Static error definitions
var (
	ErrWebhookConfigNil      = errors.New("webhook config is nil")
	ErrWebhookURLRequired    = errors.New("webhook URL is required")
	ErrWebhookURLInvalid     = errors.New("invalid webhook URL format")
	ErrUnsupportedHTTPMethod = errors.New("unsupported HTTP method")
	ErrWebhookStatusError    = errors.New("webhook endpoint returned error status")
)

// WebhookChannel implements generic webhook notifications
type WebhookChannel struct {
	config    *types.WebhookConfig
	rateLimit *types.RateLimit
	client    *http.Client
	template  *template.Template
}

// WebhookPayload represents a generic webhook payload
type WebhookPayload struct {
	// Standard fields
	Event     string    `json:"event"`
	Timestamp time.Time `json:"timestamp"`
	Source    string    `json:"source"`
	Version   string    `json:"version"`

	// Notification data
	Notification *NotificationData `json:"notification"`

	// Repository context
	Repository *RepositoryContext `json:"repository,omitempty"`

	// Coverage data
	Coverage *CoverageContext `json:"coverage,omitempty"`

	// Custom data
	Custom map[string]interface{} `json:"custom,omitempty"`
}

// NotificationData represents notification information for webhooks
type NotificationData struct {
	ID       string            `json:"id"`
	Type     string            `json:"type"`
	Severity string            `json:"severity"`
	Priority string            `json:"priority"`
	Urgency  string            `json:"urgency"`
	Subject  string            `json:"subject"`
	Message  string            `json:"message"`
	Author   string            `json:"author,omitempty"`
	Actions  []ActionData      `json:"actions,omitempty"`
	Links    []LinkData        `json:"links,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// RepositoryContext represents repository information
type RepositoryContext struct {
	Name      string `json:"name"`
	Branch    string `json:"branch"`
	CommitSHA string `json:"commit_sha,omitempty"`
	PRNumber  int    `json:"pr_number,omitempty"`
	URL       string `json:"url,omitempty"`
	PRURL     string `json:"pr_url,omitempty"`
	CommitURL string `json:"commit_url,omitempty"`
}

// CoverageContext represents coverage information
type CoverageContext struct {
	Current      float64              `json:"current"`
	Previous     float64              `json:"previous,omitempty"`
	Change       float64              `json:"change,omitempty"`
	Target       float64              `json:"target,omitempty"`
	Threshold    float64              `json:"threshold,omitempty"`
	Status       string               `json:"status"`
	Trend        *TrendContext        `json:"trend,omitempty"`
	QualityGates *QualityGatesContext `json:"quality_gates,omitempty"`
}

// TrendContext represents trend information
type TrendContext struct {
	Direction  string  `json:"direction"`
	Magnitude  string  `json:"magnitude"`
	Confidence float64 `json:"confidence"`
	Prediction float64 `json:"prediction,omitempty"`
}

// QualityGatesContext represents quality gate information
type QualityGatesContext struct {
	Passed      bool     `json:"passed"`
	Failed      []string `json:"failed,omitempty"`
	TotalGates  int      `json:"total_gates"`
	PassedGates int      `json:"passed_gates"`
}

// ActionData represents action information
type ActionData struct {
	Text  string `json:"text"`
	URL   string `json:"url"`
	Type  string `json:"type"`
	Style string `json:"style,omitempty"`
}

// LinkData is defined in email.go

// NewWebhookChannel creates a new generic webhook notification channel
func NewWebhookChannel(config *types.WebhookConfig) *WebhookChannel {
	channel := &WebhookChannel{
		config: config,
		rateLimit: &types.RateLimit{
			RequestsPerMinute: 60, // Configurable rate limit
			RequestsPerHour:   3600,
			RequestsPerDay:    86400,
			BurstSize:         10,
		},
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Custom template not available in current config structure

	return channel
}

// Send implements the NotificationChannel interface for webhooks
func (w *WebhookChannel) Send(ctx context.Context, notification *types.Notification) (*types.DeliveryResult, error) {
	startTime := time.Now()
	result := &types.DeliveryResult{
		Channel:   types.ChannelWebhook,
		Timestamp: startTime,
	}

	// Build webhook payload
	payload, err := w.buildWebhookPayload(notification)
	if err != nil {
		result.Error = fmt.Errorf("failed to build webhook payload: %w", err)
		return result, result.Error
	}

	// Send webhook request
	err = w.sendWebhookRequest(ctx, payload)
	if err != nil {
		result.Error = fmt.Errorf("failed to send webhook: %w", err)
		return result, result.Error
	}

	result.DeliveryTime = time.Since(startTime)
	result.Success = true
	result.MessageID = fmt.Sprintf("webhook_%d", time.Now().Unix())

	return result, nil
}

// ValidateConfig validates the webhook channel configuration
func (w *WebhookChannel) ValidateConfig() error {
	if w.config == nil {
		return ErrWebhookConfigNil
	}

	if w.config.URL == "" {
		return ErrWebhookURLRequired
	}

	// Validate URL format
	if !isValidWebhookURL(w.config.URL) {
		return ErrWebhookURLInvalid
	}

	// Validate HTTP method
	if w.config.Method == "" {
		w.config.Method = "POST"
	} else {
		method := strings.ToUpper(w.config.Method)
		if method != "POST" && method != "PUT" && method != "PATCH" {
			return fmt.Errorf("%w: %s", ErrUnsupportedHTTPMethod, w.config.Method)
		}
	}

	// Validate content type
	if w.config.ContentType == "" {
		w.config.ContentType = "application/json"
	}

	return nil
}

// GetChannelType returns the channel type
func (w *WebhookChannel) GetChannelType() types.ChannelType {
	return types.ChannelWebhook
}

// SupportsRichContent returns whether the channel supports rich content
func (w *WebhookChannel) SupportsRichContent() bool {
	return true
}

// GetRateLimit returns the rate limit configuration
func (w *WebhookChannel) GetRateLimit() *types.RateLimit {
	return w.rateLimit
}

// buildWebhookPayload builds the webhook payload from a notification
func (w *WebhookChannel) buildWebhookPayload(notification *types.Notification) ([]byte, error) {
	// Use custom template if available
	if w.template != nil {
		return w.buildCustomPayload(notification)
	}

	// Build standard payload
	payload := &WebhookPayload{
		Event:     string("coverage_event"),
		Timestamp: notification.Timestamp,
		Source:    "gofortress-coverage",
		Version:   "1.0.0",
	}

	// Build notification data
	payload.Notification = &NotificationData{
		ID:       notification.ID,
		Type:     "coverage_notification",
		Severity: string(notification.Severity),
		Priority: string(notification.Priority),
		Urgency:  string(notification.Priority),
		Subject:  notification.Subject,
		Message:  notification.Message,
		Author:   notification.Author,
		Actions:  make([]ActionData, 0), // No actions in base notification type
		Links:    make([]LinkData, 0),   // No links in base notification type
		Metadata: make(map[string]string),
	}

	// Actions and links not available in base notification type

	// Build repository context
	if notification.Repository != "" {
		payload.Repository = &RepositoryContext{
			Name:      notification.Repository,
			Branch:    notification.Branch,
			CommitSHA: notification.CommitSHA,
			PRNumber:  notification.PRNumber,
		}

		// Generate URLs
		if notification.Repository != "" {
			payload.Repository.URL = fmt.Sprintf("https://github.com/%s", notification.Repository)

			if notification.CommitSHA != "" {
				payload.Repository.CommitURL = fmt.Sprintf("https://github.com/%s/commit/%s", notification.Repository, notification.CommitSHA)
			}

			if notification.PRNumber > 0 {
				payload.Repository.PRURL = fmt.Sprintf("https://github.com/%s/pull/%d", notification.Repository, notification.PRNumber)
			}
		}
	}

	// Build coverage context
	if notification.CoverageData != nil {
		payload.Coverage = &CoverageContext{
			Current:   notification.CoverageData.Current,
			Previous:  notification.CoverageData.Previous,
			Change:    notification.CoverageData.Change,
			Target:    notification.CoverageData.Target,
			Threshold: notification.CoverageData.Target,
			Status:    w.getCoverageStatus(notification.CoverageData),
		}

		// Add trend context
		if notification.TrendData != nil {
			payload.Coverage.Trend = &TrendContext{
				Direction:  notification.TrendData.Direction,
				Magnitude:  notification.TrendData.Direction,
				Confidence: notification.TrendData.Confidence,
				Prediction: notification.TrendData.Confidence,
			}
		}
	}

	// Add custom data
	payload.Custom = make(map[string]interface{})
	payload.Custom["generator"] = "GoFortress Coverage System"
	payload.Custom["format_version"] = "1.0"

	// Marshal to JSON
	return json.Marshal(payload) //nolint:musttag // WebhookPayload has JSON tags
}

// buildCustomPayload builds a custom payload using the configured template
func (w *WebhookChannel) buildCustomPayload(notification *types.Notification) ([]byte, error) {
	// Prepare template data
	templateData := map[string]interface{}{
		"notification": notification,
		"timestamp":    notification.Timestamp,
		"event":        string("coverage_event"),
		"severity":     string(notification.Severity),
		"priority":     string(notification.Priority),
		"repository":   notification.Repository,
		"branch":       notification.Branch,
		"author":       notification.Author,
		"pr_number":    notification.PRNumber,
		"commit_sha":   notification.CommitSHA,
	}

	// Add coverage data if available
	if notification.CoverageData != nil {
		templateData["coverage"] = map[string]interface{}{
			"current":   notification.CoverageData.Current,
			"previous":  notification.CoverageData.Previous,
			"change":    notification.CoverageData.Change,
			"target":    notification.CoverageData.Target,
			"threshold": notification.CoverageData.Target,
		}
	}

	// Add trend data if available
	if notification.TrendData != nil {
		templateData["trend"] = map[string]interface{}{
			"direction":  notification.TrendData.Direction,
			"magnitude":  notification.TrendData.Direction,
			"confidence": notification.TrendData.Confidence,
			"prediction": notification.TrendData.Confidence,
		}
	}

	// Execute template
	var buf bytes.Buffer
	err := w.template.Execute(&buf, templateData)
	if err != nil {
		return nil, fmt.Errorf("failed to execute custom template: %w", err)
	}

	return buf.Bytes(), nil
}

// sendWebhookRequest sends the webhook HTTP request
func (w *WebhookChannel) sendWebhookRequest(ctx context.Context, payload []byte) error {
	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, w.config.Method, w.config.URL, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", w.config.ContentType)
	req.Header.Set("User-Agent", "GoFortress-Coverage/1.0")

	// Add custom headers
	for key, value := range w.config.Headers {
		req.Header.Set(key, value)
	}

	// Add authentication if configured
	if w.config.AuthToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", w.config.AuthToken))
	}

	// Send request
	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%w: %d", ErrWebhookStatusError, resp.StatusCode)
	}

	return nil
}

// getCoverageStatus determines the coverage status based on data
func (w *WebhookChannel) getCoverageStatus(coverageData *types.CoverageData) string {
	if coverageData.Target > 0 {
		if coverageData.Current >= coverageData.Target {
			return "target_met"
		}
		if coverageData.Current >= coverageData.Target*0.9 {
			return "close_to_target"
		}
		if coverageData.Current >= coverageData.Target*0.8 {
			return "below_target"
		}
		return "well_below_target"
	}

	// General coverage assessment
	if coverageData.Current >= 90 {
		return "excellent"
	}
	if coverageData.Current >= 80 {
		return "good"
	}
	if coverageData.Current >= 70 {
		return "acceptable"
	}
	if coverageData.Current >= 50 {
		return "poor"
	}
	return "critical"
}

// isValidWebhookURL validates a webhook URL
func isValidWebhookURL(url string) bool {
	if len(url) < 8 {
		return false
	}

	return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")
}

// WebhookChannelBuilder provides a builder pattern for webhook configuration
type WebhookChannelBuilder struct {
	config *types.WebhookConfig
}

// NewWebhookChannelBuilder creates a new webhook channel builder
func NewWebhookChannelBuilder() *WebhookChannelBuilder {
	return &WebhookChannelBuilder{
		config: &types.WebhookConfig{
			Method:      "POST",
			ContentType: "application/json",
			Headers:     make(map[string]string),
		},
	}
}

// URL sets the webhook URL
func (b *WebhookChannelBuilder) URL(url string) *WebhookChannelBuilder {
	b.config.URL = url
	return b
}

// Method sets the HTTP method
func (b *WebhookChannelBuilder) Method(method string) *WebhookChannelBuilder {
	b.config.Method = strings.ToUpper(method)
	return b
}

// ContentType sets the content type
func (b *WebhookChannelBuilder) ContentType(contentType string) *WebhookChannelBuilder {
	b.config.ContentType = contentType
	return b
}

// Header adds a custom header
func (b *WebhookChannelBuilder) Header(key, value string) *WebhookChannelBuilder {
	b.config.Headers[key] = value
	return b
}

// AuthToken sets the authentication token
func (b *WebhookChannelBuilder) AuthToken(token string) *WebhookChannelBuilder {
	b.config.AuthToken = token
	return b
}

// CustomTemplate sets a custom payload template (not implemented in current config)
func (b *WebhookChannelBuilder) CustomTemplate(_ string) *WebhookChannelBuilder {
	// CustomTemplate field not available in current WebhookConfig
	return b
}

// Build creates the webhook channel
func (b *WebhookChannelBuilder) Build() *WebhookChannel {
	return NewWebhookChannel(b.config)
}

// Common webhook templates for popular services

// GetSlackWebhookTemplate returns a Slack-compatible webhook template
func GetSlackWebhookTemplate() string {
	return `{
	"text": "{{.notification.Subject}}",
	"attachments": [
		{
			"color": "{{if eq .severity \"critical\"}}danger{{else if eq .severity \"warning\"}}warning{{else}}good{{end}}",
			"title": "{{.notification.Subject}}",
			"text": "{{.notification.Message}}",
			"fields": [
				{{if .repository}}{
					"title": "Repository",
					"value": "{{.repository}}",
					"short": true
				},{{end}}
				{{if .branch}}{
					"title": "Branch", 
					"value": "{{.branch}}",
					"short": true
				},{{end}}
				{{if .coverage}}{
					"title": "Coverage",
					"value": "{{printf \"%.1f%%\" .coverage.current}}{{if .coverage.change}} ({{if gt .coverage.change 0}}+{{end}}{{printf \"%.1f%%\" .coverage.change}}){{end}}",
					"short": true
				}{{end}}
			],
			"footer": "GoFortress Coverage",
			"ts": {{.timestamp.Unix}}
		}
	]
}`
}

// GetDiscordWebhookTemplate returns a Discord-compatible webhook template
func GetDiscordWebhookTemplate() string {
	return `{
	"embeds": [
		{
			"title": "{{.notification.Subject}}",
			"description": "{{.notification.Message}}",
			"color": {{if eq .severity "critical"}}15158332{{else if eq .severity "warning"}}15105570{{else}}3447003{{end}},
			"fields": [
				{{if .repository}}{
					"name": "📁 Repository",
					"value": "[{{.repository}}](https://github.com/{{.repository}})",
					"inline": true
				},{{end}}
				{{if .branch}}{
					"name": "🌿 Branch",
					"value": "{{.branch}}",
					"inline": true
				},{{end}}
				{{if .coverage}}{
					"name": "📊 Coverage",
					"value": "{{printf \"%.1f%%\" .coverage.current}}{{if .coverage.change}} ({{if gt .coverage.change 0}}+{{end}}{{printf \"%.1f%%\" .coverage.change}}){{end}}",
					"inline": true
				}{{end}}
			],
			"footer": {
				"text": "GoFortress Coverage"
			},
			"timestamp": "{{.timestamp.Format \"2006-01-02T15:04:05Z07:00\"}}"
		}
	]
}`
}

// GetTeamsWebhookTemplate returns a Microsoft Teams-compatible webhook template
func GetTeamsWebhookTemplate() string {
	return `{
	"@type": "MessageCard",
	"@context": "http://schema.org/extensions",
	"themeColor": "{{if eq .severity \"critical\"}}d73027{{else if eq .severity \"warning\"}}fc8d59{{else}}4575b4{{end}}",
	"summary": "{{.notification.Subject}}",
	"title": "{{.notification.Subject}}",
	"text": "{{.notification.Message}}",
	"sections": [
		{
			"facts": [
				{{if .repository}}{
					"name": "Repository",
					"value": "{{.repository}}"
				},{{end}}
				{{if .branch}}{
					"name": "Branch",
					"value": "{{.branch}}"
				},{{end}}
				{{if .coverage}}{
					"name": "Coverage",
					"value": "{{printf \"%.1f%%\" .coverage.current}}{{if .coverage.change}} ({{if gt .coverage.change 0}}+{{end}}{{printf \"%.1f%%\" .coverage.change}}){{end}}"
				},{{end}}
				{
					"name": "Severity",
					"value": "{{.severity}}"
				}
			]
		}
	]
}`
}
