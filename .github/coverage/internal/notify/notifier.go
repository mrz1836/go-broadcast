// Package notify provides multi-channel notification capabilities for coverage events
package notify

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mrz1836/go-broadcast/coverage/internal/types"
)

// Static error definitions
var (
	ErrNotificationEngineDisabled = errors.New("notification engine is disabled")
	ErrNoChannelsConfigured       = errors.New("no channels configured for notification")
	ErrChannelSendFailures        = errors.New("failed to send to some channels")
)

// NotificationEngine manages multi-channel notifications for coverage events
type NotificationEngine struct {
	config   *NotificationConfig
	channels map[types.ChannelType]types.NotificationChannel
}

// NotificationConfig holds configuration for the notification system
type NotificationConfig struct {
	// Global settings
	Enabled          bool                `json:"enabled"`
	DefaultChannels  []types.ChannelType `json:"default_channels"`
	RateLimitPerHour int                 `json:"rate_limit_per_hour"`
	RetryAttempts    int                 `json:"retry_attempts"`
	RetryDelay       time.Duration       `json:"retry_delay"`

	// Channel configurations
	SlackConfig   *types.SlackConfig   `json:"slack_config,omitempty"`
	DiscordConfig *types.DiscordConfig `json:"discord_config,omitempty"`
	EmailConfig   *types.EmailConfig   `json:"email_config,omitempty"`
	WebhookConfig *types.WebhookConfig `json:"webhook_config,omitempty"`
	TeamsConfig   *types.TeamsConfig   `json:"teams_config,omitempty"`
}

// NewNotificationEngine creates a new notification engine with the provided configuration
func NewNotificationEngine(config *NotificationConfig) *NotificationEngine {
	if config == nil {
		config = &NotificationConfig{
			Enabled:          true,
			DefaultChannels:  []types.ChannelType{types.ChannelSlack},
			RateLimitPerHour: 100,
			RetryAttempts:    3,
			RetryDelay:       time.Minute,
		}
	}

	engine := &NotificationEngine{
		config:   config,
		channels: make(map[types.ChannelType]types.NotificationChannel),
	}

	// Initialize enabled channels
	engine.initializeChannels()

	return engine
}

// Send sends a notification through the specified channels
func (ne *NotificationEngine) Send(ctx context.Context, notification *types.Notification) error {
	if !ne.config.Enabled {
		return ErrNotificationEngineDisabled
	}

	if len(notification.Metadata) == 0 {
		notification.Metadata = make(map[string]interface{})
	}
	notification.Metadata["engine_id"] = "gofortress-coverage"
	notification.Metadata["timestamp"] = time.Now()

	// Determine which channels to use
	channels := ne.config.DefaultChannels
	if len(channels) == 0 {
		return ErrNoChannelsConfigured
	}

	// Send to each channel
	var errors []string
	for _, channelType := range channels {
		if channel, exists := ne.channels[channelType]; exists {
			result := ne.sendWithRetry(ctx, channel, notification)
			if !result.Success {
				errors = append(errors, fmt.Sprintf("%s: %v", channelType, result.Error))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("%w: %s", ErrChannelSendFailures, strings.Join(errors, ", "))
	}

	return nil
}

// sendWithRetry sends a notification with retry logic
func (ne *NotificationEngine) sendWithRetry(ctx context.Context, channel types.NotificationChannel, notification *types.Notification) *types.DeliveryResult {
	var lastError error
	startTime := time.Now()

	for attempt := 0; attempt <= ne.config.RetryAttempts; attempt++ {
		if attempt > 0 {
			// Wait before retry
			select {
			case <-ctx.Done():
				return &types.DeliveryResult{
					Success:      false,
					Channel:      channel.GetChannelType(),
					Timestamp:    time.Now(),
					Error:        ctx.Err(),
					DeliveryTime: time.Since(startTime),
				}
			case <-time.After(ne.config.RetryDelay):
			}
		}

		result, err := channel.Send(ctx, notification)
		if err == nil && result != nil && result.Success {
			return result
		}

		lastError = err
		if result != nil && result.Error != nil {
			lastError = result.Error
		}
	}

	return &types.DeliveryResult{
		Success:      false,
		Channel:      channel.GetChannelType(),
		Timestamp:    time.Now(),
		Error:        lastError,
		DeliveryTime: time.Since(startTime),
	}
}

// initializeChannels initializes the configured notification channels
func (ne *NotificationEngine) initializeChannels() {
	// This would initialize channels based on configuration
	// For now, we'll leave it empty to avoid circular imports
	// TODO: Implement proper channel initialization
	ne.channels = make(map[types.ChannelType]types.NotificationChannel)
}

// GetChannelStatus returns the status of all configured channels
func (ne *NotificationEngine) GetChannelStatus() map[types.ChannelType]bool {
	status := make(map[types.ChannelType]bool)

	for channelType, channel := range ne.channels {
		err := channel.ValidateConfig()
		status[channelType] = err == nil
	}

	return status
}

// AddChannel adds a new notification channel
func (ne *NotificationEngine) AddChannel(channel types.NotificationChannel) error {
	if err := channel.ValidateConfig(); err != nil {
		return fmt.Errorf("invalid channel configuration: %w", err)
	}

	ne.channels[channel.GetChannelType()] = channel
	return nil
}

// RemoveChannel removes a notification channel
func (ne *NotificationEngine) RemoveChannel(channelType types.ChannelType) {
	delete(ne.channels, channelType)
}
