package events

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/mrz1836/go-broadcast/coverage/internal/notify"
)

// Mock subscriber for testing
type mockSubscriber struct {
	receivedEvents []notify.Event
	mutex          sync.Mutex
	shouldFail     bool
}

func (m *mockSubscriber) HandleEvent(ctx context.Context, event notify.Event) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if m.shouldFail {
		return fmt.Errorf("mock subscriber error")
	}
	
	m.receivedEvents = append(m.receivedEvents, event)
	return nil
}

func (m *mockSubscriber) GetReceivedEvents() []notify.Event {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	events := make([]notify.Event, len(m.receivedEvents))
	copy(events, m.receivedEvents)
	return events
}

func (m *mockSubscriber) Reset() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.receivedEvents = nil
}

func TestNewEventProcessor(t *testing.T) {
	config := EventProcessorConfig{
		BufferSize:       100,
		BatchSize:        10,
		BatchTimeout:     time.Second,
		WorkerCount:      2,
		EnableRetries:    true,
		MaxRetries:       3,
		RetryBackoff:     time.Millisecond * 100,
	}
	
	processor := NewEventProcessor(config)
	if processor == nil {
		t.Fatal("NewEventProcessor returned nil")
	}
	
	if processor.config.BufferSize != config.BufferSize {
		t.Error("Processor config not set correctly")
	}
}

func TestEventProcessorStartStop(t *testing.T) {
	config := EventProcessorConfig{
		BufferSize:   10,
		WorkerCount:  1,
		BatchTimeout: time.Millisecond * 100,
	}
	
	processor := NewEventProcessor(config)
	
	// Start processor
	err := processor.Start(context.Background())
	if err != nil {
		t.Fatalf("Failed to start processor: %v", err)
	}
	
	// Verify it's running
	if !processor.IsRunning() {
		t.Error("Processor should be running")
	}
	
	// Stop processor
	err = processor.Stop()
	if err != nil {
		t.Fatalf("Failed to stop processor: %v", err)
	}
	
	// Verify it's stopped
	if processor.IsRunning() {
		t.Error("Processor should be stopped")
	}
}

func TestEventProcessorSubscription(t *testing.T) {
	processor := NewEventProcessor(EventProcessorConfig{})
	
	subscriber := &mockSubscriber{}
	
	// Subscribe to events
	err := processor.Subscribe(notify.EventCoverageThreshold, subscriber)
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}
	
	// Verify subscription
	subscribers := processor.GetSubscribers(notify.EventCoverageThreshold)
	if len(subscribers) != 1 {
		t.Errorf("Expected 1 subscriber, got %d", len(subscribers))
	}
	
	// Unsubscribe
	err = processor.Unsubscribe(notify.EventCoverageThreshold, subscriber)
	if err != nil {
		t.Fatalf("Failed to unsubscribe: %v", err)
	}
	
	// Verify unsubscription
	subscribers = processor.GetSubscribers(notify.EventCoverageThreshold)
	if len(subscribers) != 0 {
		t.Errorf("Expected 0 subscribers after unsubscribe, got %d", len(subscribers))
	}
}

func TestEventProcessorPublish(t *testing.T) {
	config := EventProcessorConfig{
		BufferSize:   10,
		WorkerCount:  1,
		BatchTimeout: time.Millisecond * 100,
	}
	
	processor := NewEventProcessor(config)
	processor.Start(context.Background())
	defer processor.Stop()
	
	subscriber := &mockSubscriber{}
	processor.Subscribe(notify.EventCoverageThreshold, subscriber)
	
	// Create test event
	event := notify.Event{
		ID:        "test-001",
		Type:      notify.EventCoverageThreshold,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"coverage": 75.0,
			"threshold": 80.0,
		},
		Source:     "test",
		Repository: "test/repo",
		Branch:     "main",
	}
	
	// Publish event
	err := processor.PublishEvent(context.Background(), event)
	if err != nil {
		t.Fatalf("Failed to publish event: %v", err)
	}
	
	// Wait for processing
	time.Sleep(time.Millisecond * 200)
	
	// Verify event was received
	receivedEvents := subscriber.GetReceivedEvents()
	if len(receivedEvents) != 1 {
		t.Errorf("Expected 1 received event, got %d", len(receivedEvents))
	}
	
	if receivedEvents[0].ID != event.ID {
		t.Errorf("Expected event ID %s, got %s", event.ID, receivedEvents[0].ID)
	}
}

func TestEventProcessorFiltering(t *testing.T) {
	config := EventProcessorConfig{
		BufferSize:   10,
		WorkerCount:  1,
		BatchTimeout: time.Millisecond * 50,
	}
	
	processor := NewEventProcessor(config)
	processor.Start(context.Background())
	defer processor.Stop()
	
	subscriber := &mockSubscriber{}
	processor.Subscribe(notify.EventCoverageThreshold, subscriber)
	
	// Add filter
	filter := EventFilter{
		EventTypes:   []notify.EventType{notify.EventCoverageThreshold},
		MinSeverity:  notify.SeverityWarning,
		Repositories: []string{"test/repo"},
		Branches:     []string{"main", "develop"},
	}
	processor.AddFilter("test-filter", filter)
	
	tests := []struct {
		name      string
		event     notify.Event
		shouldPass bool
	}{
		{
			name: "matching event",
			event: notify.Event{
				Type:       notify.EventCoverageThreshold,
				Severity:   notify.SeverityWarning,
				Repository: "test/repo",
				Branch:     "main",
			},
			shouldPass: true,
		},
		{
			name: "wrong event type",
			event: notify.Event{
				Type:       notify.EventCoverageImprovement,
				Severity:   notify.SeverityWarning,
				Repository: "test/repo",
				Branch:     "main",
			},
			shouldPass: false,
		},
		{
			name: "severity too low",
			event: notify.Event{
				Type:       notify.EventCoverageThreshold,
				Severity:   notify.SeverityInfo,
				Repository: "test/repo",
				Branch:     "main",
			},
			shouldPass: false,
		},
		{
			name: "wrong repository",
			event: notify.Event{
				Type:       notify.EventCoverageThreshold,
				Severity:   notify.SeverityWarning,
				Repository: "other/repo",
				Branch:     "main",
			},
			shouldPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subscriber.Reset()
			
			processor.PublishEvent(context.Background(), tt.event)
			time.Sleep(time.Millisecond * 100)
			
			receivedEvents := subscriber.GetReceivedEvents()
			if tt.shouldPass {
				if len(receivedEvents) == 0 {
					t.Error("Expected event to pass filter")
				}
			} else {
				if len(receivedEvents) > 0 {
					t.Error("Expected event to be filtered out")
				}
			}
		})
	}
}

func TestEventProcessorAggregation(t *testing.T) {
	config := EventProcessorConfig{
		BufferSize:         10,
		WorkerCount:        1,
		BatchTimeout:       time.Millisecond * 100,
		EnableAggregation:  true,
		AggregationWindow:  time.Millisecond * 200,
		MaxAggregateSize:   3,
	}
	
	processor := NewEventProcessor(config)
	processor.Start(context.Background())
	defer processor.Stop()
	
	subscriber := &mockSubscriber{}
	processor.Subscribe(notify.EventCoverageThreshold, subscriber)
	
	// Publish similar events that should be aggregated
	baseEvent := notify.Event{
		Type:       notify.EventCoverageThreshold,
		Repository: "test/repo",
		Branch:     "main",
		Timestamp:  time.Now(),
	}
	
	for i := 0; i < 3; i++ {
		event := baseEvent
		event.ID = fmt.Sprintf("event-%d", i)
		processor.PublishEvent(context.Background(), event)
	}
	
	// Wait for aggregation window
	time.Sleep(time.Millisecond * 250)
	
	receivedEvents := subscriber.GetReceivedEvents()
	
	// Should receive one aggregated event instead of three
	if len(receivedEvents) != 1 {
		t.Errorf("Expected 1 aggregated event, got %d", len(receivedEvents))
	}
	
	if receivedEvents[0].AggregatedCount != 3 {
		t.Errorf("Expected aggregated count of 3, got %d", receivedEvents[0].AggregatedCount)
	}
}

func TestEventProcessorDeduplication(t *testing.T) {
	config := EventProcessorConfig{
		BufferSize:          10,
		WorkerCount:         1,
		BatchTimeout:        time.Millisecond * 50,
		EnableDeduplication: true,
		DeduplicationWindow: time.Millisecond * 200,
	}
	
	processor := NewEventProcessor(config)
	processor.Start(context.Background())
	defer processor.Stop()
	
	subscriber := &mockSubscriber{}
	processor.Subscribe(notify.EventCoverageThreshold, subscriber)
	
	// Publish duplicate events
	event := notify.Event{
		ID:         "duplicate-test",
		Type:       notify.EventCoverageThreshold,
		Repository: "test/repo",
		Branch:     "main",
		Timestamp:  time.Now(),
	}
	
	// Publish same event multiple times
	for i := 0; i < 3; i++ {
		processor.PublishEvent(context.Background(), event)
	}
	
	time.Sleep(time.Millisecond * 100)
	
	receivedEvents := subscriber.GetReceivedEvents()
	
	// Should only receive one event due to deduplication
	if len(receivedEvents) != 1 {
		t.Errorf("Expected 1 event after deduplication, got %d", len(receivedEvents))
	}
}

func TestEventProcessorRetry(t *testing.T) {
	config := EventProcessorConfig{
		BufferSize:    10,
		WorkerCount:   1,
		BatchTimeout:  time.Millisecond * 50,
		EnableRetries: true,
		MaxRetries:    2,
		RetryBackoff:  time.Millisecond * 10,
	}
	
	processor := NewEventProcessor(config)
	processor.Start(context.Background())
	defer processor.Stop()
	
	// Create failing subscriber
	subscriber := &mockSubscriber{shouldFail: true}
	processor.Subscribe(notify.EventCoverageThreshold, subscriber)
	
	event := notify.Event{
		ID:   "retry-test",
		Type: notify.EventCoverageThreshold,
	}
	
	processor.PublishEvent(context.Background(), event)
	
	// Wait for retries to complete
	time.Sleep(time.Millisecond * 200)
	
	// Check metrics for retry attempts
	metrics := processor.GetMetrics()
	if metrics.TotalRetries == 0 {
		t.Error("Expected retry attempts")
	}
}

func TestEventProcessorBatching(t *testing.T) {
	config := EventProcessorConfig{
		BufferSize:   20,
		BatchSize:    3,
		BatchTimeout: time.Millisecond * 200,
		WorkerCount:  1,
	}
	
	processor := NewEventProcessor(config)
	processor.Start(context.Background())
	defer processor.Stop()
	
	subscriber := &mockSubscriber{}
	processor.Subscribe(notify.EventCoverageThreshold, subscriber)
	
	// Publish events to trigger batching
	for i := 0; i < 5; i++ {
		event := notify.Event{
			ID:   fmt.Sprintf("batch-%d", i),
			Type: notify.EventCoverageThreshold,
		}
		processor.PublishEvent(context.Background(), event)
	}
	
	// Wait for batch processing
	time.Sleep(time.Millisecond * 300)
	
	receivedEvents := subscriber.GetReceivedEvents()
	if len(receivedEvents) != 5 {
		t.Errorf("Expected 5 events, got %d", len(receivedEvents))
	}
}

func TestEventProcessorMetrics(t *testing.T) {
	config := EventProcessorConfig{
		BufferSize:   10,
		WorkerCount:  1,
		BatchTimeout: time.Millisecond * 50,
	}
	
	processor := NewEventProcessor(config)
	processor.Start(context.Background())
	defer processor.Stop()
	
	subscriber := &mockSubscriber{}
	processor.Subscribe(notify.EventCoverageThreshold, subscriber)
	
	// Publish some events
	for i := 0; i < 3; i++ {
		event := notify.Event{
			ID:   fmt.Sprintf("metrics-%d", i),
			Type: notify.EventCoverageThreshold,
		}
		processor.PublishEvent(context.Background(), event)
	}
	
	time.Sleep(time.Millisecond * 100)
	
	metrics := processor.GetMetrics()
	
	if metrics.TotalEvents != 3 {
		t.Errorf("Expected 3 total events, got %d", metrics.TotalEvents)
	}
	
	if metrics.ProcessedEvents != 3 {
		t.Errorf("Expected 3 processed events, got %d", metrics.ProcessedEvents)
	}
	
	if metrics.ActiveSubscribers == 0 {
		t.Error("Expected active subscribers")
	}
}

func TestEventProcessorConcurrency(t *testing.T) {
	config := EventProcessorConfig{
		BufferSize:   100,
		WorkerCount:  3,
		BatchTimeout: time.Millisecond * 50,
	}
	
	processor := NewEventProcessor(config)
	processor.Start(context.Background())
	defer processor.Stop()
	
	subscriber := &mockSubscriber{}
	processor.Subscribe(notify.EventCoverageThreshold, subscriber)
	
	// Publish events concurrently
	var wg sync.WaitGroup
	eventCount := 50
	
	for i := 0; i < eventCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			event := notify.Event{
				ID:   fmt.Sprintf("concurrent-%d", id),
				Type: notify.EventCoverageThreshold,
			}
			processor.PublishEvent(context.Background(), event)
		}(i)
	}
	
	wg.Wait()
	time.Sleep(time.Millisecond * 200)
	
	receivedEvents := subscriber.GetReceivedEvents()
	if len(receivedEvents) != eventCount {
		t.Errorf("Expected %d events, got %d", eventCount, len(receivedEvents))
	}
}

func BenchmarkEventProcessorPublish(b *testing.B) {
	config := EventProcessorConfig{
		BufferSize:   1000,
		WorkerCount:  2,
		BatchTimeout: time.Millisecond * 10,
	}
	
	processor := NewEventProcessor(config)
	processor.Start(context.Background())
	defer processor.Stop()
	
	subscriber := &mockSubscriber{}
	processor.Subscribe(notify.EventCoverageThreshold, subscriber)
	
	event := notify.Event{
		ID:   "benchmark",
		Type: notify.EventCoverageThreshold,
		Data: map[string]interface{}{
			"coverage": 85.0,
		},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processor.PublishEvent(context.Background(), event)
	}
}

func BenchmarkEventProcessorWithFiltering(b *testing.B) {
	config := EventProcessorConfig{
		BufferSize:   1000,
		WorkerCount:  2,
		BatchTimeout: time.Millisecond * 10,
	}
	
	processor := NewEventProcessor(config)
	processor.Start(context.Background())
	defer processor.Stop()
	
	subscriber := &mockSubscriber{}
	processor.Subscribe(notify.EventCoverageThreshold, subscriber)
	
	// Add complex filter
	filter := EventFilter{
		EventTypes:   []notify.EventType{notify.EventCoverageThreshold, notify.EventCoverageRegression},
		MinSeverity:  notify.SeverityWarning,
		Repositories: []string{"repo1", "repo2", "repo3"},
		Branches:     []string{"main", "develop", "staging"},
	}
	processor.AddFilter("benchmark-filter", filter)
	
	event := notify.Event{
		ID:         "benchmark-filtered",
		Type:       notify.EventCoverageThreshold,
		Severity:   notify.SeverityWarning,
		Repository: "repo1",
		Branch:     "main",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processor.PublishEvent(context.Background(), event)
	}
}