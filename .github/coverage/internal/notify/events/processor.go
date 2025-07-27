// Package events provides event processing capabilities for the notification system
package events

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/mrz1836/go-broadcast/coverage/internal/notify"
)

// EventProcessor manages the processing and routing of coverage events
type EventProcessor struct {
	config         *ProcessorConfig
	channels       map[notify.ChannelType]notify.NotificationChannel
	eventHistory   *EventHistory
	filters        []EventFilter
	aggregators    map[string]*EventAggregator
	subscribers    []EventSubscriber
	mu             sync.RWMutex
	stopCh         chan struct{}
	eventCh        chan *CoverageEvent
}

// ProcessorConfig holds configuration for the event processor
type ProcessorConfig struct {
	// Event processing settings
	MaxEventHistory      int           `json:"max_event_history"`
	EventRetention       time.Duration `json:"event_retention"`
	DeduplicationWindow  time.Duration `json:"deduplication_window"`
	
	// Aggregation settings
	AggregationEnabled   bool          `json:"aggregation_enabled"`
	AggregationWindow    time.Duration `json:"aggregation_window"`
	MinEventsForBatch    int           `json:"min_events_for_batch"`
	MaxEventsPerBatch    int           `json:"max_events_per_batch"`
	
	// Processing limits
	MaxConcurrentEvents  int           `json:"max_concurrent_events"`
	EventTimeout         time.Duration `json:"event_timeout"`
	RetryAttempts        int           `json:"retry_attempts"`
	RetryDelay           time.Duration `json:"retry_delay"`
	
	// Filtering settings
	DefaultFilters       []FilterRule  `json:"default_filters"`
	EnableRateLimiting   bool          `json:"enable_rate_limiting"`
	RateLimitPerHour     int           `json:"rate_limit_per_hour"`
}

// CoverageEvent represents a coverage-related event
type CoverageEvent struct {
	// Event metadata
	ID           string            `json:"id"`
	Type         notify.EventType  `json:"type"`
	Timestamp    time.Time         `json:"timestamp"`
	Source       string            `json:"source"`
	
	// Context information
	Repository   string            `json:"repository"`
	Branch       string            `json:"branch"`
	CommitSHA    string            `json:"commit_sha,omitempty"`
	PRNumber     int               `json:"pr_number,omitempty"`
	Author       string            `json:"author,omitempty"`
	
	// Coverage data
	CoverageData *CoverageEventData `json:"coverage_data,omitempty"`
	
	// Event-specific data
	EventData    map[string]interface{} `json:"event_data,omitempty"`
	
	// Processing metadata
	Priority     notify.Priority   `json:"priority"`
	Severity     notify.SeverityLevel `json:"severity"`
	Tags         []string          `json:"tags,omitempty"`
	
	// Routing information
	TargetChannels []notify.ChannelType `json:"target_channels,omitempty"`
	ExcludeChannels []notify.ChannelType `json:"exclude_channels,omitempty"`
	
	// Processing state
	ProcessedAt  *time.Time        `json:"processed_at,omitempty"`
	Attempts     int               `json:"attempts"`
	LastError    string            `json:"last_error,omitempty"`
}

// CoverageEventData represents coverage-specific event data
type CoverageEventData struct {
	Current        float64           `json:"current"`
	Previous       float64           `json:"previous"`
	Change         float64           `json:"change"`
	Target         float64           `json:"target"`
	Threshold      float64           `json:"threshold"`
	LinesTotal     int               `json:"lines_total"`
	LinesCovered   int               `json:"lines_covered"`
	BranchCoverage float64           `json:"branch_coverage"`
	FunctionCoverage float64         `json:"function_coverage"`
	TestCount      int               `json:"test_count"`
	TestsPassed    int               `json:"tests_passed"`
	QualityGates   *QualityGateData  `json:"quality_gates,omitempty"`
	TrendData      *TrendEventData   `json:"trend_data,omitempty"`
}

// QualityGateData represents quality gate information
type QualityGateData struct {
	Passed       bool     `json:"passed"`
	TotalGates   int      `json:"total_gates"`
	PassedGates  int      `json:"passed_gates"`
	FailedGates  []string `json:"failed_gates"`
	Score        float64  `json:"score"`
}

// TrendEventData represents trend information
type TrendEventData struct {
	Direction    string  `json:"direction"`
	Magnitude    string  `json:"magnitude"`
	Confidence   float64 `json:"confidence"`
	Volatility   string  `json:"volatility"`
	Prediction   float64 `json:"prediction,omitempty"`
}

// EventHistory manages the history of processed events
type EventHistory struct {
	events      []CoverageEvent   `json:"events"`
	maxSize     int               `json:"max_size"`
	retention   time.Duration     `json:"retention"`
	mu          sync.RWMutex
}

// EventFilter represents a filter for events
type EventFilter interface {
	ShouldProcess(event *CoverageEvent) bool
	GetName() string
	GetDescription() string
}

// FilterRule represents a configuration-based filter rule
type FilterRule struct {
	Name           string                 `json:"name"`
	Type           FilterType             `json:"type"`
	Condition      FilterCondition        `json:"condition"`
	Value          interface{}            `json:"value"`
	Action         FilterAction           `json:"action"`
	Channels       []notify.ChannelType   `json:"channels,omitempty"`
}

// FilterType defines types of filters
type FilterType string

const (
	FilterTypeEventType     FilterType = "event_type"
	FilterTypeSeverity      FilterType = "severity"
	FilterTypePriority      FilterType = "priority"
	FilterTypeRepository    FilterType = "repository"
	FilterTypeBranch        FilterType = "branch"
	FilterTypeAuthor        FilterType = "author"
	FilterTypeCoverage      FilterType = "coverage"
	FilterTypeChange        FilterType = "change"
	FilterTypeTimeOfDay     FilterType = "time_of_day"
	FilterTypeDayOfWeek     FilterType = "day_of_week"
	FilterTypeTag           FilterType = "tag"
)

// FilterCondition defines filter conditions
type FilterCondition string

const (
	ConditionEquals           FilterCondition = "equals"
	ConditionNotEquals        FilterCondition = "not_equals"
	ConditionGreaterThan      FilterCondition = "greater_than"
	ConditionLessThan         FilterCondition = "less_than"
	ConditionContains         FilterCondition = "contains"
	ConditionNotContains      FilterCondition = "not_contains"
	ConditionStartsWith       FilterCondition = "starts_with"
	ConditionEndsWith         FilterCondition = "ends_with"
	ConditionInList           FilterCondition = "in_list"
	ConditionNotInList        FilterCondition = "not_in_list"
	ConditionMatches          FilterCondition = "matches"
	ConditionBetween          FilterCondition = "between"
)

// FilterAction defines actions to take when filter matches
type FilterAction string

const (
	ActionAllow    FilterAction = "allow"
	ActionDeny     FilterAction = "deny"
	ActionRoute    FilterAction = "route"
	ActionModify   FilterAction = "modify"
	ActionDelay    FilterAction = "delay"
	ActionAggregate FilterAction = "aggregate"
)

// EventAggregator manages aggregation of related events
type EventAggregator struct {
	Name           string              `json:"name"`
	Window         time.Duration       `json:"window"`
	MinEvents      int                 `json:"min_events"`
	MaxEvents      int                 `json:"max_events"`
	GroupBy        []string            `json:"group_by"`
	AggregateFunc  AggregationFunction `json:"aggregate_func"`
	events         []CoverageEvent     `json:"events"`
	lastFlush      time.Time           `json:"last_flush"`
	mu             sync.Mutex
}

// AggregationFunction defines how to aggregate events
type AggregationFunction string

const (
	AggregateSum     AggregationFunction = "sum"
	AggregateAverage AggregationFunction = "average"
	AggregateCount   AggregationFunction = "count"
	AggregateMax     AggregationFunction = "max"
	AggregateMin     AggregationFunction = "min"
	AggregateLatest  AggregationFunction = "latest"
	AggregateCustom  AggregationFunction = "custom"
)

// EventSubscriber represents a subscriber to events
type EventSubscriber interface {
	OnEvent(event *CoverageEvent) error
	GetSubscriberInfo() SubscriberInfo
}

// SubscriberInfo provides information about an event subscriber
type SubscriberInfo struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	EventTypes  []notify.EventType  `json:"event_types"`
	Channels    []notify.ChannelType `json:"channels"`
}

// NewEventProcessor creates a new event processor
func NewEventProcessor(config *ProcessorConfig) *EventProcessor {
	if config == nil {
		config = &ProcessorConfig{
			MaxEventHistory:      1000,
			EventRetention:       24 * time.Hour,
			DeduplicationWindow:  5 * time.Minute,
			AggregationEnabled:   true,
			AggregationWindow:    10 * time.Minute,
			MinEventsForBatch:    3,
			MaxEventsPerBatch:    50,
			MaxConcurrentEvents:  10,
			EventTimeout:         30 * time.Second,
			RetryAttempts:        3,
			RetryDelay:           time.Minute,
			EnableRateLimiting:   true,
			RateLimitPerHour:     100,
		}
	}
	
	processor := &EventProcessor{
		config:      config,
		channels:    make(map[notify.ChannelType]notify.NotificationChannel),
		eventHistory: &EventHistory{
			events:    make([]CoverageEvent, 0),
			maxSize:   config.MaxEventHistory,
			retention: config.EventRetention,
		},
		filters:     make([]EventFilter, 0),
		aggregators: make(map[string]*EventAggregator),
		subscribers: make([]EventSubscriber, 0),
		stopCh:      make(chan struct{}),
		eventCh:     make(chan *CoverageEvent, 100),
	}
	
	// Initialize default filters
	processor.initializeDefaultFilters()
	
	// Initialize default aggregators
	processor.initializeDefaultAggregators()
	
	// Start event processing goroutine
	go processor.processEvents()
	
	// Start periodic cleanup goroutine
	go processor.periodicCleanup()
	
	return processor
}

// ProcessEvent processes a coverage event
func (p *EventProcessor) ProcessEvent(ctx context.Context, event *CoverageEvent) error {
	// Set processing timestamp
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	
	// Generate ID if not set
	if event.ID == "" {
		event.ID = p.generateEventID(event)
	}
	
	// Check for duplicate events
	if p.isDuplicateEvent(event) {
		return fmt.Errorf("duplicate event detected: %s", event.ID)
	}
	
	// Apply filters
	if !p.shouldProcessEvent(event) {
		return fmt.Errorf("event filtered out: %s", event.ID)
	}
	
	// Add to processing queue
	select {
	case p.eventCh <- event:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return fmt.Errorf("event queue full")
	}
}

// AddChannel adds a notification channel
func (p *EventProcessor) AddChannel(channel notify.NotificationChannel) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.channels[channel.GetChannelType()] = channel
}

// AddFilter adds an event filter
func (p *EventProcessor) AddFilter(filter EventFilter) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.filters = append(p.filters, filter)
}

// AddSubscriber adds an event subscriber
func (p *EventProcessor) AddSubscriber(subscriber EventSubscriber) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.subscribers = append(p.subscribers, subscriber)
}

// GetEventHistory returns the event history
func (p *EventProcessor) GetEventHistory() []CoverageEvent {
	p.eventHistory.mu.RLock()
	defer p.eventHistory.mu.RUnlock()
	
	// Return a copy of events
	events := make([]CoverageEvent, len(p.eventHistory.events))
	copy(events, p.eventHistory.events)
	return events
}

// GetEventStats returns statistics about event processing
func (p *EventProcessor) GetEventStats() EventStats {
	p.eventHistory.mu.RLock()
	defer p.eventHistory.mu.RUnlock()
	
	stats := EventStats{
		TotalEvents:     len(p.eventHistory.events),
		EventsByType:    make(map[notify.EventType]int),
		EventsBySeverity: make(map[notify.SeverityLevel]int),
		EventsByChannel: make(map[notify.ChannelType]int),
	}
	
	for _, event := range p.eventHistory.events {
		stats.EventsByType[event.Type]++
		stats.EventsBySeverity[event.Severity]++
		
		for _, channel := range event.TargetChannels {
			stats.EventsByChannel[channel]++
		}
	}
	
	return stats
}

// EventStats represents event processing statistics
type EventStats struct {
	TotalEvents      int                               `json:"total_events"`
	EventsByType     map[notify.EventType]int         `json:"events_by_type"`
	EventsBySeverity map[notify.SeverityLevel]int     `json:"events_by_severity"`
	EventsByChannel  map[notify.ChannelType]int       `json:"events_by_channel"`
	LastEventTime    time.Time                        `json:"last_event_time"`
}

// Stop stops the event processor
func (p *EventProcessor) Stop() {
	close(p.stopCh)
}

// Internal methods

func (p *EventProcessor) processEvents() {
	for {
		select {
		case event := <-p.eventCh:
			p.handleEvent(event)
		case <-p.stopCh:
			return
		}
	}
}

func (p *EventProcessor) handleEvent(event *CoverageEvent) {
	ctx, cancel := context.WithTimeout(context.Background(), p.config.EventTimeout)
	defer cancel()
	
	// Determine target channels
	channels := p.determineTargetChannels(event)
	
	// Check aggregation
	if p.config.AggregationEnabled {
		if p.shouldAggregateEvent(event) {
			p.addToAggregation(event)
			return
		}
	}
	
	// Convert to notification
	notification := p.convertToNotification(event)
	notification.Channels = channels
	
	// Notify subscribers
	p.notifySubscribers(event)
	
	// Send notifications
	p.sendNotifications(ctx, notification)
	
	// Add to history
	p.addToHistory(event)
}

func (p *EventProcessor) determineTargetChannels(event *CoverageEvent) []notify.ChannelType {
	// Start with specified target channels
	channels := make([]notify.ChannelType, 0)
	if len(event.TargetChannels) > 0 {
		channels = append(channels, event.TargetChannels...)
	} else {
		// Use all available channels if none specified
		p.mu.RLock()
		for channelType := range p.channels {
			channels = append(channels, channelType)
		}
		p.mu.RUnlock()
	}
	
	// Remove excluded channels
	if len(event.ExcludeChannels) > 0 {
		filtered := make([]notify.ChannelType, 0)
		for _, channel := range channels {
			excluded := false
			for _, excludeChannel := range event.ExcludeChannels {
				if channel == excludeChannel {
					excluded = true
					break
				}
			}
			if !excluded {
				filtered = append(filtered, channel)
			}
		}
		channels = filtered
	}
	
	return channels
}

func (p *EventProcessor) convertToNotification(event *CoverageEvent) *notify.Notification {
	notification := &notify.Notification{
		ID:        event.ID,
		Timestamp: event.Timestamp,
		EventType: event.Type,
		Severity:  event.Severity,
		Priority:  event.Priority,
		Repository: event.Repository,
		Branch:    event.Branch,
		CommitSHA: event.CommitSHA,
		PRNumber:  event.PRNumber,
		Author:    event.Author,
	}
	
	// Set subject and message based on event type
	notification.Subject, notification.Message = p.generateNotificationContent(event)
	
	// Convert coverage data
	if event.CoverageData != nil {
		notification.CoverageData = &notify.CoverageData{
			Current:   event.CoverageData.Current,
			Previous:  event.CoverageData.Previous,
			Change:    event.CoverageData.Change,
			Target:    event.CoverageData.Target,
			Threshold: event.CoverageData.Threshold,
		}
	}
	
	// Convert trend data
	if event.CoverageData != nil && event.CoverageData.TrendData != nil {
		notification.TrendData = &notify.TrendData{
			Direction:  event.CoverageData.TrendData.Direction,
			Magnitude:  event.CoverageData.TrendData.Magnitude,
			Confidence: event.CoverageData.TrendData.Confidence,
			Prediction: event.CoverageData.TrendData.Prediction,
		}
	}
	
	return notification
}

func (p *EventProcessor) generateNotificationContent(event *CoverageEvent) (string, string) {
	switch event.Type {
	case notify.EventCoverageThreshold:
		return p.generateThresholdContent(event)
	case notify.EventCoverageRegression:
		return p.generateRegressionContent(event)
	case notify.EventCoverageImprovement:
		return p.generateImprovementContent(event)
	case notify.EventMilestoneReached:
		return p.generateMilestoneContent(event)
	case notify.EventTrendAlert:
		return p.generateTrendAlertContent(event)
	case notify.EventPredictionAlert:
		return p.generatePredictionAlertContent(event)
	case notify.EventQualityAlert:
		return p.generateQualityAlertContent(event)
	default:
		return p.generateGenericContent(event)
	}
}

func (p *EventProcessor) generateThresholdContent(event *CoverageEvent) (string, string) {
	if event.CoverageData == nil {
		return "Coverage Threshold Event", "Coverage threshold event occurred"
	}
	
	if event.CoverageData.Current < event.CoverageData.Threshold {
		return fmt.Sprintf("Coverage Below Threshold: %.1f%%", event.CoverageData.Current),
			fmt.Sprintf("Coverage in %s is %.1f%%, which is below the threshold of %.1f%%",
				event.Repository, event.CoverageData.Current, event.CoverageData.Threshold)
	} else {
		return fmt.Sprintf("Coverage Above Threshold: %.1f%%", event.CoverageData.Current),
			fmt.Sprintf("Coverage in %s is %.1f%%, which meets the threshold of %.1f%%",
				event.Repository, event.CoverageData.Current, event.CoverageData.Threshold)
	}
}

func (p *EventProcessor) generateRegressionContent(event *CoverageEvent) (string, string) {
	if event.CoverageData == nil {
		return "Coverage Regression", "Coverage has decreased"
	}
	
	return fmt.Sprintf("Coverage Regression: %.1f%% → %.1f%%", event.CoverageData.Previous, event.CoverageData.Current),
		fmt.Sprintf("Coverage in %s has decreased by %.1f%% (from %.1f%% to %.1f%%)",
			event.Repository, math.Abs(event.CoverageData.Change), event.CoverageData.Previous, event.CoverageData.Current)
}

func (p *EventProcessor) generateImprovementContent(event *CoverageEvent) (string, string) {
	if event.CoverageData == nil {
		return "Coverage Improvement", "Coverage has increased"
	}
	
	return fmt.Sprintf("Coverage Improvement: %.1f%% → %.1f%%", event.CoverageData.Previous, event.CoverageData.Current),
		fmt.Sprintf("Great news! Coverage in %s has increased by %.1f%% (from %.1f%% to %.1f%%)",
			event.Repository, event.CoverageData.Change, event.CoverageData.Previous, event.CoverageData.Current)
}

func (p *EventProcessor) generateMilestoneContent(event *CoverageEvent) (string, string) {
	if event.CoverageData == nil {
		return "Coverage Milestone", "Coverage milestone reached"
	}
	
	milestone := math.Round(event.CoverageData.Current/10) * 10
	return fmt.Sprintf("Coverage Milestone: %.0f%% Reached!", milestone),
		fmt.Sprintf("Congratulations! %s has reached %.0f%% coverage milestone (current: %.1f%%)",
			event.Repository, milestone, event.CoverageData.Current)
}

func (p *EventProcessor) generateTrendAlertContent(event *CoverageEvent) (string, string) {
	if event.CoverageData == nil || event.CoverageData.TrendData == nil {
		return "Coverage Trend Alert", "Coverage trend alert triggered"
	}
	
	trend := event.CoverageData.TrendData
	return fmt.Sprintf("Coverage Trend Alert: %s Trend", strings.Title(trend.Direction)),
		fmt.Sprintf("Coverage trend in %s is %s with %s magnitude (%.0f%% confidence)",
			event.Repository, trend.Direction, trend.Magnitude, trend.Confidence*100)
}

func (p *EventProcessor) generatePredictionAlertContent(event *CoverageEvent) (string, string) {
	if event.CoverageData == nil || event.CoverageData.TrendData == nil {
		return "Coverage Prediction Alert", "Coverage prediction alert triggered"
	}
	
	prediction := event.CoverageData.TrendData.Prediction
	return fmt.Sprintf("Coverage Prediction Alert: %.1f%% Predicted", prediction),
		fmt.Sprintf("Based on current trends, coverage in %s is predicted to reach %.1f%% (%.0f%% confidence)",
			event.Repository, prediction, event.CoverageData.TrendData.Confidence*100)
}

func (p *EventProcessor) generateQualityAlertContent(event *CoverageEvent) (string, string) {
	if event.CoverageData == nil || event.CoverageData.QualityGates == nil {
		return "Quality Alert", "Quality gate alert triggered"
	}
	
	gates := event.CoverageData.QualityGates
	if gates.Passed {
		return "Quality Gates Passed",
			fmt.Sprintf("All quality gates passed in %s (%d/%d gates, score: %.1f)",
				event.Repository, gates.PassedGates, gates.TotalGates, gates.Score)
	} else {
		return "Quality Gates Failed",
			fmt.Sprintf("Quality gates failed in %s (%d/%d gates passed, failed: %s)",
				event.Repository, gates.PassedGates, gates.TotalGates, strings.Join(gates.FailedGates, ", "))
	}
}

func (p *EventProcessor) generateGenericContent(event *CoverageEvent) (string, string) {
	return fmt.Sprintf("Coverage Event: %s", event.Type),
		fmt.Sprintf("Coverage event of type %s occurred in %s", event.Type, event.Repository)
}

func (p *EventProcessor) notifySubscribers(event *CoverageEvent) {
	p.mu.RLock()
	subscribers := make([]EventSubscriber, len(p.subscribers))
	copy(subscribers, p.subscribers)
	p.mu.RUnlock()
	
	for _, subscriber := range subscribers {
		go func(s EventSubscriber) {
			if err := s.OnEvent(event); err != nil {
				// Log error (in real implementation)
				fmt.Printf("Subscriber error: %v\n", err)
			}
		}(subscriber)
	}
}

func (p *EventProcessor) sendNotifications(ctx context.Context, notification *notify.Notification) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	for _, channelType := range notification.Channels {
		if channel, exists := p.channels[channelType]; exists {
			go func(ch notify.NotificationChannel) {
				_, err := ch.Send(ctx, notification)
				if err != nil {
					// Log error (in real implementation)
					fmt.Printf("Channel send error: %v\n", err)
				}
			}(channel)
		}
	}
}

func (p *EventProcessor) addToHistory(event *CoverageEvent) {
	p.eventHistory.mu.Lock()
	defer p.eventHistory.mu.Unlock()
	
	// Mark as processed
	now := time.Now()
	event.ProcessedAt = &now
	
	// Add to history
	p.eventHistory.events = append(p.eventHistory.events, *event)
	
	// Trim history if needed
	if len(p.eventHistory.events) > p.eventHistory.maxSize {
		p.eventHistory.events = p.eventHistory.events[1:]
	}
}

func (p *EventProcessor) shouldProcessEvent(event *CoverageEvent) bool {
	p.mu.RLock()
	filters := make([]EventFilter, len(p.filters))
	copy(filters, p.filters)
	p.mu.RUnlock()
	
	for _, filter := range filters {
		if !filter.ShouldProcess(event) {
			return false
		}
	}
	
	return true
}

func (p *EventProcessor) isDuplicateEvent(event *CoverageEvent) bool {
	p.eventHistory.mu.RLock()
	defer p.eventHistory.mu.RUnlock()
	
	cutoff := time.Now().Add(-p.config.DeduplicationWindow)
	
	for _, historicalEvent := range p.eventHistory.events {
		if historicalEvent.Timestamp.Before(cutoff) {
			continue
		}
		
		if p.eventsAreDuplicate(event, &historicalEvent) {
			return true
		}
	}
	
	return false
}

func (p *EventProcessor) eventsAreDuplicate(event1, event2 *CoverageEvent) bool {
	return event1.Type == event2.Type &&
		event1.Repository == event2.Repository &&
		event1.Branch == event2.Branch &&
		event1.CommitSHA == event2.CommitSHA &&
		event1.PRNumber == event2.PRNumber
}

func (p *EventProcessor) shouldAggregateEvent(event *CoverageEvent) bool {
	// Simple aggregation logic - could be made more sophisticated
	return event.Type == notify.EventCoverageThreshold ||
		event.Type == notify.EventCoverageRegression
}

func (p *EventProcessor) addToAggregation(event *CoverageEvent) {
	aggregatorName := string(event.Type)
	
	p.mu.Lock()
	aggregator, exists := p.aggregators[aggregatorName]
	if !exists {
		aggregator = &EventAggregator{
			Name:      aggregatorName,
			Window:    p.config.AggregationWindow,
			MinEvents: p.config.MinEventsForBatch,
			MaxEvents: p.config.MaxEventsPerBatch,
			events:    make([]CoverageEvent, 0),
			lastFlush: time.Now(),
		}
		p.aggregators[aggregatorName] = aggregator
	}
	p.mu.Unlock()
	
	aggregator.mu.Lock()
	defer aggregator.mu.Unlock()
	
	aggregator.events = append(aggregator.events, *event)
	
	// Check if we should flush
	if len(aggregator.events) >= aggregator.MaxEvents ||
		time.Since(aggregator.lastFlush) >= aggregator.Window {
		p.flushAggregator(aggregator)
	}
}

func (p *EventProcessor) flushAggregator(aggregator *EventAggregator) {
	if len(aggregator.events) < aggregator.MinEvents {
		return
	}
	
	// Create aggregated event
	aggregatedEvent := p.createAggregatedEvent(aggregator)
	
	// Process aggregated event
	go p.handleEvent(aggregatedEvent)
	
	// Clear aggregator
	aggregator.events = make([]CoverageEvent, 0)
	aggregator.lastFlush = time.Now()
}

func (p *EventProcessor) createAggregatedEvent(aggregator *EventAggregator) *CoverageEvent {
	if len(aggregator.events) == 0 {
		return nil
	}
	
	// Use the latest event as base
	baseEvent := aggregator.events[len(aggregator.events)-1]
	
	// Create aggregated event
	aggregatedEvent := &CoverageEvent{
		ID:        p.generateEventID(&baseEvent) + "_aggregated",
		Type:      baseEvent.Type,
		Timestamp: time.Now(),
		Source:    "event_aggregator",
		Repository: baseEvent.Repository,
		Branch:    baseEvent.Branch,
		Priority:  baseEvent.Priority,
		Severity:  baseEvent.Severity,
		EventData: map[string]interface{}{
			"aggregated_count": len(aggregator.events),
			"aggregation_window": aggregator.Window.String(),
			"original_events": len(aggregator.events),
		},
	}
	
	// Aggregate coverage data
	if len(aggregator.events) > 0 && aggregator.events[0].CoverageData != nil {
		aggregatedEvent.CoverageData = p.aggregateCoverageData(aggregator.events)
	}
	
	return aggregatedEvent
}

func (p *EventProcessor) aggregateCoverageData(events []CoverageEvent) *CoverageEventData {
	if len(events) == 0 {
		return nil
	}
	
	// Simple aggregation - use latest values for most fields
	latest := events[len(events)-1].CoverageData
	
	return &CoverageEventData{
		Current:        latest.Current,
		Previous:       events[0].CoverageData.Current, // Use first event as "previous"
		Change:         latest.Current - events[0].CoverageData.Current,
		Target:         latest.Target,
		Threshold:      latest.Threshold,
		LinesTotal:     latest.LinesTotal,
		LinesCovered:   latest.LinesCovered,
		BranchCoverage: latest.BranchCoverage,
		FunctionCoverage: latest.FunctionCoverage,
		TestCount:      latest.TestCount,
		TestsPassed:    latest.TestsPassed,
		QualityGates:   latest.QualityGates,
		TrendData:      latest.TrendData,
	}
}

func (p *EventProcessor) generateEventID(event *CoverageEvent) string {
	return fmt.Sprintf("%s_%s_%s_%d", 
		event.Type, event.Repository, event.Branch, event.Timestamp.Unix())
}

func (p *EventProcessor) initializeDefaultFilters() {
	// Add default filters based on configuration
	for _, rule := range p.config.DefaultFilters {
		filter := NewConfigFilter(rule)
		p.filters = append(p.filters, filter)
	}
}

func (p *EventProcessor) initializeDefaultAggregators() {
	// Initialize default aggregators for common event types
	if p.config.AggregationEnabled {
		p.aggregators["coverage_threshold"] = &EventAggregator{
			Name:      "coverage_threshold",
			Window:    p.config.AggregationWindow,
			MinEvents: p.config.MinEventsForBatch,
			MaxEvents: p.config.MaxEventsPerBatch,
			events:    make([]CoverageEvent, 0),
			lastFlush: time.Now(),
		}
	}
}

func (p *EventProcessor) periodicCleanup() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			p.cleanupHistory()
			p.flushPendingAggregations()
		case <-p.stopCh:
			return
		}
	}
}

func (p *EventProcessor) cleanupHistory() {
	p.eventHistory.mu.Lock()
	defer p.eventHistory.mu.Unlock()
	
	cutoff := time.Now().Add(-p.eventHistory.retention)
	filtered := make([]CoverageEvent, 0)
	
	for _, event := range p.eventHistory.events {
		if event.Timestamp.After(cutoff) {
			filtered = append(filtered, event)
		}
	}
	
	p.eventHistory.events = filtered
}

func (p *EventProcessor) flushPendingAggregations() {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	for _, aggregator := range p.aggregators {
		aggregator.mu.Lock()
		if len(aggregator.events) >= aggregator.MinEvents &&
			time.Since(aggregator.lastFlush) >= aggregator.Window {
			p.flushAggregator(aggregator)
		}
		aggregator.mu.Unlock()
	}
}

// ConfigFilter implements EventFilter based on configuration rules
type ConfigFilter struct {
	rule FilterRule
}

// NewConfigFilter creates a new configuration-based filter
func NewConfigFilter(rule FilterRule) *ConfigFilter {
	return &ConfigFilter{rule: rule}
}

// ShouldProcess implements EventFilter
func (f *ConfigFilter) ShouldProcess(event *CoverageEvent) bool {
	matches := f.evaluateCondition(event)
	
	switch f.rule.Action {
	case ActionAllow:
		return matches
	case ActionDeny:
		return !matches
	default:
		return true
	}
}

// GetName implements EventFilter
func (f *ConfigFilter) GetName() string {
	return f.rule.Name
}

// GetDescription implements EventFilter
func (f *ConfigFilter) GetDescription() string {
	return fmt.Sprintf("Filter based on %s %s %v", f.rule.Type, f.rule.Condition, f.rule.Value)
}

func (f *ConfigFilter) evaluateCondition(event *CoverageEvent) bool {
	var fieldValue interface{}
	
	// Extract field value based on filter type
	switch f.rule.Type {
	case FilterTypeEventType:
		fieldValue = string(event.Type)
	case FilterTypeSeverity:
		fieldValue = string(event.Severity)
	case FilterTypePriority:
		fieldValue = string(event.Priority)
	case FilterTypeRepository:
		fieldValue = event.Repository
	case FilterTypeBranch:
		fieldValue = event.Branch
	case FilterTypeAuthor:
		fieldValue = event.Author
	case FilterTypeCoverage:
		if event.CoverageData != nil {
			fieldValue = event.CoverageData.Current
		}
	case FilterTypeChange:
		if event.CoverageData != nil {
			fieldValue = event.CoverageData.Change
		}
	default:
		return true
	}
	
	// Evaluate condition
	return f.evaluateConditionValue(fieldValue, f.rule.Condition, f.rule.Value)
}

func (f *ConfigFilter) evaluateConditionValue(fieldValue interface{}, condition FilterCondition, expectedValue interface{}) bool {
	switch condition {
	case ConditionEquals:
		return fieldValue == expectedValue
	case ConditionNotEquals:
		return fieldValue != expectedValue
	case ConditionGreaterThan:
		if fv, ok := fieldValue.(float64); ok {
			if ev, ok := expectedValue.(float64); ok {
				return fv > ev
			}
		}
	case ConditionLessThan:
		if fv, ok := fieldValue.(float64); ok {
			if ev, ok := expectedValue.(float64); ok {
				return fv < ev
			}
		}
	case ConditionContains:
		if fv, ok := fieldValue.(string); ok {
			if ev, ok := expectedValue.(string); ok {
				return strings.Contains(fv, ev)
			}
		}
	case ConditionStartsWith:
		if fv, ok := fieldValue.(string); ok {
			if ev, ok := expectedValue.(string); ok {
				return strings.HasPrefix(fv, ev)
			}
		}
	case ConditionEndsWith:
		if fv, ok := fieldValue.(string); ok {
			if ev, ok := expectedValue.(string); ok {
				return strings.HasSuffix(fv, ev)
			}
		}
	}
	
	return false
}

// Helper function to create common events
func CreateCoverageEvent(eventType notify.EventType, repository, branch string, coverageData *CoverageEventData) *CoverageEvent {
	return &CoverageEvent{
		Type:         eventType,
		Timestamp:    time.Now(),
		Source:       "coverage_system",
		Repository:   repository,
		Branch:       branch,
		CoverageData: coverageData,
		Priority:     notify.PriorityNormal,
		Severity:     notify.SeverityInfo,
	}
}