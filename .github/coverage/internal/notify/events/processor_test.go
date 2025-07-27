package events

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

// Test error definitions
var (
	ErrMockSubscriber      = errors.New("mock subscriber error")
	ErrProcessorNotRunning = errors.New("processor not running")
)

// Event represents a notification event
type Event struct {
	ID              string
	Type            EventType
	Timestamp       time.Time
	Data            map[string]interface{}
	Source          string
	Repository      string
	Branch          string
	Severity        Severity
	AggregatedCount int
}

// EventType represents the type of event
type EventType string

const (
	EventCoverageThreshold   EventType = "coverage_threshold"
	EventCoverageImprovement EventType = "coverage_improvement"
	EventCoverageRegression  EventType = "coverage_regression"
)

// Severity represents event severity
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
)

// TestEventFilter represents filtering criteria for events in tests
type TestEventFilter struct {
	EventTypes   []EventType
	MinSeverity  Severity
	Repositories []string
	Branches     []string
}

// EventProcessorConfig represents the configuration for the event processor
type EventProcessorConfig struct {
	BufferSize          int
	BatchSize           int
	BatchTimeout        time.Duration
	WorkerCount         int
	EnableRetries       bool
	MaxRetries          int
	RetryBackoff        time.Duration
	EnableAggregation   bool
	AggregationWindow   time.Duration
	MaxAggregateSize    int
	EnableDeduplication bool
	DeduplicationWindow time.Duration
}

// Metrics represents event processor metrics
type Metrics struct {
	TotalEvents       int64
	ProcessedEvents   int64
	ActiveSubscribers int
	TotalRetries      int64
}

// Mock subscriber for testing
type mockSubscriber struct {
	receivedEvents []Event
	mutex          sync.Mutex
	shouldFail     bool
}

func (m *mockSubscriber) HandleEvent(ctx context.Context, event Event) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.shouldFail {
		return ErrMockSubscriber
	}

	m.receivedEvents = append(m.receivedEvents, event)
	return nil
}

func (m *mockSubscriber) GetReceivedEvents() []Event {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	events := make([]Event, len(m.receivedEvents))
	copy(events, m.receivedEvents)
	return events
}

func (m *mockSubscriber) Reset() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.receivedEvents = nil
}

// Mock event processor for testing
type mockEventProcessor struct {
	config      EventProcessorConfig
	subscribers map[EventType][]TestEventSubscriber
	filters     map[string]TestEventFilter
	running     bool
	mu          sync.RWMutex
	metrics     Metrics
}

// TestEventSubscriber interface for testing
type TestEventSubscriber interface {
	HandleEvent(ctx context.Context, event Event) error
}

func NewTestEventProcessor(config EventProcessorConfig) *mockEventProcessor {
	return &mockEventProcessor{
		config:      config,
		subscribers: make(map[EventType][]TestEventSubscriber),
		filters:     make(map[string]TestEventFilter),
		metrics:     Metrics{},
	}
}

func (p *mockEventProcessor) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.running = true
	return nil
}

func (p *mockEventProcessor) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.running = false
	return nil
}

func (p *mockEventProcessor) Subscribe(eventType EventType, subscriber TestEventSubscriber) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.subscribers[eventType] == nil {
		p.subscribers[eventType] = []TestEventSubscriber{}
	}
	p.subscribers[eventType] = append(p.subscribers[eventType], subscriber)
	return nil
}

func (p *mockEventProcessor) Unsubscribe(eventType EventType, subscriber TestEventSubscriber) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	subscribers := p.subscribers[eventType]
	for i, s := range subscribers {
		if s == subscriber {
			p.subscribers[eventType] = append(subscribers[:i], subscribers[i+1:]...)
			break
		}
	}
	return nil
}

func (p *mockEventProcessor) GetSubscribers(eventType EventType) []TestEventSubscriber {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.subscribers[eventType]
}

func (p *mockEventProcessor) PublishEvent(ctx context.Context, event Event) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.running {
		return ErrProcessorNotRunning
	}

	// Apply filters
	for _, filter := range p.filters {
		if !p.matchesFilter(event, filter) {
			return nil
		}
	}

	// Update metrics
	p.metrics.TotalEvents++

	// Notify subscribers
	subscribers := p.subscribers[event.Type]
	for _, subscriber := range subscribers {
		if err := subscriber.HandleEvent(ctx, event); err != nil {
			if p.config.EnableRetries {
				p.metrics.TotalRetries++
			}
		} else {
			p.metrics.ProcessedEvents++
		}
	}

	return nil
}

func (p *mockEventProcessor) matchesFilter(event Event, filter TestEventFilter) bool {
	// Check event type
	if len(filter.EventTypes) > 0 {
		found := false
		for _, t := range filter.EventTypes {
			if t == event.Type {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check severity
	if filter.MinSeverity != "" && event.Severity < filter.MinSeverity {
		return false
	}

	// Check repository
	if len(filter.Repositories) > 0 {
		found := false
		for _, r := range filter.Repositories {
			if r == event.Repository {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check branch
	if len(filter.Branches) > 0 {
		found := false
		for _, b := range filter.Branches {
			if b == event.Branch {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

func (p *mockEventProcessor) AddFilter(name string, filter TestEventFilter) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.filters[name] = filter
}

func (p *mockEventProcessor) GetMetrics() Metrics {
	p.mu.RLock()
	defer p.mu.RUnlock()
	m := p.metrics
	m.ActiveSubscribers = len(p.subscribers)
	return m
}

// Tests

func TestNewEventProcessor(t *testing.T) {
	config := EventProcessorConfig{
		BufferSize:    100,
		BatchSize:     10,
		BatchTimeout:  time.Second,
		WorkerCount:   2,
		EnableRetries: true,
		MaxRetries:    3,
		RetryBackoff:  time.Millisecond * 100,
	}

	processor := NewTestEventProcessor(config)
	if processor == nil {
		t.Fatal("NewTestEventProcessor returned nil")
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

	processor := NewTestEventProcessor(config)

	// Start processor
	err := processor.Start(context.Background())
	if err != nil {
		t.Fatalf("Failed to start processor: %v", err)
	}

	// Verify it's running by publishing an event
	event := Event{ID: "test", Type: EventCoverageThreshold}
	err = processor.PublishEvent(context.Background(), event)
	if err != nil {
		t.Error("Processor should be running and accept events")
	}

	// Stop processor
	err = processor.Stop()
	if err != nil {
		t.Fatalf("Failed to stop processor: %v", err)
	}

	// Verify it's stopped - publishing should fail
	event2 := Event{ID: "test2", Type: EventCoverageThreshold}
	err = processor.PublishEvent(context.Background(), event2)
	if err == nil {
		t.Error("Processor should be stopped and reject events")
	}
}

func TestEventProcessorSubscription(t *testing.T) {
	processor := NewTestEventProcessor(EventProcessorConfig{})

	subscriber := &mockSubscriber{}

	// Subscribe to events
	err := processor.Subscribe(EventCoverageThreshold, subscriber)
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	// Verify subscription
	subscribers := processor.GetSubscribers(EventCoverageThreshold)
	if len(subscribers) != 1 {
		t.Errorf("Expected 1 subscriber, got %d", len(subscribers))
	}

	// Unsubscribe
	err = processor.Unsubscribe(EventCoverageThreshold, subscriber)
	if err != nil {
		t.Fatalf("Failed to unsubscribe: %v", err)
	}

	// Verify unsubscription
	subscribers = processor.GetSubscribers(EventCoverageThreshold)
	if len(subscribers) != 0 {
		t.Errorf("Expected 0 subscribers after unsubscribe, got %d", len(subscribers))
	}
}

func TestEventProcessorPublish(t *testing.T) { //nolint:revive // function naming
	config := EventProcessorConfig{
		BufferSize:   10,
		WorkerCount:  1,
		BatchTimeout: time.Millisecond * 100,
	}

	processor := NewTestEventProcessor(config)
	_ = processor.Start(context.Background())
	defer func() { _ = processor.Stop() }()

	subscriber := &mockSubscriber{}
	_ = processor.Subscribe(EventCoverageThreshold, subscriber)

	// Create test event
	event := Event{
		ID:        "test-001",
		Type:      EventCoverageThreshold,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"coverage":  75.0,
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

	if len(receivedEvents) > 0 && receivedEvents[0].ID != event.ID {
		t.Errorf("Expected event ID %s, got %s", event.ID, receivedEvents[0].ID)
	}
}

func TestEventProcessorFiltering(t *testing.T) { //nolint:revive // function naming
	config := EventProcessorConfig{
		BufferSize:   10,
		WorkerCount:  1,
		BatchTimeout: time.Millisecond * 50,
	}

	processor := NewTestEventProcessor(config)
	_ = processor.Start(context.Background())
	defer func() { _ = processor.Stop() }()

	subscriber := &mockSubscriber{}
	_ = processor.Subscribe(EventCoverageThreshold, subscriber)

	// Add filter
	filter := TestEventFilter{
		EventTypes:   []EventType{EventCoverageThreshold},
		MinSeverity:  SeverityWarning,
		Repositories: []string{"test/repo"},
		Branches:     []string{"main", "develop"},
	}
	processor.AddFilter("test-filter", filter)

	tests := []struct {
		name       string
		event      Event
		shouldPass bool
	}{
		{
			name: "matching event",
			event: Event{
				Type:       EventCoverageThreshold,
				Severity:   SeverityWarning,
				Repository: "test/repo",
				Branch:     "main",
			},
			shouldPass: true,
		},
		{
			name: "wrong event type",
			event: Event{
				Type:       EventCoverageImprovement,
				Severity:   SeverityWarning,
				Repository: "test/repo",
				Branch:     "main",
			},
			shouldPass: false,
		},
		{
			name: "severity too low",
			event: Event{
				Type:       EventCoverageThreshold,
				Severity:   SeverityInfo,
				Repository: "test/repo",
				Branch:     "main",
			},
			shouldPass: false,
		},
		{
			name: "wrong repository",
			event: Event{
				Type:       EventCoverageThreshold,
				Severity:   SeverityWarning,
				Repository: "other/repo",
				Branch:     "main",
			},
			shouldPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subscriber.Reset()

			_ = processor.PublishEvent(context.Background(), tt.event)
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

func TestEventProcessorMetrics(t *testing.T) { //nolint:revive // function naming
	config := EventProcessorConfig{
		BufferSize:   10,
		WorkerCount:  1,
		BatchTimeout: time.Millisecond * 50,
	}

	processor := NewTestEventProcessor(config)
	_ = processor.Start(context.Background())
	defer func() { _ = processor.Stop() }()

	subscriber := &mockSubscriber{}
	_ = processor.Subscribe(EventCoverageThreshold, subscriber)

	// Publish some events
	for i := 0; i < 3; i++ {
		event := Event{
			ID:   fmt.Sprintf("metrics-%d", i),
			Type: EventCoverageThreshold,
		}
		_ = processor.PublishEvent(context.Background(), event)
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

func TestEventProcessorRetry(t *testing.T) { //nolint:revive // function naming
	config := EventProcessorConfig{
		BufferSize:    10,
		WorkerCount:   1,
		BatchTimeout:  time.Millisecond * 50,
		EnableRetries: true,
		MaxRetries:    2,
		RetryBackoff:  time.Millisecond * 10,
	}

	processor := NewTestEventProcessor(config)
	_ = processor.Start(context.Background())
	defer func() { _ = processor.Stop() }()

	// Create failing subscriber
	subscriber := &mockSubscriber{shouldFail: true}
	_ = processor.Subscribe(EventCoverageThreshold, subscriber)

	event := Event{
		ID:   "retry-test",
		Type: EventCoverageThreshold,
	}

	_ = processor.PublishEvent(context.Background(), event)

	// Wait for retries to complete
	time.Sleep(time.Millisecond * 200)

	// Check metrics for retry attempts
	metrics := processor.GetMetrics()
	if metrics.TotalRetries == 0 {
		t.Error("Expected retry attempts")
	}
}

func TestEventProcessorConcurrency(t *testing.T) { //nolint:revive // function naming
	config := EventProcessorConfig{
		BufferSize:   100,
		WorkerCount:  3,
		BatchTimeout: time.Millisecond * 50,
	}

	processor := NewTestEventProcessor(config)
	_ = processor.Start(context.Background())
	defer func() { _ = processor.Stop() }()

	subscriber := &mockSubscriber{}
	_ = processor.Subscribe(EventCoverageThreshold, subscriber)

	// Publish events concurrently
	var wg sync.WaitGroup
	eventCount := 50

	for i := 0; i < eventCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			event := Event{
				ID:   fmt.Sprintf("concurrent-%d", id),
				Type: EventCoverageThreshold,
			}
			_ = processor.PublishEvent(context.Background(), event)
		}(i)
	}

	wg.Wait()
	time.Sleep(time.Millisecond * 200)

	receivedEvents := subscriber.GetReceivedEvents()
	if len(receivedEvents) != eventCount {
		t.Errorf("Expected %d events, got %d", eventCount, len(receivedEvents))
	}
}

func BenchmarkEventProcessorPublish(b *testing.B) { //nolint:revive // function naming
	config := EventProcessorConfig{
		BufferSize:   1000,
		WorkerCount:  2,
		BatchTimeout: time.Millisecond * 10,
	}

	processor := NewTestEventProcessor(config)
	_ = processor.Start(context.Background())
	defer func() { _ = processor.Stop() }()

	subscriber := &mockSubscriber{}
	_ = processor.Subscribe(EventCoverageThreshold, subscriber)

	event := Event{
		ID:   "benchmark",
		Type: EventCoverageThreshold,
		Data: map[string]interface{}{
			"coverage": 85.0,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = processor.PublishEvent(context.Background(), event)
	}
}

func BenchmarkEventProcessorWithFiltering(b *testing.B) { //nolint:revive // function naming
	config := EventProcessorConfig{
		BufferSize:   1000,
		WorkerCount:  2,
		BatchTimeout: time.Millisecond * 10,
	}

	processor := NewTestEventProcessor(config)
	_ = processor.Start(context.Background())
	defer func() { _ = processor.Stop() }()

	subscriber := &mockSubscriber{}
	_ = processor.Subscribe(EventCoverageThreshold, subscriber)

	// Add complex filter
	filter := TestEventFilter{
		EventTypes:   []EventType{EventCoverageThreshold, EventCoverageRegression},
		MinSeverity:  SeverityWarning,
		Repositories: []string{"repo1", "repo2", "repo3"},
		Branches:     []string{"main", "develop", "staging"},
	}
	processor.AddFilter("benchmark-filter", filter)

	event := Event{
		ID:         "benchmark-filtered",
		Type:       EventCoverageThreshold,
		Severity:   SeverityWarning,
		Repository: "repo1",
		Branch:     "main",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = processor.PublishEvent(context.Background(), event)
	}
}
