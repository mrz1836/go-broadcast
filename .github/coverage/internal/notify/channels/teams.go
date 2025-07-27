// Package channels provides Microsoft Teams notification channel implementation
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
	ErrTeamsAPIError         = errors.New("Teams API returned error status")
	ErrTeamsConfigNil        = errors.New("Teams config is nil")
	ErrTeamsWebhookRequired  = errors.New("Teams webhook URL is required")
	ErrTeamsWebhookInvalid   = errors.New("invalid Teams webhook URL format")
)

// TeamsChannel implements Microsoft Teams webhook notifications
type TeamsChannel struct {
	config    *types.TeamsConfig
	rateLimit *types.RateLimit
	client    *http.Client
}

// TeamsMessage represents a Microsoft Teams message payload
type TeamsMessage struct {
	Type       string         `json:"@type"`
	Context    string         `json:"@context"`
	ThemeColor string         `json:"themeColor,omitempty"`
	Summary    string         `json:"summary"`
	Title      string         `json:"title,omitempty"`
	Text       string         `json:"text,omitempty"`
	Sections   []TeamsSection `json:"sections,omitempty"`
	Actions    []TeamsAction  `json:"potentialAction,omitempty"`
}

// TeamsSection represents a section in a Teams message
type TeamsSection struct {
	ActivityTitle    string      `json:"activityTitle,omitempty"`
	ActivitySubtitle string      `json:"activitySubtitle,omitempty"`
	ActivityImage    string      `json:"activityImage,omitempty"`
	Text             string      `json:"text,omitempty"`
	Facts            []TeamsFact `json:"facts,omitempty"`
	Markdown         bool        `json:"markdown,omitempty"`
}

// TeamsFact represents a fact in a Teams section
type TeamsFact struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// TeamsAction represents an action in a Teams message
type TeamsAction struct {
	Type    string        `json:"@type"`
	Name    string        `json:"name"`
	Targets []TeamsTarget `json:"targets,omitempty"`
}

// TeamsTarget represents a target for a Teams action
type TeamsTarget struct {
	OS  string `json:"os"`
	URI string `json:"uri"`
}

// NewTeamsChannel creates a new Microsoft Teams notification channel
func NewTeamsChannel(config *types.TeamsConfig) *TeamsChannel {
	return &TeamsChannel{
		config: config,
		rateLimit: &types.RateLimit{
			RequestsPerMinute: 30, // Teams rate limit (conservative)
			RequestsPerHour:   1800,
			RequestsPerDay:    43200,
			BurstSize:         5,
		},
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Send implements the NotificationChannel interface for Teams
func (t *TeamsChannel) Send(ctx context.Context, notification *types.Notification) (*types.DeliveryResult, error) {
	startTime := time.Now()
	result := &types.DeliveryResult{
		Channel:   types.ChannelTeams,
		Timestamp: startTime,
	}

	// Build Teams message
	message := t.buildTeamsMessage(notification)

	// Marshal message to JSON
	payload, err := json.Marshal(message)
	if err != nil {
		result.Error = fmt.Errorf("failed to marshal Teams message: %w", err)
		return result, result.Error
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", t.config.WebhookURL, bytes.NewBuffer(payload))
	if err != nil {
		result.Error = fmt.Errorf("failed to create request: %w", err)
		return result, result.Error
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := t.client.Do(req)
	if err != nil {
		result.Error = fmt.Errorf("failed to send request: %w", err)
		return result, result.Error
	}
	defer func() { _ = resp.Body.Close() }()

	result.DeliveryTime = time.Since(startTime)

	// Check response
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		result.Success = true
		result.MessageID = fmt.Sprintf("teams_%d", time.Now().Unix())
	} else {
		result.Error = fmt.Errorf("%w: %d", ErrTeamsAPIError, resp.StatusCode)
	}

	return result, nil
}

// ValidateConfig validates the Teams channel configuration
func (t *TeamsChannel) ValidateConfig() error {
	if t.config == nil {
		return ErrTeamsConfigNil
	}

	if t.config.WebhookURL == "" {
		return ErrTeamsWebhookRequired
	}

	// Validate webhook URL format
	if !isValidTeamsWebhookURL(t.config.WebhookURL) {
		return ErrTeamsWebhookInvalid
	}

	return nil
}

// GetChannelType returns the channel type
func (t *TeamsChannel) GetChannelType() types.ChannelType {
	return types.ChannelTeams
}

// SupportsRichContent returns whether the channel supports rich content
func (t *TeamsChannel) SupportsRichContent() bool {
	return true
}

// GetRateLimit returns the rate limit configuration
func (t *TeamsChannel) GetRateLimit() *types.RateLimit {
	return t.rateLimit
}

// buildTeamsMessage builds a Teams message from a notification
func (t *TeamsChannel) buildTeamsMessage(notification *types.Notification) *TeamsMessage {
	message := &TeamsMessage{
		Type:       "MessageCard",
		Context:    "http://schema.org/extensions",
		ThemeColor: t.getThemeColor(notification.Severity),
		Summary:    notification.Subject,
		Title:      notification.Subject,
	}

	// Use custom title if configured
	if t.config.ActivityTitle != "" {
		message.Title = t.config.ActivityTitle
	}

	// Use custom theme color if configured
	if t.config.ThemeColor != "" {
		message.ThemeColor = t.config.ThemeColor
	}

	// Build main section
	section := t.buildMainSection(notification)
	message.Sections = []TeamsSection{*section}

	// Add actions
	actions := t.buildActions(notification)
	if len(actions) > 0 {
		message.Actions = actions
	}

	return message
}

// buildMainSection builds the main section of the Teams message
func (t *TeamsChannel) buildMainSection(notification *types.Notification) *TeamsSection {
	section := &TeamsSection{
		ActivityTitle:    notification.Subject,
		ActivitySubtitle: fmt.Sprintf("%s • %s", notification.Repository, notification.Branch),
		Text:             notification.Message,
		Facts:            make([]TeamsFact, 0),
		Markdown:         true,
	}

	// Add activity image if configured
	if t.config.ActivityImage != "" {
		section.ActivityImage = t.config.ActivityImage
	} else {
		section.ActivityImage = "https://github.com/favicon.ico"
	}

	// Add repository fact
	if notification.Repository != "" {
		section.Facts = append(section.Facts, TeamsFact{
			Name:  "Repository",
			Value: notification.Repository,
		})
	}

	// Add branch fact
	if notification.Branch != "" {
		section.Facts = append(section.Facts, TeamsFact{
			Name:  "Branch",
			Value: notification.Branch,
		})
	}

	// Add PR fact
	if notification.PRNumber > 0 {
		section.Facts = append(section.Facts, TeamsFact{
			Name:  "Pull Request",
			Value: fmt.Sprintf("#%d", notification.PRNumber),
		})
	}

	// Add author fact
	if notification.Author != "" {
		section.Facts = append(section.Facts, TeamsFact{
			Name:  "Author",
			Value: notification.Author,
		})
	}

	// Add coverage fact
	if notification.CoverageData != nil {
		coverageText := fmt.Sprintf("%.1f%%", notification.CoverageData.Current)
		if notification.CoverageData.Previous > 0 {
			change := notification.CoverageData.Change
			if change > 0 {
				coverageText += fmt.Sprintf(" (+%.1f%%)", change)
			} else if change < 0 {
				coverageText += fmt.Sprintf(" (%.1f%%)", change)
			}
		}

		section.Facts = append(section.Facts, TeamsFact{
			Name:  "Coverage",
			Value: coverageText,
		})
	}

	// Add severity fact
	section.Facts = append(section.Facts, TeamsFact{
		Name:  "Severity",
		Value: string(notification.Severity),
	})

	// Add timestamp fact
	section.Facts = append(section.Facts, TeamsFact{
		Name:  "Time",
		Value: notification.Timestamp.Format("2006-01-02 15:04:05"),
	})

	// Add trend information
	if notification.TrendData != nil {
		section.Facts = append(section.Facts, TeamsFact{
			Name:  "Trend",
			Value: fmt.Sprintf("%s (%.0f%% confidence)", notification.TrendData.Direction, notification.TrendData.Confidence*100),
		})
	}

	return section
}

// buildActions builds the actions for the Teams message
func (t *TeamsChannel) buildActions(notification *types.Notification) []TeamsAction {
	actions := make([]TeamsAction, 0)

	// Add commit action
	if notification.CommitSHA != "" {
		commitURL := fmt.Sprintf("https://github.com/%s/commit/%s", notification.Repository, notification.CommitSHA)
		shortSHA := notification.CommitSHA
		if len(shortSHA) > 8 {
			shortSHA = shortSHA[:8]
		}
		commitAction := TeamsAction{
			Type: "OpenUri",
			Name: fmt.Sprintf("View Commit %s", shortSHA),
			Targets: []TeamsTarget{
				{
					OS:  "default",
					URI: commitURL,
				},
			},
		}
		actions = append(actions, commitAction)
	}

	// Add default repository action
	if notification.Repository != "" {
		repoURL := fmt.Sprintf("https://github.com/%s", notification.Repository)
		repoAction := TeamsAction{
			Type: "OpenUri",
			Name: "View Repository",
			Targets: []TeamsTarget{
				{
					OS:  "default",
					URI: repoURL,
				},
			},
		}
		actions = append(actions, repoAction)
	}

	// Add PR action
	if notification.Repository != "" && notification.PRNumber > 0 {
		prURL := fmt.Sprintf("https://github.com/%s/pull/%d", notification.Repository, notification.PRNumber)
		prAction := TeamsAction{
			Type: "OpenUri",
			Name: fmt.Sprintf("View PR #%d", notification.PRNumber),
			Targets: []TeamsTarget{
				{
					OS:  "default",
					URI: prURL,
				},
			},
		}
		actions = append(actions, prAction)
	}

	return actions
}

// getThemeColor returns the theme color for a severity level
func (t *TeamsChannel) getThemeColor(severity types.SeverityLevel) string {
	switch severity {
	case types.SeverityInfo:
		return "0078d4" // Blue
	case types.SeverityWarning:
		return "ffaa44" // Orange
	case types.SeverityCritical:
		return "d13438" // Red
	case types.SeverityEmergency:
		return "a80000" // Dark Red
	default:
		return "6264a7" // Purple
	}
}

// isValidTeamsWebhookURL validates a Teams webhook URL
func isValidTeamsWebhookURL(url string) bool {
	return len(url) > 20 && (containsString(url, "outlook.office.com/webhook/") ||
		containsString(url, "outlook.office365.com/webhook/"))
}
