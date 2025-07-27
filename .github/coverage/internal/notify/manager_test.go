package notify

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/mrz1836/go-broadcast/coverage/internal/config"
)

// Mock channel for testing
type mockChannel struct {
	channelType     ChannelType
	deliveryResults []*DeliveryResult
	shouldFail      bool
}

func (m *mockChannel) Send(ctx context.Context, notification *Notification) (*DeliveryResult, error) {
	result := &DeliveryResult{
		Channel:      m.channelType,
		Timestamp:    time.Now(),
		Success:      !m.shouldFail,
		DeliveryTime: 100 * time.Millisecond,
		MessageID:    "mock_123",
	}
	
	if m.shouldFail {
		result.Error = fmt.Errorf("mock delivery failed")
	}
	
	m.deliveryResults = append(m.deliveryResults, result)
	return result, result.Error
}

func (m *mockChannel) ValidateConfig() error {
	return nil
}

func (m *mockChannel) GetChannelType() ChannelType {
	return m.channelType
}

func (m *mockChannel) SupportsRichContent() bool {
	return true
}

func (m *mockChannel) GetRateLimit() *RateLimit {
	return &RateLimit{
		RequestsPerMinute: 60,
		RequestsPerHour:   3600,
		RequestsPerDay:    86400,
		BurstSize:        10,
	}
}

func TestNewNotificationManager(t *testing.T) {
	cfg := &config.Config{
		Notifications: config.NotificationConfig{
			Enabled: true,
			Channels: map[string]config.ChannelConfig{
				"slack": {
					Type:    "slack",
					Enabled: true,
				},
			},
		},
	}
	
	manager := NewNotificationManager(cfg)
	if manager == nil {
		t.Fatal("NewNotificationManager returned nil")
	}
	if manager.config != cfg {
		t.Error("Notification manager config not set correctly")
	}
}

func TestRegisterChannel(t *testing.T) {
	manager := NewNotificationManager(&config.Config{})
	
	mockChan := &mockChannel{channelType: ChannelSlack}
	
	err := manager.RegisterChannel("test-slack", mockChan)
	if err != nil {
		t.Fatalf("RegisterChannel() error = %v", err)
	}
	
	// Verify channel was registered
	if _, exists := manager.channels["test-slack"]; !exists {
		t.Error("Channel was not registered correctly")
	}
}

func TestSendNotification(t *testing.T) {
	manager := NewNotificationManager(&config.Config{})
	
	// Register mock channels
	slackMock := &mockChannel{channelType: ChannelSlack}
	emailMock := &mockChannel{channelType: ChannelEmail}
	
	manager.RegisterChannel("slack", slackMock)
	manager.RegisterChannel("email", emailMock)
	
	notification := &Notification{
		ID:        "test-001",
		EventType: EventCoverageThreshold,
		Subject:   "Coverage Alert",
		Message:   "Coverage has dropped below threshold",
		Severity:  SeverityWarning,
		Priority:  PriorityHigh,
		Urgency:   UrgencyNormal,
		Timestamp: time.Now(),
		Repository: "test/repo",
		Branch:     "main",
		Author:     "test-user",
	}
	
	tests := []struct {
		name         string
		channels     []string
		expectSuccess bool
	}{
		{
			name:         "send to single channel",
			channels:     []string{"slack"},
			expectSuccess: true,
		},
		{
			name:         "send to multiple channels",
			channels:     []string{"slack", "email"},
			expectSuccess: true,
		},
		{
			name:         "send to non-existent channel",
			channels:     []string{"invalid"},
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := manager.SendNotification(context.Background(), notification, tt.channels)
			
			if tt.expectSuccess {
				if err != nil {
					t.Errorf("SendNotification() error = %v, expected success", err)
				}
				if len(results) != len(tt.channels) {
					t.Errorf("Expected %d results, got %d", len(tt.channels), len(results))
				}
			} else {
				if err == nil {
					t.Error("SendNotification() expected error but got none")
				}
			}
		})
	}
}

func TestSendToAllChannels(t *testing.T) {
	manager := NewNotificationManager(&config.Config{})
	
	// Register multiple mock channels
	channels := map[string]*mockChannel{
		"slack":   {channelType: ChannelSlack},
		"email":   {channelType: ChannelEmail},
		"discord": {channelType: ChannelDiscord},
	}
	
	for name, ch := range channels {
		manager.RegisterChannel(name, ch)
	}
	
	notification := &Notification{
		ID:        "broadcast-001",
		EventType: EventSystemAlert,
		Subject:   "System Alert",
		Message:   "Critical system event",
		Severity:  SeverityCritical,
		Priority:  PriorityUrgent,
		Timestamp: time.Now(),
	}
	
	results, err := manager.SendToAllChannels(context.Background(), notification)
	if err != nil {
		t.Fatalf("SendToAllChannels() error = %v", err)
	}
	
	if len(results) != len(channels) {
		t.Errorf("Expected %d results, got %d", len(channels), len(results))
	}
	
	// Verify all channels received the notification
	for _, result := range results {
		if !result.Success {
			t.Errorf("Channel %s delivery failed: %v", result.Channel, result.Error)
		}
	}
}

func TestNotificationFiltering(t *testing.T) {
	cfg := &config.Config{
		Notifications: config.NotificationConfig{
			Enabled: true,
			Filters: []config.NotificationFilter{
				{
					EventTypes:     []string{"coverage_threshold"},
					MinSeverity:    "warning",
					Channels:       []string{"slack"},
					BranchPatterns: []string{"main", "develop"},
				},
			},
		},
	}
	
	manager := NewNotificationManager(cfg)
	slackMock := &mockChannel{channelType: ChannelSlack}
	manager.RegisterChannel("slack", slackMock)
	
	tests := []struct {
		name         string
		notification *Notification
		expectSend   bool
	}{
		{
			name: "matching filter",
			notification: &Notification{
				EventType: EventCoverageThreshold,
				Severity:  SeverityWarning,
				Branch:    "main",
			},
			expectSend: true,
		},
		{
			name: "severity too low",
			notification: &Notification{
				EventType: EventCoverageThreshold,
				Severity:  SeverityInfo,
				Branch:    "main",
			},
			expectSend: false,
		},
		{
			name: "wrong branch",
			notification: &Notification{
				EventType: EventCoverageThreshold,
				Severity:  SeverityWarning,
				Branch:    "feature/test",
			},
			expectSend: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed := manager.shouldSendNotification(tt.notification, "slack")
			if allowed != tt.expectSend {
				t.Errorf("shouldSendNotification() = %v, expected %v", allowed, tt.expectSend)
			}
		})
	}
}

func TestRateLimiting(t *testing.T) {
	manager := NewNotificationManager(&config.Config{})
	
	// Set up channel with tight rate limits for testing
	mockChan := &mockChannel{
		channelType: ChannelSlack,
	}
	
	// Override rate limit for testing
	manager.rateLimiters = map[string]*RateLimiter{
		"slack": NewRateLimiter(&RateLimit{
			RequestsPerMinute: 2,   // Very restrictive for testing
			RequestsPerHour:   10,
			RequestsPerDay:    100,
			BurstSize:        1,
		}),
	}
	
	manager.RegisterChannel("slack", mockChan)
	
	notification := &Notification{
		ID:        "rate-test",
		EventType: EventCoverageThreshold,
		Subject:   "Rate Limit Test",
		Severity:  SeverityInfo,
		Timestamp: time.Now(),
	}
	
	// Send notifications rapidly
	successCount := 0
	for i := 0; i < 5; i++ {
		results, err := manager.SendNotification(context.Background(), notification, []string{"slack"})
		if err == nil && len(results) > 0 && results[0].Success {
			successCount++
		}
		time.Sleep(10 * time.Millisecond) // Brief delay
	}
	
	// Should be rate limited after burst
	if successCount >= 5 {
		t.Error("Rate limiting not working - all notifications went through")
	}
}

func TestNotificationTemplating(t *testing.T) {
	manager := NewNotificationManager(&config.Config{})
	
	notification := &Notification{
		ID:         "template-test",
		Subject:    "Coverage Alert for {{.Repository}}",
		Message:    "Coverage is {{.CoverageData.Current}}%",
		Repository: "test/repo",
		CoverageData: &CoverageData{
			Current: 85.5,
		},
	}
	
	templated, err := manager.renderNotificationTemplate(notification)
	if err != nil {
		t.Fatalf("renderNotificationTemplate() error = %v", err)
	}
	
	expectedSubject := "Coverage Alert for test/repo"
	if templated.Subject != expectedSubject {
		t.Errorf("Template subject = %v, expected %v", templated.Subject, expectedSubject)
	}
	
	expectedMessage := "Coverage is 85.5%"
	if templated.Message != expectedMessage {
		t.Errorf("Template message = %v, expected %v", templated.Message, expectedMessage)
	}
}

func TestNotificationRetry(t *testing.T) {
	manager := NewNotificationManager(&config.Config{})
	
	// Mock channel that fails initially then succeeds
	failingMock := &mockChannel{
		channelType: ChannelSlack,
		shouldFail:  true,
	}
	
	manager.RegisterChannel("slack", failingMock)
	
	notification := &Notification{
		ID:        "retry-test",
		EventType: EventCoverageThreshold,
		Subject:   "Retry Test",
		Severity:  SeverityInfo,
		Timestamp: time.Now(),
	}
	
	// First attempt should fail
	results, err := manager.SendNotification(context.Background(), notification, []string{"slack"})
	if err == nil {
		t.Error("Expected first attempt to fail")
	}
	
	// Make the channel succeed now
	failingMock.shouldFail = false
	
	// Retry should succeed
	retryCtx := context.WithValue(context.Background(), "retry_attempt", 1)
	results, err = manager.SendNotification(retryCtx, notification, []string{"slack"})
	if err != nil {
		t.Errorf("Retry attempt failed: %v", err)
	}
	
	if len(results) == 0 || !results[0].Success {
		t.Error("Retry should have succeeded")
	}
}

func TestNotificationBatching(t *testing.T) {
	manager := NewNotificationManager(&config.Config{})
	
	mockChan := &mockChannel{channelType: ChannelSlack}
	manager.RegisterChannel("slack", mockChan)
	
	// Create multiple notifications
	notifications := []*Notification{
		{ID: "batch-1", Subject: "Alert 1", Severity: SeverityInfo},
		{ID: "batch-2", Subject: "Alert 2", Severity: SeverityInfo},
		{ID: "batch-3", Subject: "Alert 3", Severity: SeverityInfo},
	}
	
	batchOptions := BatchOptions{
		MaxBatchSize:   3,
		BatchTimeout:   time.Second,
		Deduplication:  true,
	}
	
	results, err := manager.SendBatchNotifications(context.Background(), notifications, []string{"slack"}, batchOptions)
	if err != nil {
		t.Fatalf("SendBatchNotifications() error = %v", err)
	}
	
	if len(results) != len(notifications) {
		t.Errorf("Expected %d results, got %d", len(notifications), len(results))
	}
}

func TestNotificationAggregation(t *testing.T) {
	manager := NewNotificationManager(&config.Config{})
	
	// Create similar notifications that should be aggregated
	notifications := []*Notification{
		{
			ID:         "agg-1",
			EventType:  EventCoverageThreshold,
			Subject:    "Coverage Alert",
			Repository: "test/repo",
			Branch:     "main",
			Severity:   SeverityWarning,
		},
		{
			ID:         "agg-2",
			EventType:  EventCoverageThreshold,
			Subject:    "Coverage Alert",
			Repository: "test/repo",
			Branch:     "main",
			Severity:   SeverityWarning,
		},
	}
	
	aggregated, err := manager.aggregateNotifications(notifications, AggregationOptions{
		GroupBy:      []string{"event_type", "repository", "branch"},
		TimeWindow:   time.Minute,
		MaxGroupSize: 5,
	})
	if err != nil {
		t.Fatalf("aggregateNotifications() error = %v", err)
	}
	
	// Should aggregate into single notification
	if len(aggregated) != 1 {
		t.Errorf("Expected 1 aggregated notification, got %d", len(aggregated))
	}
	
	if aggregated[0].AggregatedCount != 2 {
		t.Errorf("Expected aggregated count of 2, got %d", aggregated[0].AggregatedCount)
	}
}

func TestGetDeliveryStats(t *testing.T) {
	manager := NewNotificationManager(&config.Config{})
	
	mockChan := &mockChannel{channelType: ChannelSlack}
	manager.RegisterChannel("slack", mockChan)
	
	// Send a few notifications to generate stats
	notification := &Notification{
		ID:        "stats-test",
		Subject:   "Test",
		Severity:  SeverityInfo,
		Timestamp: time.Now(),
	}
	
	for i := 0; i < 3; i++ {
		manager.SendNotification(context.Background(), notification, []string{"slack"})
	}
	
	stats, err := manager.GetDeliveryStats(context.Background(), time.Hour)
	if err != nil {
		t.Fatalf("GetDeliveryStats() error = %v", err)
	}
	
	if stats.TotalNotifications == 0 {
		t.Error("Expected some notifications in stats")
	}
	
	if stats.SuccessRate < 0 || stats.SuccessRate > 1 {
		t.Errorf("Success rate should be between 0 and 1, got %v", stats.SuccessRate)
	}
}

func BenchmarkSendNotification(b *testing.B) {
	manager := NewNotificationManager(&config.Config{})
	mockChan := &mockChannel{channelType: ChannelSlack}
	manager.RegisterChannel("slack", mockChan)
	
	notification := &Notification{
		ID:        "bench-test",
		EventType: EventCoverageThreshold,
		Subject:   "Benchmark Test",
		Message:   "Testing notification performance",
		Severity:  SeverityInfo,
		Timestamp: time.Now(),
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := manager.SendNotification(context.Background(), notification, []string{"slack"})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSendBatchNotifications(b *testing.B) {
	manager := NewNotificationManager(&config.Config{})
	mockChan := &mockChannel{channelType: ChannelSlack}
	manager.RegisterChannel("slack", mockChan)
	
	// Create batch of notifications
	notifications := make([]*Notification, 10)
	for i := 0; i < 10; i++ {
		notifications[i] = &Notification{
			ID:        fmt.Sprintf("batch-%d", i),
			Subject:   fmt.Sprintf("Alert %d", i),
			Severity:  SeverityInfo,
			Timestamp: time.Now(),
		}
	}
	
	batchOptions := BatchOptions{
		MaxBatchSize: 10,
		BatchTimeout: time.Second,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := manager.SendBatchNotifications(context.Background(), notifications, []string{"slack"}, batchOptions)
		if err != nil {
			b.Fatal(err)
		}
	}
}