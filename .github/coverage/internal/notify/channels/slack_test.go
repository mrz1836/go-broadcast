package channels

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mrz1836/go-broadcast/coverage/internal/notify"
)

func TestNewSlackChannel(t *testing.T) {
	config := &notify.SlackConfig{
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
		w.Write([]byte("ok"))
	}))
	defer server.Close()
	
	config := &notify.SlackConfig{
		WebhookURL: server.URL,
		Channel:    "#coverage",
		Username:   "coverage-bot",
	}
	
	channel := NewSlackChannel(config)
	
	notification := &notify.Notification{
		ID:        "test-001",
		EventType: notify.EventCoverageThreshold,
		Subject:   "Coverage Alert",
		Message:   "Coverage has dropped below threshold",
		Severity:  notify.SeverityWarning,
		Priority:  notify.PriorityHigh,
		Timestamp: time.Now(),
		Repository: "test/repo",
		Branch:     "main",
		Author:     "test-user",
		CoverageData: &notify.CoverageData{
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
	
	if result.Channel != notify.ChannelSlack {
		t.Errorf("Expected channel type %v, got %v", notify.ChannelSlack, result.Channel)
	}
	
	if result.DeliveryTime <= 0 {
		t.Error("Delivery time should be positive")
	}
}

func TestSlackChannelValidateConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    *notify.SlackConfig
		expectErr bool
	}{
		{
			name: "valid config",
			config: &notify.SlackConfig{
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
			config: &notify.SlackConfig{
				WebhookURL: "",
				Channel:    "#coverage",
			},
			expectErr: true,
		},
		{
			name: "invalid webhook URL",
			config: &notify.SlackConfig{
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
	channel := NewSlackChannel(&notify.SlackConfig{})
	if channel.GetChannelType() != notify.ChannelSlack {
		t.Errorf("Expected channel type %v, got %v", notify.ChannelSlack, channel.GetChannelType())
	}
}

func TestSlackChannelSupportsRichContent(t *testing.T) {
	channel := NewSlackChannel(&notify.SlackConfig{})
	if !channel.SupportsRichContent() {
		t.Error("Slack channel should support rich content")
	}
}

func TestBuildSlackMessage(t *testing.T) {
	config := &notify.SlackConfig{
		Channel:  "#coverage",
		Username: "coverage-bot",
		IconURL:  "https://example.com/icon.png",
	}
	
	channel := NewSlackChannel(config)
	
	notification := &notify.Notification{
		Subject:    "Test Coverage Alert",
		Message:    "This is a test message",
		Severity:   notify.SeverityWarning,
		Repository: "test/repo",
		Branch:     "main",
		Author:     "developer",
		CoverageData: &notify.CoverageData{
			Current:  82.5,
			Previous: 85.0,
			Change:   -2.5,
			Target:   90.0,
		},
		Links: []notify.Link{
			{Text: "View Report", URL: "https://example.com/report"},
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
	
	expectedFields := []string{"Repository", "Branch", "Author", "Coverage"}
	for _, expected := range expectedFields {
		if !fieldNames[expected] {
			t.Errorf("Missing expected field: %s", expected)
		}
	}
}

func TestBuildSlackAttachment(t *testing.T) {
	channel := NewSlackChannel(&notify.SlackConfig{})
	
	notification := &notify.Notification{
		Subject:  "Coverage Regression",
		Message:  "Coverage has decreased significantly",
		Severity: notify.SeverityCritical,
		CoverageData: &notify.CoverageData{
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
	channel := NewSlackChannel(&notify.SlackConfig{})
	
	tests := []struct {
		severity      notify.SeverityLevel
		expectedColor string
	}{
		{notify.SeverityInfo, "good"},
		{notify.SeverityWarning, "warning"},
		{notify.SeverityCritical, "danger"},
		{notify.SeverityEmergency, "danger"},
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
		{"http://hooks.slack.com/services/T00/B00/TOKEN", false}, // HTTP not HTTPS
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
	config := &notify.SlackConfig{
		WebhookURL:   "https://hooks.slack.com/test",
		Channel:      "#custom-channel",
		Username:     "custom-bot",
		IconEmoji:    ":robot_face:",
		ThreadReply:  true,
		Mentioning:   true,
	}
	
	channel := NewSlackChannel(config)
	
	notification := &notify.Notification{
		Subject:   "Custom Test",
		Message:   "Testing custom configuration",
		Severity:  notify.SeverityInfo,
		Priority:  notify.PriorityUrgent,
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Bad Request"))
	}))
	defer server.Close()
	
	config := &notify.SlackConfig{
		WebhookURL: server.URL,
		Channel:    "#test",
	}
	
	channel := NewSlackChannel(config)
	
	notification := &notify.Notification{
		Subject:  "Error Test",
		Message:  "This should fail",
		Severity: notify.SeverityInfo,
	}
	
	result, err := channel.Send(context.Background(), notification)
	if err == nil {
		t.Error("Expected error for bad request, got nil")
	}
	
	if result.Success {
		t.Error("Expected failed delivery result")
	}
	
	if result.Error == nil {
		t.Error("Result should contain error details")
	}
}

func TestSlackChannelWithRichContent(t *testing.T) {
	config := &notify.SlackConfig{
		WebhookURL: "https://hooks.slack.com/test",
		Channel:    "#test",
	}
	
	channel := NewSlackChannel(config)
	
	notification := &notify.Notification{
		Subject:  "Rich Content Test",
		Message:  "Base message",
		Severity: notify.SeverityInfo,
		RichContent: &notify.RichContent{
			Markdown: "**Bold text** and *italic text*",
			HTML:     "<strong>Bold text</strong>",
		},
	}
	
	message := channel.buildSlackMessage(notification)
	
	// Should use markdown content when available
	if !strings.Contains(message.Attachments[0].Text, "**Bold text**") {
		t.Error("Should use rich markdown content")
	}
}

func BenchmarkSlackChannelSend(b *testing.B) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()
	
	config := &notify.SlackConfig{
		WebhookURL: server.URL,
		Channel:    "#benchmark",
	}
	
	channel := NewSlackChannel(config)
	
	notification := &notify.Notification{
		Subject:    "Benchmark Test",
		Message:    "Performance testing",
		Severity:   notify.SeverityInfo,
		Repository: "test/repo",
		CoverageData: &notify.CoverageData{
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
	config := &notify.SlackConfig{
		Channel:  "#test",
		Username: "bot",
	}
	
	channel := NewSlackChannel(config)
	
	notification := &notify.Notification{
		Subject:    "Benchmark Message",
		Message:    "Testing message building performance",
		Severity:   notify.SeverityWarning,
		Repository: "test/repo",
		Branch:     "main",
		Author:     "developer",
		CoverageData: &notify.CoverageData{
			Current:  80.5,
			Previous: 78.0,
			Change:   2.5,
			Target:   85.0,
		},
		Links: []notify.Link{
			{Text: "Report", URL: "https://example.com"},
		},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = channel.buildSlackMessage(notification)
	}
}