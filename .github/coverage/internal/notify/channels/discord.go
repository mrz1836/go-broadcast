// Package channels provides Discord notification channel implementation
package channels

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/mrz1836/go-broadcast/coverage/internal/types"
)

// DiscordChannel implements Discord webhook notifications
type DiscordChannel struct {
	config    *types.DiscordConfig
	rateLimit *types.RateLimit
	client    *http.Client
}

// DiscordMessage represents a Discord message payload
type DiscordMessage struct {
	Content   string         `json:"content,omitempty"`
	Username  string         `json:"username,omitempty"`
	AvatarURL string         `json:"avatar_url,omitempty"`
	TTS       bool           `json:"tts,omitempty"`
	Embeds    []DiscordEmbed `json:"embeds,omitempty"`
}

// DiscordEmbed represents a Discord embed
type DiscordEmbed struct {
	Title       string              `json:"title,omitempty"`
	Type        string              `json:"type,omitempty"`
	Description string              `json:"description,omitempty"`
	URL         string              `json:"url,omitempty"`
	Timestamp   string              `json:"timestamp,omitempty"`
	Color       int                 `json:"color,omitempty"`
	Footer      *DiscordEmbedFooter `json:"footer,omitempty"`
	Image       *DiscordEmbedImage  `json:"image,omitempty"`
	Thumbnail   *DiscordEmbedImage  `json:"thumbnail,omitempty"`
	Author      *DiscordEmbedAuthor `json:"author,omitempty"`
	Fields      []DiscordEmbedField `json:"fields,omitempty"`
}

// DiscordEmbedFooter represents a Discord embed footer
type DiscordEmbedFooter struct {
	Text    string `json:"text"`
	IconURL string `json:"icon_url,omitempty"`
}

// DiscordEmbedImage represents a Discord embed image
type DiscordEmbedImage struct {
	URL    string `json:"url"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
}

// DiscordEmbedAuthor represents a Discord embed author
type DiscordEmbedAuthor struct {
	Name    string `json:"name"`
	URL     string `json:"url,omitempty"`
	IconURL string `json:"icon_url,omitempty"`
}

// DiscordEmbedField represents a Discord embed field
type DiscordEmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

// NewDiscordChannel creates a new Discord notification channel
func NewDiscordChannel(config *types.DiscordConfig) *DiscordChannel {
	return &DiscordChannel{
		config: config,
		rateLimit: &types.RateLimit{
			RequestsPerMinute: 30, // Discord rate limit (conservative)
			RequestsPerHour:   1800,
			RequestsPerDay:    43200,
			BurstSize:         5,
		},
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Send implements the NotificationChannel interface for Discord
func (d *DiscordChannel) Send(ctx context.Context, notification *types.Notification) (*types.DeliveryResult, error) {
	startTime := time.Now()
	result := &types.DeliveryResult{
		Channel:   types.ChannelDiscord,
		Timestamp: startTime,
	}

	// Build Discord message
	message := d.buildDiscordMessage(notification)

	// Marshal message to JSON
	payload, err := json.Marshal(message)
	if err != nil {
		result.Error = fmt.Errorf("failed to marshal Discord message: %w", err)
		return result, result.Error
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", d.config.WebhookURL, bytes.NewBuffer(payload))
	if err != nil {
		result.Error = fmt.Errorf("failed to create request: %w", err)
		return result, result.Error
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := d.client.Do(req)
	if err != nil {
		result.Error = fmt.Errorf("failed to send request: %w", err)
		return result, result.Error
	}
	defer func() { _ = resp.Body.Close() }()

	result.DeliveryTime = time.Since(startTime)

	// Check response
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		result.Success = true
		result.MessageID = fmt.Sprintf("discord_%d", time.Now().Unix())
	} else {
		result.Error = fmt.Errorf("Discord API returned status %d", resp.StatusCode)
	}

	return result, nil
}

// ValidateConfig validates the Discord channel configuration
func (d *DiscordChannel) ValidateConfig() error {
	if d.config == nil {
		return fmt.Errorf("Discord config is nil")
	}

	if d.config.WebhookURL == "" {
		return fmt.Errorf("Discord webhook URL is required")
	}

	// Validate webhook URL format
	if !isValidDiscordWebhookURL(d.config.WebhookURL) {
		return fmt.Errorf("invalid Discord webhook URL format")
	}

	return nil
}

// GetChannelType returns the channel type
func (d *DiscordChannel) GetChannelType() types.ChannelType {
	return types.ChannelDiscord
}

// SupportsRichContent returns whether the channel supports rich content
func (d *DiscordChannel) SupportsRichContent() bool {
	return true
}

// GetRateLimit returns the rate limit configuration
func (d *DiscordChannel) GetRateLimit() *types.RateLimit {
	return d.rateLimit
}

// buildDiscordMessage builds a Discord message from a notification
func (d *DiscordChannel) buildDiscordMessage(notification *types.Notification) *DiscordMessage {
	message := &DiscordMessage{
		Username:  d.config.Username,
		AvatarURL: d.config.AvatarURL,
		TTS:       false,
	}

	// Build embed
	embed := d.buildDiscordEmbed(notification)
	message.Embeds = []DiscordEmbed{*embed}

	// Add content if no rich content
	if notification.RichContent == nil || notification.RichContent.Markdown == "" {
		message.Content = fmt.Sprintf("**%s**\n%s", notification.Subject, notification.Message)
	}

	return message
}

// buildDiscordEmbed builds a Discord embed from notification data
func (d *DiscordChannel) buildDiscordEmbed(notification *types.Notification) *DiscordEmbed {
	embed := &DiscordEmbed{
		Title:       notification.Subject,
		Description: notification.Message,
		Color:       d.getEmbedColor(notification.Severity),
		Timestamp:   notification.Timestamp.Format(time.RFC3339),
		Type:        "rich",
		Fields:      make([]DiscordEmbedField, 0),
	}

	// Set custom embed color if configured
	if d.config.EmbedColor > 0 {
		embed.Color = d.config.EmbedColor
	}

	// Add footer
	embed.Footer = &DiscordEmbedFooter{
		Text:    "GoFortress Coverage",
		IconURL: "https://github.com/favicon.ico",
	}

	// Add author information
	if notification.Author != "" {
		embed.Author = &DiscordEmbedAuthor{
			Name:    notification.Author,
			IconURL: fmt.Sprintf("https://github.com/%s.png", notification.Author),
			URL:     fmt.Sprintf("https://github.com/%s", notification.Author),
		}
	}

	// Add repository field
	if notification.Repository != "" {
		repoURL := fmt.Sprintf("https://github.com/%s", notification.Repository)
		embed.Fields = append(embed.Fields, DiscordEmbedField{
			Name:   "📁 Repository",
			Value:  fmt.Sprintf("[%s](%s)", notification.Repository, repoURL),
			Inline: true,
		})
	}

	// Add branch field
	if notification.Branch != "" {
		branchURL := fmt.Sprintf("https://github.com/%s/tree/%s", notification.Repository, notification.Branch)
		embed.Fields = append(embed.Fields, DiscordEmbedField{
			Name:   "🌿 Branch",
			Value:  fmt.Sprintf("[%s](%s)", notification.Branch, branchURL),
			Inline: true,
		})
	}

	// Add PR field
	if notification.PRNumber > 0 {
		prURL := fmt.Sprintf("https://github.com/%s/pull/%d", notification.Repository, notification.PRNumber)
		embed.Fields = append(embed.Fields, DiscordEmbedField{
			Name:   "🔀 Pull Request",
			Value:  fmt.Sprintf("[#%d](%s)", notification.PRNumber, prURL),
			Inline: true,
		})
	}

	// Add coverage field
	if notification.CoverageData != nil {
		coverageIcon := d.getCoverageIcon(notification.CoverageData.Change)
		coverageText := fmt.Sprintf("%.1f%%", notification.CoverageData.Current)

		if notification.CoverageData.Previous > 0 {
			change := notification.CoverageData.Change
			if change > 0 {
				coverageText += fmt.Sprintf(" (+%.1f%%)", change)
			} else if change < 0 {
				coverageText += fmt.Sprintf(" (%.1f%%)", change)
			}
		}

		embed.Fields = append(embed.Fields, DiscordEmbedField{
			Name:   fmt.Sprintf("%s Coverage", coverageIcon),
			Value:  coverageText,
			Inline: true,
		})

		// Add target progress if available
		if notification.CoverageData.Target > 0 {
			progress := (notification.CoverageData.Current / notification.CoverageData.Target) * 100
			progressBar := d.generateProgressBar(progress)
			embed.Fields = append(embed.Fields, DiscordEmbedField{
				Name:   "🎯 Target Progress",
				Value:  fmt.Sprintf("%s %.1f%% of %.1f%%", progressBar, notification.CoverageData.Current, notification.CoverageData.Target),
				Inline: false,
			})
		}
	}

	// Add severity field
	severityIcon := d.getSeverityIcon(notification.Severity)
	embed.Fields = append(embed.Fields, DiscordEmbedField{
		Name:   fmt.Sprintf("%s Severity", severityIcon),
		Value:  string(notification.Severity),
		Inline: true,
	})

	// Add priority field
	priorityIcon := d.getPriorityIcon(notification.Priority)
	embed.Fields = append(embed.Fields, DiscordEmbedField{
		Name:   fmt.Sprintf("%s Priority", priorityIcon),
		Value:  string(notification.Priority),
		Inline: true,
	})

	// Add trend information
	if notification.TrendData != nil {
		trendIcon := d.getTrendIcon(notification.TrendData.Direction)
		embed.Fields = append(embed.Fields, DiscordEmbedField{
			Name:   fmt.Sprintf("%s Trend", trendIcon),
			Value:  fmt.Sprintf("%s (%.0f%% confidence)", notification.TrendData.Direction, notification.TrendData.Confidence*100),
			Inline: true,
		})
	}

	// Add commit information
	if notification.CommitSHA != "" {
		commitURL := fmt.Sprintf("https://github.com/%s/commit/%s", notification.Repository, notification.CommitSHA)
		shortSHA := notification.CommitSHA
		if len(shortSHA) > 8 {
			shortSHA = shortSHA[:8]
		}
		embed.Fields = append(embed.Fields, DiscordEmbedField{
			Name:   "📝 Commit",
			Value:  fmt.Sprintf("[%s](%s)", shortSHA, commitURL),
			Inline: true,
		})
	}

	return embed
}

// getEmbedColor returns the embed color for a severity level
func (d *DiscordChannel) getEmbedColor(severity types.SeverityLevel) int {
	switch severity {
	case types.SeverityInfo:
		return 0x3498db // Blue
	case types.SeverityWarning:
		return 0xf39c12 // Orange
	case types.SeverityCritical:
		return 0xe74c3c // Red
	case types.SeverityEmergency:
		return 0x8b0000 // Dark Red
	default:
		return 0x9b59b6 // Purple
	}
}

// getCoverageIcon returns an icon for coverage change
func (d *DiscordChannel) getCoverageIcon(change float64) string {
	if change > 0 {
		return "📈"
	} else if change < 0 {
		return "📉"
	}
	return "📊"
}

// getSeverityIcon returns an icon for severity level
func (d *DiscordChannel) getSeverityIcon(severity types.SeverityLevel) string {
	switch severity {
	case types.SeverityInfo:
		return "ℹ️"
	case types.SeverityWarning:
		return "⚠️"
	case types.SeverityCritical:
		return "🚨"
	case types.SeverityEmergency:
		return "🔥"
	default:
		return "📢"
	}
}

// getPriorityIcon returns an icon for priority level
func (d *DiscordChannel) getPriorityIcon(priority types.Priority) string {
	switch priority {
	case types.PriorityLow:
		return "🔵"
	case types.PriorityNormal:
		return "🟡"
	case types.PriorityHigh:
		return "🟠"
	case types.PriorityUrgent:
		return "🔴"
	default:
		return "⚪"
	}
}

// getTrendIcon returns an icon for trend direction
func (d *DiscordChannel) getTrendIcon(direction string) string {
	switch direction {
	case "upward":
		return "📈"
	case "downward":
		return "📉"
	case "stable":
		return "➡️"
	default:
		return "📊"
	}
}

// generateProgressBar generates a visual progress bar
func (d *DiscordChannel) generateProgressBar(percentage float64) string {
	if percentage > 100 {
		percentage = 100
	} else if percentage < 0 {
		percentage = 0
	}

	// Use Discord-compatible progress bar emojis
	filled := int(percentage / 10)
	empty := 10 - filled

	bar := ""
	for i := 0; i < filled; i++ {
		bar += "🟩"
	}
	for i := 0; i < empty; i++ {
		bar += "⬜"
	}

	return bar
}

// isValidDiscordWebhookURL validates a Discord webhook URL
func isValidDiscordWebhookURL(url string) bool {
	return len(url) > 20 && strings.Contains(url, "discord.com/api/webhooks/")
}
