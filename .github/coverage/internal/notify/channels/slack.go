// Package channels provides specific notification channel implementations
package channels

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/mrz1836/go-broadcast/coverage/internal/types"
)

// Static error definitions
var (
	ErrSlackAPIError        = errors.New("slack API returned error status")
	ErrSlackConfigNil       = errors.New("slack config is nil")
	ErrSlackWebhookRequired = errors.New("slack webhook URL is required")
	ErrSlackWebhookInvalid  = errors.New("invalid Slack webhook URL format")
)

// SlackChannel implements Slack webhook notifications
type SlackChannel struct {
	config    *types.SlackConfig
	rateLimit *types.RateLimit
	client    *http.Client
}

// SlackMessage represents a Slack message payload
type SlackMessage struct {
	Text        string            `json:"text,omitempty"`
	Username    string            `json:"username,omitempty"`
	IconEmoji   string            `json:"icon_emoji,omitempty"`
	IconURL     string            `json:"icon_url,omitempty"`
	Channel     string            `json:"channel,omitempty"`
	LinkNames   bool              `json:"link_names,omitempty"`
	Attachments []SlackAttachment `json:"attachments,omitempty"`
	Blocks      []SlackBlock      `json:"blocks,omitempty"`
}

// SlackAttachment represents a Slack message attachment
type SlackAttachment struct {
	Color      string       `json:"color,omitempty"`
	Pretext    string       `json:"pretext,omitempty"`
	Title      string       `json:"title,omitempty"`
	TitleLink  string       `json:"title_link,omitempty"`
	Text       string       `json:"text,omitempty"`
	Fields     []SlackField `json:"fields,omitempty"`
	Footer     string       `json:"footer,omitempty"`
	FooterIcon string       `json:"footer_icon,omitempty"`
	Timestamp  int64        `json:"ts,omitempty"`
	ThumbURL   string       `json:"thumb_url,omitempty"`
	AuthorName string       `json:"author_name,omitempty"`
	AuthorLink string       `json:"author_link,omitempty"`
	AuthorIcon string       `json:"author_icon,omitempty"`
}

// SlackField represents a field in a Slack attachment
type SlackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// SlackBlock represents a Slack block element
type SlackBlock struct {
	Type     string             `json:"type"`
	Text     *SlackTextElement  `json:"text,omitempty"`
	Elements []SlackElement     `json:"elements,omitempty"`
	Fields   []SlackTextElement `json:"fields,omitempty"`
}

// SlackTextElement represents a text element in Slack blocks
type SlackTextElement struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// SlackElement represents an element in Slack blocks
type SlackElement struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	URL  string `json:"url,omitempty"`
}

// NewSlackChannel creates a new Slack notification channel
func NewSlackChannel(config *types.SlackConfig) *SlackChannel {
	return &SlackChannel{
		config: config,
		rateLimit: &types.RateLimit{
			RequestsPerMinute: 60, // Slack rate limit
			RequestsPerHour:   3600,
			RequestsPerDay:    86400,
			BurstSize:         10,
		},
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Send implements the NotificationChannel interface for Slack
func (s *SlackChannel) Send(ctx context.Context, notification *types.Notification) (*types.DeliveryResult, error) {
	startTime := time.Now()
	result := &types.DeliveryResult{
		Channel:   types.ChannelSlack,
		Timestamp: startTime,
	}

	// Build Slack message
	message := s.buildSlackMessage(notification)

	// Marshal message to JSON
	payload, err := json.Marshal(message)
	if err != nil {
		result.Error = fmt.Errorf("failed to marshal Slack message: %w", err)
		return result, result.Error
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", s.config.WebhookURL, bytes.NewBuffer(payload))
	if err != nil {
		result.Error = fmt.Errorf("failed to create request: %w", err)
		return result, result.Error
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := s.client.Do(req)
	if err != nil {
		result.Error = fmt.Errorf("failed to send request: %w", err)
		return result, result.Error
	}
	defer func() { _ = resp.Body.Close() }()

	result.DeliveryTime = time.Since(startTime)

	// Check response
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		result.Success = true
		result.MessageID = fmt.Sprintf("slack_%d", time.Now().Unix())
	} else {
		result.Error = fmt.Errorf("%w: %d", ErrSlackAPIError, resp.StatusCode)
	}

	return result, nil
}

// ValidateConfig validates the Slack channel configuration
func (s *SlackChannel) ValidateConfig() error {
	if s.config == nil {
		return ErrSlackConfigNil
	}

	if s.config.WebhookURL == "" {
		return ErrSlackWebhookRequired
	}

	// Validate webhook URL format
	if !isValidSlackWebhookURL(s.config.WebhookURL) {
		return ErrSlackWebhookInvalid
	}

	return nil
}

// GetChannelType returns the channel type
func (s *SlackChannel) GetChannelType() types.ChannelType {
	return types.ChannelSlack
}

// SupportsRichContent returns whether the channel supports rich content
func (s *SlackChannel) SupportsRichContent() bool {
	return true
}

// GetRateLimit returns the rate limit configuration
func (s *SlackChannel) GetRateLimit() *types.RateLimit {
	return s.rateLimit
}

// buildSlackMessage builds a Slack message from a notification
func (s *SlackChannel) buildSlackMessage(notification *types.Notification) *SlackMessage {
	message := &SlackMessage{
		Username:  s.config.Username,
		IconEmoji: s.config.IconEmoji,
		IconURL:   s.config.IconURL,
		Channel:   s.config.Channel,
		LinkNames: true,
	}

	// Use rich content if available
	if notification.RichContent != nil && notification.RichContent.Markdown != "" {
		message.Text = notification.RichContent.Markdown
	} else {
		message.Text = fmt.Sprintf("*%s*\n%s", notification.Subject, notification.Message)
	}

	// Add attachment with detailed information
	attachment := s.buildSlackAttachment(notification)
	if attachment != nil {
		message.Attachments = []SlackAttachment{*attachment}
	}

	return message
}

// buildSlackAttachment builds a Slack attachment from notification data
func (s *SlackChannel) buildSlackAttachment(notification *types.Notification) *SlackAttachment {
	attachment := &SlackAttachment{
		Color:     s.getSeverityColor(notification.Severity),
		Title:     notification.Subject,
		Text:      notification.Message,
		Footer:    "GoFortress Coverage",
		Timestamp: notification.Timestamp.Unix(),
		Fields:    make([]SlackField, 0),
	}

	// Add repository information
	if notification.Repository != "" {
		attachment.Fields = append(attachment.Fields, SlackField{
			Title: "Repository",
			Value: notification.Repository,
			Short: true,
		})
	}

	// Add branch information
	if notification.Branch != "" {
		attachment.Fields = append(attachment.Fields, SlackField{
			Title: "Branch",
			Value: notification.Branch,
			Short: true,
		})
	}

	// Add PR information
	if notification.PRNumber > 0 {
		prLink := fmt.Sprintf("https://github.com/%s/pull/%d", notification.Repository, notification.PRNumber)
		attachment.Fields = append(attachment.Fields, SlackField{
			Title: "Pull Request",
			Value: fmt.Sprintf("<%s|#%d>", prLink, notification.PRNumber),
			Short: true,
		})
	}

	// Add coverage information
	if notification.CoverageData != nil {
		coverageText := fmt.Sprintf("%.1f%%", notification.CoverageData.Current)
		if notification.CoverageData.Previous > 0 {
			change := notification.CoverageData.Change
			changeIcon := "🔄"
			if change > 0 {
				changeIcon = "📈"
			} else if change < 0 {
				changeIcon = "📉"
			}
			coverageText += fmt.Sprintf(" (%s %+.1f%%)", changeIcon, change)
		}

		attachment.Fields = append(attachment.Fields, SlackField{
			Title: "Coverage",
			Value: coverageText,
			Short: true,
		})
	}

	// Add author information
	if notification.Author != "" {
		attachment.AuthorName = notification.Author
		attachment.AuthorIcon = fmt.Sprintf("https://github.com/%s.png", notification.Author)
		attachment.AuthorLink = fmt.Sprintf("https://github.com/%s", notification.Author)
	}

	// Add trend information
	if notification.TrendData != nil {
		trendIcon := "📊"
		switch notification.TrendData.Direction {
		case "upward":
			trendIcon = "📈"
		case "downward":
			trendIcon = "📉"
		}

		attachment.Fields = append(attachment.Fields, SlackField{
			Title: "Trend",
			Value: fmt.Sprintf("%s %s (%.0f%% confidence)", trendIcon, notification.TrendData.Direction, notification.TrendData.Confidence*100),
			Short: true,
		})
	}

	// Add commit information
	if notification.CommitSHA != "" {
		commitURL := fmt.Sprintf("https://github.com/%s/commit/%s", notification.Repository, notification.CommitSHA)
		shortSHA := notification.CommitSHA
		if len(shortSHA) > 8 {
			shortSHA = shortSHA[:8]
		}
		attachment.Fields = append(attachment.Fields, SlackField{
			Title: "Commit",
			Value: fmt.Sprintf("<%s|%s>", commitURL, shortSHA),
			Short: true,
		})
	}

	return attachment
}

// getSeverityColor returns the color for a severity level
func (s *SlackChannel) getSeverityColor(severity types.SeverityLevel) string {
	switch severity {
	case types.SeverityInfo:
		return "good"
	case types.SeverityWarning:
		return "warning"
	case types.SeverityCritical:
		return "danger"
	case types.SeverityEmergency:
		return "#ff0000"
	default:
		return "#cccccc"
	}
}

// isValidSlackWebhookURL validates a Slack webhook URL
func isValidSlackWebhookURL(url string) bool {
	return len(url) > 20 && (containsString(url, "hooks.slack.com/services/") ||
		containsString(url, "hooks.slack.com/workflows/"))
}

// Helper function
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && findString(s, substr) != -1
}

func findString(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
