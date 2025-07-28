// Package types defines common data structures and configuration types used throughout the coverage system.
// This package provides shared type definitions for notification channels, configuration objects,
// and other data structures that need to be used across multiple packages.
package types

import (
	"context"
	"time"
)

// ChannelType represents different notification channel types
type ChannelType string

const (
	// ChannelSlack represents Slack notification channel
	ChannelSlack ChannelType = "slack"
	// ChannelEmail represents email notification channel
	ChannelEmail ChannelType = "email"
	// ChannelWebhook represents webhook notification channel
	ChannelWebhook ChannelType = "webhook"
	// ChannelTeams represents Microsoft Teams notification channel
	ChannelTeams ChannelType = "teams"
	// ChannelDiscord represents Discord notification channel
	ChannelDiscord ChannelType = "discord"
)

// SeverityLevel represents notification severity levels
type SeverityLevel string

const (
	// SeverityInfo represents informational severity level
	SeverityInfo SeverityLevel = "info"
	// SeverityWarning represents warning severity level
	SeverityWarning SeverityLevel = "warning"
	// SeverityCritical represents critical severity level
	SeverityCritical SeverityLevel = "critical"
	// SeverityEmergency represents emergency severity level
	SeverityEmergency SeverityLevel = "emergency"
)

// Priority represents notification priority levels
type Priority string

const (
	// PriorityLow represents low priority level
	PriorityLow Priority = "low"
	// PriorityNormal represents normal priority level
	PriorityNormal Priority = "normal"
	// PriorityHigh represents high priority level
	PriorityHigh Priority = "high"
	// PriorityUrgent represents urgent priority level
	PriorityUrgent Priority = "urgent"
)

// Notification represents a coverage notification
type Notification struct {
	ID           string                 `json:"id"`
	Subject      string                 `json:"subject"`
	Message      string                 `json:"message"`
	Severity     SeverityLevel          `json:"severity"`
	Priority     Priority               `json:"priority"`
	Timestamp    time.Time              `json:"timestamp"`
	Author       string                 `json:"author,omitempty"`
	Repository   string                 `json:"repository,omitempty"`
	Branch       string                 `json:"branch,omitempty"`
	PRNumber     int                    `json:"pr_number,omitempty"`
	CommitSHA    string                 `json:"commit_sha,omitempty"`
	CoverageData *CoverageData          `json:"coverage_data,omitempty"`
	TrendData    *TrendData             `json:"trend_data,omitempty"`
	RichContent  *RichContent           `json:"rich_content,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// CoverageData represents coverage information in notifications
type CoverageData struct {
	Current  float64 `json:"current"`
	Previous float64 `json:"previous"`
	Change   float64 `json:"change"`
	Target   float64 `json:"target,omitempty"`
}

// TrendData represents trend information
type TrendData struct {
	Direction  string  `json:"direction"`
	Confidence float64 `json:"confidence"`
}

// RichContent represents rich content for notifications
type RichContent struct {
	Markdown string            `json:"markdown,omitempty"`
	HTML     string            `json:"html,omitempty"`
	Fields   map[string]string `json:"fields,omitempty"`
}

// DeliveryResult represents the result of a notification delivery
type DeliveryResult struct {
	Channel      ChannelType   `json:"channel"`
	Success      bool          `json:"success"`
	MessageID    string        `json:"message_id,omitempty"`
	Timestamp    time.Time     `json:"timestamp"`
	DeliveryTime time.Duration `json:"delivery_time"`
	Error        error         `json:"error,omitempty"`
}

// RateLimit represents rate limiting configuration
type RateLimit struct {
	RequestsPerMinute int           `json:"requests_per_minute"`
	RequestsPerHour   int           `json:"requests_per_hour"`
	RequestsPerDay    int           `json:"requests_per_day"`
	BurstSize         int           `json:"burst_size"`
	Window            time.Duration `json:"window"`
}

// NotificationChannel represents a notification delivery channel
type NotificationChannel interface {
	Send(ctx context.Context, notification *Notification) (*DeliveryResult, error)
	ValidateConfig() error
	GetChannelType() ChannelType
	SupportsRichContent() bool
	GetRateLimit() *RateLimit
}
