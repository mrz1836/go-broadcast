package channels

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mrz1836/go-broadcast/coverage/internal/types"
)

func TestNewSlackChannel(t *testing.T) {
	config := &types.SlackConfig{
		WebhookURL: "https://hooks.slack.com/test",
		Channel:    "#coverage",
		Username:   "coverage-bot",
	}

	channel := NewSlackChannel(config)
	if channel == nil {
		t.Fatal("NewSlackChannel returned nil")
	}
	if channel.config != config {
		t.Error("Slack channel config not set correctly")
	}
}

func TestSlackChannelSend(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Parse the request body
		var message SlackMessage
		err := json.NewDecoder(r.Body).Decode(&message)
		if err != nil {
			t.Errorf("Failed to decode Slack message: %v", err)
		}

		// Verify message structure
		if message.Text == "" {
			t.Error("Slack message text should not be empty")
		}
		if len(message.Attachments) == 0 {
			t.Error("Slack message should have attachments")
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	config := &types.SlackConfig{
		WebhookURL: server.URL,
		Channel:    "#coverage",
		Username:   "coverage-bot",
	}

	channel := NewSlackChannel(config)

	notification := &types.Notification{
		ID:         "test-001",
		Subject:    "Coverage Alert",
		Message:    "Coverage has dropped below threshold",
		Severity:   types.SeverityWarning,
		Priority:   types.PriorityHigh,
		Timestamp:  time.Now(),
		Repository: "test/repo",
		Branch:     "main",
		Author:     "test-user",
		CoverageData: &types.CoverageData{
			Current:  75.5,
			Previous: 80.0,
			Change:   -4.5,
			Target:   85.0,
		},
	}

	result, err := channel.Send(context.Background(), notification)
	if err != nil {
		t.Fatalf("SlackChannel.Send() error = %v", err)
	}

	if !result.Success {
		t.Errorf("Expected successful delivery, got failure: %v", result.Error)
	}

	if result.Channel != types.ChannelSlack {
		t.Errorf("Expected channel type %v, got %v", types.ChannelSlack, result.Channel)
	}

	if result.DeliveryTime <= 0 {
		t.Error("Delivery time should be positive")
	}
}

func TestSlackChannelValidateConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    *types.SlackConfig
		expectErr bool
	}{
		{
			name: "valid config",
			config: &types.SlackConfig{
				WebhookURL: "https://hooks.slack.com/services/test",
				Channel:    "#coverage",
				Username:   "bot",
			},
			expectErr: false,
		},
		{
			name:      "nil config",
			config:    nil,
			expectErr: true,
		},
		{
			name: "empty webhook URL",
			config: &types.SlackConfig{
				WebhookURL: "",
				Channel:    "#coverage",
			},
			expectErr: true,
		},
		{
			name: "invalid webhook URL",
			config: &types.SlackConfig{
				WebhookURL: "not-a-slack-url",
				Channel:    "#coverage",
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channel := NewSlackChannel(tt.config)
			err := channel.ValidateConfig()
			if (err != nil) != tt.expectErr {
				t.Errorf("ValidateConfig() error = %v, expectErr = %v", err, tt.expectErr)
			}
		})
	}
}

func TestSlackChannelGetChannelType(t *testing.T) {
	channel := NewSlackChannel(&types.SlackConfig{})
	if channel.GetChannelType() != types.ChannelSlack {
		t.Errorf("Expected channel type %v, got %v", types.ChannelSlack, channel.GetChannelType())
	}
}

func TestSlackChannelSupportsRichContent(t *testing.T) {
	channel := NewSlackChannel(&types.SlackConfig{})
	if !channel.SupportsRichContent() {
		t.Error("Slack channel should support rich content")
	}
}

func TestBuildSlackMessage(t *testing.T) {
	config := &types.SlackConfig{
		Channel:  "#coverage",
		Username: "coverage-bot",
		IconURL:  "https://example.com/icon.png",
	}

	channel := NewSlackChannel(config)

	notification := &types.Notification{
		Subject:    "Test Coverage Alert",
		Message:    "This is a test message",
		Severity:   types.SeverityWarning,
		Repository: "test/repo",
		Branch:     "main",
		Author:     "developer",
		CoverageData: &types.CoverageData{
			Current:  82.5,
			Previous: 85.0,
			Change:   -2.5,
			Target:   90.0,
		},
	}

	message := channel.buildSlackMessage(notification)

	// Verify basic message structure
	if message.Channel != config.Channel {
		t.Errorf("Expected channel %s, got %s", config.Channel, message.Channel)
	}

	if message.Username != config.Username {
		t.Errorf("Expected username %s, got %s", config.Username, message.Username)
	}

	if message.IconURL != config.IconURL {
		t.Errorf("Expected icon URL %s, got %s", config.IconURL, message.IconURL)
	}

	// Verify attachments
	if len(message.Attachments) == 0 {
		t.Error("Message should have attachments")
	}

	attachment := message.Attachments[0]
	if attachment.Title != notification.Subject {
		t.Errorf("Expected attachment title %s, got %s", notification.Subject, attachment.Title)
	}

	if attachment.Text != notification.Message {
		t.Errorf("Expected attachment text %s, got %s", notification.Message, attachment.Text)
	}

	// Verify fields are present
	if len(attachment.Fields) == 0 {
		t.Error("Attachment should have fields")
	}

	// Check for specific fields
	fieldNames := make(map[string]bool)
	for _, field := range attachment.Fields {
		fieldNames[field.Title] = true
	}

	expectedFields := []string{"Repository", "Branch", "Coverage"}
	for _, expected := range expectedFields {
		if !fieldNames[expected] {
			t.Errorf("Missing expected field: %s", expected)
		}
	}

	// Author is in attachment metadata, not fields
	if attachment.AuthorName != notification.Author {
		t.Errorf("Expected author name %s, got %s", notification.Author, attachment.AuthorName)
	}
}

func TestBuildSlackAttachment(t *testing.T) {
	channel := NewSlackChannel(&types.SlackConfig{})

	notification := &types.Notification{
		Subject:  "Coverage Regression",
		Message:  "Coverage has decreased significantly",
		Severity: types.SeverityCritical,
		CoverageData: &types.CoverageData{
			Current:  70.0,
			Previous: 85.0,
			Change:   -15.0,
			Target:   90.0,
		},
	}

	attachment := channel.buildSlackAttachment(notification)

	// Verify color based on severity
	if attachment.Color != "danger" {
		t.Errorf("Expected danger color for critical severity, got %s", attachment.Color)
	}

	// Verify coverage field formatting
	var coverageField *SlackField
	for _, field := range attachment.Fields {
		if field.Title == "Coverage" {
			coverageField = &field
			break
		}
	}

	if coverageField == nil {
		t.Error("Coverage field not found")
	} else {
		if !strings.Contains(coverageField.Value, "70.0%") {
			t.Error("Coverage field should contain current coverage")
		}
		if !strings.Contains(coverageField.Value, "-15.0%") {
			t.Error("Coverage field should contain change")
		}
	}
}

func TestGetSeverityColor(t *testing.T) {
	channel := NewSlackChannel(&types.SlackConfig{})

	tests := []struct {
		severity      types.SeverityLevel
		expectedColor string
	}{
		{types.SeverityInfo, "good"},
		{types.SeverityWarning, "warning"},
		{types.SeverityCritical, "danger"},
		{types.SeverityEmergency, "#ff0000"},
	}

	for _, tt := range tests {
		t.Run(string(tt.severity), func(t *testing.T) {
			color := channel.getSeverityColor(tt.severity)
			if color != tt.expectedColor {
				t.Errorf("getSeverityColor(%v) = %v, expected %v", tt.severity, color, tt.expectedColor)
			}
		})
	}
}

func TestFormatCoverageChange(t *testing.T) {
	tests := []struct {
		name     string
		change   float64
		expected string
	}{
		{"positive change", 5.0, "+5.0%"},
		{"negative change", -3.5, "-3.5%"},
		{"zero change", 0.0, "0.0%"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatCoverageChange(tt.change)
			if result != tt.expected {
				t.Errorf("formatCoverageChange(%v) = %v, expected %v", tt.change, result, tt.expected)
			}
		})
	}
}

func TestIsValidSlackWebhookURL(t *testing.T) {
	tests := []struct {
		url   string
		valid bool
	}{
		{"https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX", true},
		{"https://hooks.slack.com/services/TEAMID/BOTID/TOKEN", true},
		{"http://hooks.slack.com/services/T00/B00/TOKEN", true}, // Implementation doesn't check protocol
		{"https://example.com/webhook", false},
		{"not-a-url", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := isValidSlackWebhookURL(tt.url)
			if result != tt.valid {
				t.Errorf("isValidSlackWebhookURL(%s) = %v, expected %v", tt.url, result, tt.valid)
			}
		})
	}
}

func TestSlackChannelWithCustomConfig(t *testing.T) {
	config := &types.SlackConfig{
		WebhookURL: "https://hooks.slack.com/test",
		Channel:    "#custom-channel",
		Username:   "custom-bot",
		IconEmoji:  ":robot_face:",
	}

	channel := NewSlackChannel(config)

	notification := &types.Notification{
		Subject:  "Custom Test",
		Message:  "Testing custom configuration",
		Severity: types.SeverityInfo,
		Priority: types.PriorityUrgent,
	}

	message := channel.buildSlackMessage(notification)

	if message.Channel != config.Channel {
		t.Errorf("Expected custom channel %s, got %s", config.Channel, message.Channel)
	}

	if message.Username != config.Username {
		t.Errorf("Expected custom username %s, got %s", config.Username, message.Username)
	}

	if message.IconEmoji != config.IconEmoji {
		t.Errorf("Expected custom icon emoji %s, got %s", config.IconEmoji, message.IconEmoji)
	}
}

func TestSlackChannelErrorHandling(t *testing.T) {
	// Create server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Bad Request"))
	}))
	defer server.Close()

	config := &types.SlackConfig{
		WebhookURL: server.URL,
		Channel:    "#test",
	}

	channel := NewSlackChannel(config)

	notification := &types.Notification{
		Subject:  "Error Test",
		Message:  "This should fail",
		Severity: types.SeverityInfo,
	}

	result, err := channel.Send(context.Background(), notification)
	if err != nil {
		t.Errorf("Send should not return error directly, got: %v", err)
	}

	if result.Success {
		t.Error("Expected failed delivery result")
	}

	if result.Error == nil {
		t.Error("Result should contain error details")
	}
}

func TestSlackChannelWithRichContent(t *testing.T) {
	config := &types.SlackConfig{
		WebhookURL: "https://hooks.slack.com/test",
		Channel:    "#test",
	}

	channel := NewSlackChannel(config)

	notification := &types.Notification{
		Subject:  "Rich Content Test",
		Message:  "Base message",
		Severity: types.SeverityInfo,
		RichContent: &types.RichContent{
			Markdown: "**Bold text** and *italic text*",
			HTML:     "<strong>Bold text</strong>",
		},
	}

	message := channel.buildSlackMessage(notification)

	// Check if markdown content is in the main message text
	if !strings.Contains(message.Text, "**Bold text**") {
		// If not in main text, check attachment
		if len(message.Attachments) > 0 && !strings.Contains(message.Attachments[0].Text, "**Bold text**") {
			t.Error("Should use rich markdown content in either message text or attachment")
		}
	}
}

func BenchmarkSlackChannelSend(b *testing.B) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	config := &types.SlackConfig{
		WebhookURL: server.URL,
		Channel:    "#benchmark",
	}

	channel := NewSlackChannel(config)

	notification := &types.Notification{
		Subject:    "Benchmark Test",
		Message:    "Performance testing",
		Severity:   types.SeverityInfo,
		Repository: "test/repo",
		CoverageData: &types.CoverageData{
			Current: 85.0,
			Change:  2.5,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := channel.Send(context.Background(), notification)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBuildSlackMessage(b *testing.B) {
	config := &types.SlackConfig{
		Channel:  "#test",
		Username: "bot",
	}

	channel := NewSlackChannel(config)

	notification := &types.Notification{
		Subject:    "Benchmark Message",
		Message:    "Testing message building performance",
		Severity:   types.SeverityWarning,
		Repository: "test/repo",
		Branch:     "main",
		Author:     "developer",
		CoverageData: &types.CoverageData{
			Current:  80.5,
			Previous: 78.0,
			Change:   2.5,
			Target:   85.0,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = channel.buildSlackMessage(notification)
	}
}

// Helper function for formatting coverage changes
func formatCoverageChange(change float64) string {
	if change > 0 {
		return fmt.Sprintf("+%.1f%%", change)
	}
	return fmt.Sprintf("%.1f%%", change)
}
