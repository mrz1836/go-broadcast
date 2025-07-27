// Package notify provides multi-channel notification capabilities for coverage events
package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// NotificationEngine manages multi-channel notifications for coverage events
type NotificationEngine struct {
	config   *NotificationConfig
	channels map[ChannelType]NotificationChannel
	events   *EventProcessor
}

// NotificationConfig holds configuration for the notification system
type NotificationConfig struct {
	// Global settings
	Enabled              bool              `json:"enabled"`
	DefaultChannels      []ChannelType     `json:"default_channels"`
	RateLimitPerHour     int               `json:"rate_limit_per_hour"`
	RetryAttempts        int               `json:"retry_attempts"`
	RetryDelay           time.Duration     `json:"retry_delay"`
	
	// Event filtering
	EnabledEvents        []EventType       `json:"enabled_events"`
	MinimumSeverity      SeverityLevel     `json:"minimum_severity"`
	CoverageThresholds   CoverageThresholds `json:"coverage_thresholds"`
	
	// Channel configurations
	SlackConfig          *SlackConfig      `json:"slack_config,omitempty"`
	WebhookConfig        *WebhookConfig    `json:"webhook_config,omitempty"`
	EmailConfig          *EmailConfig      `json:"email_config,omitempty"`
	TeamsConfig          *TeamsConfig      `json:"teams_config,omitempty"`
	DiscordConfig        *DiscordConfig    `json:"discord_config,omitempty"`
	
	// Formatting options
	TemplateConfig       *TemplateConfig   `json:"template_config"`
	MessageFormat        MessageFormat     `json:"message_format"`
	IncludeCharts        bool              `json:"include_charts"`
	IncludeBadges        bool              `json:"include_badges"`
	
	// Advanced features
	DigestEnabled        bool              `json:"digest_enabled"`
	DigestInterval       time.Duration     `json:"digest_interval"`
	QuietHours           *QuietHours       `json:"quiet_hours,omitempty"`
	EscalationConfig     *EscalationConfig `json:"escalation_config,omitempty"`
}

// NotificationChannel interface for different notification channels
type NotificationChannel interface {
	Send(ctx context.Context, notification *Notification) (*DeliveryResult, error)
	ValidateConfig() error
	GetChannelType() ChannelType
	SupportsRichContent() bool
	GetRateLimit() *RateLimit
}

// Notification represents a notification to be sent
type Notification struct {
	// Metadata
	ID                   string            `json:"id"`
	Timestamp            time.Time         `json:"timestamp"`
	EventType            EventType         `json:"event_type"`
	Severity             SeverityLevel     `json:"severity"`
	
	// Content
	Subject              string            `json:"subject"`
	Message              string            `json:"message"`
	RichContent          *RichContent      `json:"rich_content,omitempty"`
	
	// Context
	Repository           string            `json:"repository"`
	Branch               string            `json:"branch"`
	CommitSHA            string            `json:"commit_sha,omitempty"`
	PRNumber             int               `json:"pr_number,omitempty"`
	Author               string            `json:"author,omitempty"`
	
	// Coverage data
	CoverageData         *CoverageData     `json:"coverage_data,omitempty"`
	TrendData            *TrendData        `json:"trend_data,omitempty"`
	
	// Delivery options
	Channels             []ChannelType     `json:"channels"`
	Priority             Priority          `json:"priority"`
	Urgency              Urgency           `json:"urgency"`
	
	// Links and actions
	Actions              []NotificationAction `json:"actions,omitempty"`
	Links                []NotificationLink   `json:"links,omitempty"`
}

// RichContent contains rich formatting options for notifications
type RichContent struct {
	Markdown             string            `json:"markdown,omitempty"`
	HTML                 string            `json:"html,omitempty"`
	Attachments          []Attachment      `json:"attachments,omitempty"`
	Embeds               []Embed           `json:"embeds,omitempty"`
	Charts               []Chart           `json:"charts,omitempty"`
}

// Supporting data structures

type ChannelType string
const (
	ChannelSlack         ChannelType = "slack"
	ChannelWebhook       ChannelType = "webhook"
	ChannelEmail         ChannelType = "email"
	ChannelTeams         ChannelType = "teams"
	ChannelDiscord       ChannelType = "discord"
)

type EventType string
const (
	EventCoverageThreshold   EventType = "coverage_threshold"
	EventCoverageRegression  EventType = "coverage_regression"
	EventCoverageImprovement EventType = "coverage_improvement"
	EventMilestoneReached    EventType = "milestone_reached"
	EventWeeklySummary       EventType = "weekly_summary"
	EventMonthlySummary      EventType = "monthly_summary"
	EventTrendAlert          EventType = "trend_alert"
	EventPredictionAlert     EventType = "prediction_alert"
	EventQualityAlert        EventType = "quality_alert"
	EventSystemAlert         EventType = "system_alert"
)

type SeverityLevel string
const (
	SeverityInfo         SeverityLevel = "info"
	SeverityWarning      SeverityLevel = "warning"
	SeverityCritical     SeverityLevel = "critical"
	SeverityEmergency    SeverityLevel = "emergency"
)

type Priority string
const (
	PriorityLow          Priority = "low"
	PriorityNormal       Priority = "normal"
	PriorityHigh         Priority = "high"
	PriorityUrgent       Priority = "urgent"
)

type Urgency string
const (
	UrgencyNone          Urgency = "none"
	UrgencyMinor         Urgency = "minor"
	UrgencyMajor         Urgency = "major"
	UrgencyCritical      Urgency = "critical"
)

type MessageFormat string
const (
	FormatPlain          MessageFormat = "plain"
	FormatMarkdown       MessageFormat = "markdown"
	FormatHTML           MessageFormat = "html"
	FormatRich           MessageFormat = "rich"
)

// Configuration structures

type CoverageThresholds struct {
	CriticalBelow        float64           `json:"critical_below"`
	WarningBelow         float64           `json:"warning_below"`
	GoodAbove            float64           `json:"good_above"`
	ExcellentAbove       float64           `json:"excellent_above"`
	RegressionThreshold  float64           `json:"regression_threshold"`
}

type SlackConfig struct {
	WebhookURL           string            `json:"webhook_url"`
	Channel              string            `json:"channel"`
	Username             string            `json:"username"`
	IconEmoji            string            `json:"icon_emoji"`
	IconURL              string            `json:"icon_url"`
	LinkNames            bool              `json:"link_names"`
}

type WebhookConfig struct {
	URL                  string            `json:"url"`
	Method               string            `json:"method"`
	Headers              map[string]string `json:"headers"`
	AuthToken            string            `json:"auth_token"`
	ContentType          string            `json:"content_type"`
	CustomTemplate       string            `json:"custom_template"`
}

type EmailConfig struct {
	SMTPHost             string            `json:"smtp_host"`
	SMTPPort             int               `json:"smtp_port"`
	Username             string            `json:"username"`
	Password             string            `json:"password"`
	FromAddress          string            `json:"from_address"`
	ToAddresses          []string          `json:"to_addresses"`
	UseHTML              bool              `json:"use_html"`
	TLSEnabled           bool              `json:"tls_enabled"`
}

type TeamsConfig struct {
	WebhookURL           string            `json:"webhook_url"`
	ThemeColor           string            `json:"theme_color"`
	Title                string            `json:"title"`
}

type DiscordConfig struct {
	WebhookURL           string            `json:"webhook_url"`
	Username             string            `json:"username"`
	AvatarURL            string            `json:"avatar_url"`
	EmbedColor           int               `json:"embed_color"`
}

type TemplateConfig struct {
	DefaultTemplate      string            `json:"default_template"`
	EventTemplates       map[EventType]string `json:"event_templates"`
	IncludeFooter        bool              `json:"include_footer"`
	IncludeTimestamp     bool              `json:"include_timestamp"`
	DateFormat           string            `json:"date_format"`
}

type QuietHours struct {
	Enabled              bool              `json:"enabled"`
	StartHour            int               `json:"start_hour"`
	EndHour              int               `json:"end_hour"`
	Timezone             string            `json:"timezone"`
	ExcludedSeverities   []SeverityLevel   `json:"excluded_severities"`
}

type EscalationConfig struct {
	Enabled              bool              `json:"enabled"`
	EscalationDelay      time.Duration     `json:"escalation_delay"`
	EscalationChannels   []ChannelType     `json:"escalation_channels"`
	MaxEscalations       int               `json:"max_escalations"`
}

// Supporting structures

type CoverageData struct {
	Current              float64           `json:"current"`
	Previous             float64           `json:"previous"`
	Change               float64           `json:"change"`
	Target               float64           `json:"target"`
	Threshold            float64           `json:"threshold"`
}

type TrendData struct {
	Direction            string            `json:"direction"`
	Magnitude            string            `json:"magnitude"`
	Confidence           float64           `json:"confidence"`
	Prediction           float64           `json:"prediction"`
}

type NotificationAction struct {
	Text                 string            `json:"text"`
	URL                  string            `json:"url"`
	Type                 string            `json:"type"`
	Style                string            `json:"style"`
}

type NotificationLink struct {
	Text                 string            `json:"text"`
	URL                  string            `json:"url"`
	Type                 string            `json:"type"`
}

type Attachment struct {
	Filename             string            `json:"filename"`
	Content              []byte            `json:"content"`
	ContentType          string            `json:"content_type"`
}

type Embed struct {
	Title                string            `json:"title"`
	Description          string            `json:"description"`
	Color                string            `json:"color"`
	Fields               []EmbedField      `json:"fields"`
	Thumbnail            *EmbedImage       `json:"thumbnail,omitempty"`
	Image                *EmbedImage       `json:"image,omitempty"`
}

type EmbedField struct {
	Name                 string            `json:"name"`
	Value                string            `json:"value"`
	Inline               bool              `json:"inline"`
}

type EmbedImage struct {
	URL                  string            `json:"url"`
	Width                int               `json:"width,omitempty"`
	Height               int               `json:"height,omitempty"`
}

type Chart struct {
	Type                 string            `json:"type"`
	Data                 interface{}       `json:"data"`
	URL                  string            `json:"url"`
}

type DeliveryResult struct {
	Success              bool              `json:"success"`
	Channel              ChannelType       `json:"channel"`
	MessageID            string            `json:"message_id,omitempty"`
	Timestamp            time.Time         `json:"timestamp"`
	Error                error             `json:"error,omitempty"`
	RetryCount           int               `json:"retry_count"`
	DeliveryTime         time.Duration     `json:"delivery_time"`
}

type RateLimit struct {
	RequestsPerMinute    int               `json:"requests_per_minute"`
	RequestsPerHour      int               `json:"requests_per_hour"`
	RequestsPerDay       int               `json:"requests_per_day"`
	BurstSize            int               `json:"burst_size"`
}

// NewNotificationEngine creates a new notification engine with the provided configuration
func NewNotificationEngine(config *NotificationConfig) *NotificationEngine {
	if config == nil {
		config = &NotificationConfig{
			Enabled:             true,
			DefaultChannels:     []ChannelType{ChannelSlack},
			RateLimitPerHour:    100,
			RetryAttempts:       3,
			RetryDelay:          time.Minute,
			EnabledEvents:       []EventType{EventCoverageThreshold, EventCoverageRegression, EventMilestoneReached},
			MinimumSeverity:     SeverityInfo,
			CoverageThresholds: CoverageThresholds{
				CriticalBelow:       50.0,
				WarningBelow:        70.0,
				GoodAbove:           80.0,
				ExcellentAbove:      90.0,
				RegressionThreshold: 5.0,
			},
			TemplateConfig: &TemplateConfig{
				DefaultTemplate:   "standard",
				IncludeFooter:     true,
				IncludeTimestamp:  true,
				DateFormat:        "2006-01-02 15:04:05 MST",
			},
			MessageFormat:       FormatMarkdown,
			IncludeCharts:       true,
			IncludeBadges:       true,
			DigestEnabled:       false,
			DigestInterval:      24 * time.Hour,
		}
	}
	
	engine := &NotificationEngine{
		config:   config,
		channels: make(map[ChannelType]NotificationChannel),
		events:   NewEventProcessor(),
	}
	
	// Initialize enabled channels
	engine.initializeChannels()
	
	return engine
}

// SendNotification sends a notification through the specified channels
func (ne *NotificationEngine) SendNotification(ctx context.Context, notification *Notification) ([]*DeliveryResult, error) {
	if !ne.config.Enabled {
		return nil, fmt.Errorf("notification engine is disabled")
	}
	
	// Validate notification
	if err := ne.validateNotification(notification); err != nil {
		return nil, fmt.Errorf("invalid notification: %w", err)
	}
	
	// Check if event should be processed
	if !ne.shouldProcessEvent(notification) {
		return nil, fmt.Errorf("event %s filtered out", notification.EventType)
	}
	
	// Check quiet hours
	if ne.isQuietTime(notification) {
		return nil, fmt.Errorf("notification suppressed due to quiet hours")
	}
	
	// Process notification content
	if err := ne.processNotificationContent(notification); err != nil {
		return nil, fmt.Errorf("failed to process notification content: %w", err)
	}
	
	// Send to channels
	channels := notification.Channels
	if len(channels) == 0 {
		channels = ne.config.DefaultChannels
	}
	
	var results []*DeliveryResult
	for _, channelType := range channels {
		channel, exists := ne.channels[channelType]
		if !exists {
			results = append(results, &DeliveryResult{
				Success:   false,
				Channel:   channelType,
				Timestamp: time.Now(),
				Error:     fmt.Errorf("channel %s not configured", channelType),
			})
			continue
		}
		
		// Send with retry logic
		result := ne.sendWithRetry(ctx, channel, notification)
		results = append(results, result)
	}
	
	return results, nil
}

// SendCoverageEvent sends a coverage-related event notification
func (ne *NotificationEngine) SendCoverageEvent(ctx context.Context, eventType EventType, coverageData *CoverageData, metadata map[string]interface{}) error {
	notification := &Notification{
		ID:           ne.generateNotificationID(),
		Timestamp:    time.Now(),
		EventType:    eventType,
		Severity:     ne.determineSeverity(eventType, coverageData),
		CoverageData: coverageData,
		Priority:     ne.determinePriority(eventType, coverageData),
		Urgency:      ne.determineUrgency(eventType, coverageData),
	}
	
	// Extract metadata
	if metadata != nil {
		if repo, ok := metadata["repository"].(string); ok {
			notification.Repository = repo
		}
		if branch, ok := metadata["branch"].(string); ok {
			notification.Branch = branch
		}
		if sha, ok := metadata["commit_sha"].(string); ok {
			notification.CommitSHA = sha
		}
		if pr, ok := metadata["pr_number"].(int); ok {
			notification.PRNumber = pr
		}
		if author, ok := metadata["author"].(string); ok {
			notification.Author = author
		}
	}
	
	// Generate content based on event type
	ne.generateEventContent(notification)
	
	_, err := ne.SendNotification(ctx, notification)
	return err
}

// SendDigest sends a digest of recent events
func (ne *NotificationEngine) SendDigest(ctx context.Context, digestType string, period time.Duration) error {
	if !ne.config.DigestEnabled {
		return fmt.Errorf("digest notifications are disabled")
	}
	
	// This would collect events from the specified period
	// For now, return a placeholder implementation
	notification := &Notification{
		ID:        ne.generateNotificationID(),
		Timestamp: time.Now(),
		EventType: EventWeeklySummary,
		Severity:  SeverityInfo,
		Subject:   fmt.Sprintf("%s Coverage Digest", strings.Title(digestType)),
		Message:   "Coverage digest implementation pending",
		Priority:  PriorityNormal,
		Urgency:   UrgencyNone,
	}
	
	_, err := ne.SendNotification(ctx, notification)
	return err
}

// Helper methods

func (ne *NotificationEngine) initializeChannels() {
	// Initialize Slack channel
	if ne.config.SlackConfig != nil && ne.config.SlackConfig.WebhookURL != "" {
		slackChannel := NewSlackChannel(ne.config.SlackConfig)
		ne.channels[ChannelSlack] = slackChannel
	}
	
	// Initialize Webhook channel
	if ne.config.WebhookConfig != nil && ne.config.WebhookConfig.URL != "" {
		webhookChannel := NewWebhookChannel(ne.config.WebhookConfig)
		ne.channels[ChannelWebhook] = webhookChannel
	}
	
	// Initialize Email channel
	if ne.config.EmailConfig != nil && ne.config.EmailConfig.SMTPHost != "" {
		emailChannel := NewEmailChannel(ne.config.EmailConfig)
		ne.channels[ChannelEmail] = emailChannel
	}
	
	// Initialize Teams channel
	if ne.config.TeamsConfig != nil && ne.config.TeamsConfig.WebhookURL != "" {
		teamsChannel := NewTeamsChannel(ne.config.TeamsConfig)
		ne.channels[ChannelTeams] = teamsChannel
	}
	
	// Initialize Discord channel
	if ne.config.DiscordConfig != nil && ne.config.DiscordConfig.WebhookURL != "" {
		discordChannel := NewDiscordChannel(ne.config.DiscordConfig)
		ne.channels[ChannelDiscord] = discordChannel
	}
}

func (ne *NotificationEngine) validateNotification(notification *Notification) error {
	if notification == nil {
		return fmt.Errorf("notification is nil")
	}
	
	if notification.EventType == "" {
		return fmt.Errorf("event type is required")
	}
	
	if notification.Subject == "" && notification.Message == "" {
		return fmt.Errorf("either subject or message is required")
	}
	
	return nil
}

func (ne *NotificationEngine) shouldProcessEvent(notification *Notification) bool {
	// Check if event type is enabled
	eventEnabled := false
	for _, enabledEvent := range ne.config.EnabledEvents {
		if enabledEvent == notification.EventType {
			eventEnabled = true
			break
		}
	}
	
	if !eventEnabled {
		return false
	}
	
	// Check severity threshold
	return ne.compareSeverity(notification.Severity, ne.config.MinimumSeverity) >= 0
}

func (ne *NotificationEngine) isQuietTime(notification *Notification) bool {
	if ne.config.QuietHours == nil || !ne.config.QuietHours.Enabled {
		return false
	}
	
	// Check if severity is excluded from quiet hours
	for _, excludedSeverity := range ne.config.QuietHours.ExcludedSeverities {
		if notification.Severity == excludedSeverity {
			return false
		}
	}
	
	// Check current time against quiet hours
	now := time.Now()
	hour := now.Hour()
	
	start := ne.config.QuietHours.StartHour
	end := ne.config.QuietHours.EndHour
	
	if start < end {
		return hour >= start && hour < end
	} else {
		return hour >= start || hour < end
	}
}

func (ne *NotificationEngine) processNotificationContent(notification *Notification) error {
	// Apply templates if configured
	if ne.config.TemplateConfig != nil {
		if err := ne.applyTemplate(notification); err != nil {
			return fmt.Errorf("failed to apply template: %w", err)
		}
	}
	
	// Generate rich content if supported
	if ne.config.MessageFormat == FormatRich || ne.config.MessageFormat == FormatMarkdown {
		ne.generateRichContent(notification)
	}
	
	return nil
}

func (ne *NotificationEngine) sendWithRetry(ctx context.Context, channel NotificationChannel, notification *Notification) *DeliveryResult {
	var lastError error
	startTime := time.Now()
	
	for attempt := 0; attempt <= ne.config.RetryAttempts; attempt++ {
		if attempt > 0 {
			// Wait before retry
			select {
			case <-ctx.Done():
				return &DeliveryResult{
					Success:      false,
					Channel:      channel.GetChannelType(),
					Timestamp:    time.Now(),
					Error:        ctx.Err(),
					RetryCount:   attempt,
					DeliveryTime: time.Since(startTime),
				}
			case <-time.After(ne.config.RetryDelay):
			}
		}
		
		result, err := channel.Send(ctx, notification)
		if err == nil && result.Success {
			result.RetryCount = attempt
			result.DeliveryTime = time.Since(startTime)
			return result
		}
		
		lastError = err
		if result != nil && result.Error != nil {
			lastError = result.Error
		}
	}
	
	return &DeliveryResult{
		Success:      false,
		Channel:      channel.GetChannelType(),
		Timestamp:    time.Now(),
		Error:        lastError,
		RetryCount:   ne.config.RetryAttempts,
		DeliveryTime: time.Since(startTime),
	}
}

func (ne *NotificationEngine) generateNotificationID() string {
	return fmt.Sprintf("notif_%d", time.Now().UnixNano())
}

func (ne *NotificationEngine) determineSeverity(eventType EventType, coverageData *CoverageData) SeverityLevel {
	if coverageData == nil {
		return SeverityInfo
	}
	
	switch eventType {
	case EventCoverageRegression:
		if coverageData.Change < -ne.config.CoverageThresholds.RegressionThreshold {
			return SeverityCritical
		}
		return SeverityWarning
		
	case EventCoverageThreshold:
		if coverageData.Current < ne.config.CoverageThresholds.CriticalBelow {
			return SeverityCritical
		} else if coverageData.Current < ne.config.CoverageThresholds.WarningBelow {
			return SeverityWarning
		}
		return SeverityInfo
		
	case EventMilestoneReached:
		return SeverityInfo
		
	default:
		return SeverityInfo
	}
}

func (ne *NotificationEngine) determinePriority(eventType EventType, coverageData *CoverageData) Priority {
	switch eventType {
	case EventCoverageRegression:
		if coverageData != nil && coverageData.Change < -10.0 {
			return PriorityUrgent
		}
		return PriorityHigh
		
	case EventCoverageThreshold:
		if coverageData != nil && coverageData.Current < ne.config.CoverageThresholds.CriticalBelow {
			return PriorityHigh
		}
		return PriorityNormal
		
	default:
		return PriorityNormal
	}
}

func (ne *NotificationEngine) determineUrgency(eventType EventType, coverageData *CoverageData) Urgency {
	switch eventType {
	case EventCoverageRegression:
		if coverageData != nil && coverageData.Change < -15.0 {
			return UrgencyCritical
		} else if coverageData != nil && coverageData.Change < -10.0 {
			return UrgencyMajor
		}
		return UrgencyMinor
		
	default:
		return UrgencyNone
	}
}

func (ne *NotificationEngine) generateEventContent(notification *Notification) {
	switch notification.EventType {
	case EventCoverageThreshold:
		ne.generateCoverageThresholdContent(notification)
	case EventCoverageRegression:
		ne.generateCoverageRegressionContent(notification)
	case EventCoverageImprovement:
		ne.generateCoverageImprovementContent(notification)
	case EventMilestoneReached:
		ne.generateMilestoneContent(notification)
	default:
		ne.generateGenericContent(notification)
	}
}

func (ne *NotificationEngine) generateCoverageThresholdContent(notification *Notification) {
	coverage := notification.CoverageData
	if coverage == nil {
		return
	}
	
	if coverage.Current < ne.config.CoverageThresholds.CriticalBelow {
		notification.Subject = fmt.Sprintf("🚨 Critical: Coverage dropped to %.1f%%", coverage.Current)
		notification.Message = fmt.Sprintf("Coverage in %s is critically low at %.1f%% (threshold: %.1f%%). Immediate attention required.", 
			notification.Repository, coverage.Current, ne.config.CoverageThresholds.CriticalBelow)
	} else if coverage.Current < ne.config.CoverageThresholds.WarningBelow {
		notification.Subject = fmt.Sprintf("⚠️ Warning: Coverage at %.1f%%", coverage.Current)
		notification.Message = fmt.Sprintf("Coverage in %s is below warning threshold at %.1f%% (threshold: %.1f%%).", 
			notification.Repository, coverage.Current, ne.config.CoverageThresholds.WarningBelow)
	}
}

func (ne *NotificationEngine) generateCoverageRegressionContent(notification *Notification) {
	coverage := notification.CoverageData
	if coverage == nil {
		return
	}
	
	notification.Subject = fmt.Sprintf("📉 Coverage Regression: %.1f%% → %.1f%%", coverage.Previous, coverage.Current)
	notification.Message = fmt.Sprintf("Coverage in %s has decreased by %.1f%% (from %.1f%% to %.1f%%).", 
		notification.Repository, coverage.Change, coverage.Previous, coverage.Current)
	
	if notification.PRNumber > 0 {
		notification.Message += fmt.Sprintf(" This change was introduced in PR #%d.", notification.PRNumber)
	}
}

func (ne *NotificationEngine) generateCoverageImprovementContent(notification *Notification) {
	coverage := notification.CoverageData
	if coverage == nil {
		return
	}
	
	notification.Subject = fmt.Sprintf("📈 Coverage Improved: %.1f%% → %.1f%%", coverage.Previous, coverage.Current)
	notification.Message = fmt.Sprintf("Great news! Coverage in %s has increased by %.1f%% (from %.1f%% to %.1f%%).", 
		notification.Repository, coverage.Change, coverage.Previous, coverage.Current)
}

func (ne *NotificationEngine) generateMilestoneContent(notification *Notification) {
	coverage := notification.CoverageData
	if coverage == nil {
		return
	}
	
	notification.Subject = fmt.Sprintf("🎉 Milestone: %.0f%% Coverage Reached!", coverage.Current)
	notification.Message = fmt.Sprintf("Congratulations! %s has reached %.0f%% coverage milestone.", 
		notification.Repository, coverage.Current)
}

func (ne *NotificationEngine) generateGenericContent(notification *Notification) {
	notification.Subject = fmt.Sprintf("Coverage Update: %s", notification.Repository)
	notification.Message = fmt.Sprintf("Coverage event %s occurred in %s.", notification.EventType, notification.Repository)
}

func (ne *NotificationEngine) applyTemplate(notification *Notification) error {
	// Template processing would be implemented here
	// For now, just add timestamp if configured
	if ne.config.TemplateConfig.IncludeTimestamp {
		timestamp := notification.Timestamp.Format(ne.config.TemplateConfig.DateFormat)
		notification.Message += fmt.Sprintf("\n\n*%s*", timestamp)
	}
	
	if ne.config.TemplateConfig.IncludeFooter {
		notification.Message += "\n\n---\n*Generated by GoFortress Coverage System*"
	}
	
	return nil
}

func (ne *NotificationEngine) generateRichContent(notification *Notification) {
	if notification.RichContent == nil {
		notification.RichContent = &RichContent{}
	}
	
	// Generate markdown version of the message
	var markdown strings.Builder
	
	// Add emoji based on severity
	emoji := "ℹ️"
	switch notification.Severity {
	case SeverityWarning:
		emoji = "⚠️"
	case SeverityCritical:
		emoji = "🚨"
	case SeverityEmergency:
		emoji = "🔥"
	}
	
	markdown.WriteString(fmt.Sprintf("%s **%s**\n\n", emoji, notification.Subject))
	markdown.WriteString(notification.Message)
	
	// Add coverage data if available
	if notification.CoverageData != nil {
		markdown.WriteString("\n\n### Coverage Details\n")
		markdown.WriteString(fmt.Sprintf("- **Current**: %.1f%%\n", notification.CoverageData.Current))
		if notification.CoverageData.Previous > 0 {
			markdown.WriteString(fmt.Sprintf("- **Previous**: %.1f%%\n", notification.CoverageData.Previous))
			markdown.WriteString(fmt.Sprintf("- **Change**: %+.1f%%\n", notification.CoverageData.Change))
		}
	}
	
	// Add links
	if notification.Repository != "" {
		repoURL := fmt.Sprintf("https://github.com/%s", notification.Repository)
		markdown.WriteString(fmt.Sprintf("\n\n[View Repository](%s)", repoURL))
		
		if notification.PRNumber > 0 {
			prURL := fmt.Sprintf("%s/pull/%d", repoURL, notification.PRNumber)
			markdown.WriteString(fmt.Sprintf(" | [View PR #%d](%s)", notification.PRNumber, prURL))
		}
	}
	
	notification.RichContent.Markdown = markdown.String()
}

func (ne *NotificationEngine) compareSeverity(a, b SeverityLevel) int {
	severityOrder := map[SeverityLevel]int{
		SeverityInfo:      0,
		SeverityWarning:   1,
		SeverityCritical:  2,
		SeverityEmergency: 3,
	}
	
	orderA, okA := severityOrder[a]
	orderB, okB := severityOrder[b]
	
	if !okA || !okB {
		return 0
	}
	
	return orderA - orderB
}

// GetChannelStatus returns the status of all configured channels
func (ne *NotificationEngine) GetChannelStatus() map[ChannelType]bool {
	status := make(map[ChannelType]bool)
	
	for channelType, channel := range ne.channels {
		err := channel.ValidateConfig()
		status[channelType] = err == nil
	}
	
	return status
}

// AddChannel adds a new notification channel
func (ne *NotificationEngine) AddChannel(channel NotificationChannel) error {
	if err := channel.ValidateConfig(); err != nil {
		return fmt.Errorf("invalid channel configuration: %w", err)
	}
	
	ne.channels[channel.GetChannelType()] = channel
	return nil
}

// RemoveChannel removes a notification channel
func (ne *NotificationEngine) RemoveChannel(channelType ChannelType) {
	delete(ne.channels, channelType)
}