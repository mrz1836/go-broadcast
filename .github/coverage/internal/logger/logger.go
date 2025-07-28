// Package logger provides a minimal, interface-compatible logging implementation for the coverage system.
//
// This logger is designed to be compatible with logrus.Entry patterns used in the main module,
// allowing easy interface sharing and future integration while maintaining zero external dependencies.
package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// Logger defines the interface compatible with logrus.Entry patterns from the main module
type Logger interface {
	// Field manipulation methods (match logrus.Entry)
	WithField(key string, value interface{}) Logger
	WithFields(fields map[string]interface{}) Logger
	WithError(err error) Logger
	WithContext(ctx context.Context) Logger

	// Logging methods (match logrus.Entry)
	Debug(args ...interface{})
	Info(args ...interface{})
	Warn(args ...interface{})
	Error(args ...interface{})

	// Formatted logging methods (match logrus.Entry)
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

// Level represents log levels compatible with common logging libraries
type Level int

// Log levels compatible with standard logging libraries
const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
)

// String returns the string representation of the log level
func (l Level) String() string {
	switch l {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	default:
		return "INFO"
	}
}

// Config holds logger configuration
type Config struct {
	Level  Level
	Format string // "text" or "json"
	Output io.Writer
}

// entry represents a log entry with accumulated fields
// Context is stored for cancellation support, not for request scoping
type entry struct {
	logger *simpleLogger
	fields map[string]interface{}
	ctx    context.Context //nolint:containedctx // Context needed for cancellation support
	err    error
}

// simpleLogger implements the Logger interface with minimal dependencies
type simpleLogger struct {
	config *Config
}

// NewLogger creates a new logger with the given configuration
func NewLogger(config *Config) Logger {
	if config == nil {
		config = &Config{
			Level:  InfoLevel,
			Format: "text",
			Output: os.Stderr,
		}
	}

	if config.Output == nil {
		config.Output = os.Stderr
	}

	return &simpleLogger{
		config: config,
	}
}

// NewFromEnv creates a logger configured from environment variables
func NewFromEnv() Logger {
	config := &Config{
		Level:  InfoLevel,
		Format: "text",
		Output: os.Stderr,
	}

	// Parse log level from environment
	if levelStr := os.Getenv("COVERAGE_LOG_LEVEL"); levelStr != "" {
		switch strings.ToUpper(levelStr) {
		case "DEBUG":
			config.Level = DebugLevel
		case "INFO":
			config.Level = InfoLevel
		case "WARN":
			config.Level = WarnLevel
		case "ERROR":
			config.Level = ErrorLevel
		}
	}

	// Parse format from environment
	if format := os.Getenv("COVERAGE_LOG_FORMAT"); format != "" {
		if format == "json" || format == "text" {
			config.Format = format
		}
	}

	return NewLogger(config)
}

// WithField returns a new entry with the specified field
func (l *simpleLogger) WithField(key string, value interface{}) Logger {
	return &entry{
		logger: l,
		fields: map[string]interface{}{key: value},
		ctx:    context.Background(),
	}
}

// WithFields returns a new entry with the specified fields
func (l *simpleLogger) WithFields(fields map[string]interface{}) Logger {
	// Copy fields to avoid mutation
	fieldsCopy := make(map[string]interface{})
	for k, v := range fields {
		fieldsCopy[k] = v
	}

	return &entry{
		logger: l,
		fields: fieldsCopy,
		ctx:    context.Background(),
	}
}

// WithError returns a new entry with an error field
func (l *simpleLogger) WithError(err error) Logger {
	return &entry{
		logger: l,
		fields: make(map[string]interface{}),
		ctx:    context.Background(),
		err:    err,
	}
}

// WithContext returns a new entry with context
func (l *simpleLogger) WithContext(ctx context.Context) Logger {
	return &entry{
		logger: l,
		fields: make(map[string]interface{}),
		ctx:    ctx,
	}
}

// Debug logs at debug level
func (l *simpleLogger) Debug(args ...interface{}) {
	l.log(DebugLevel, fmt.Sprint(args...))
}

// Info logs at info level
func (l *simpleLogger) Info(args ...interface{}) {
	l.log(InfoLevel, fmt.Sprint(args...))
}

// Warn logs at warn level
func (l *simpleLogger) Warn(args ...interface{}) {
	l.log(WarnLevel, fmt.Sprint(args...))
}

// Error logs at error level
func (l *simpleLogger) Error(args ...interface{}) {
	l.log(ErrorLevel, fmt.Sprint(args...))
}

// Debugf logs formatted message at debug level
func (l *simpleLogger) Debugf(format string, args ...interface{}) {
	l.log(DebugLevel, fmt.Sprintf(format, args...))
}

// Infof logs formatted message at info level
func (l *simpleLogger) Infof(format string, args ...interface{}) {
	l.log(InfoLevel, fmt.Sprintf(format, args...))
}

// Warnf logs formatted message at warn level
func (l *simpleLogger) Warnf(format string, args ...interface{}) {
	l.log(WarnLevel, fmt.Sprintf(format, args...))
}

// Errorf logs formatted message at error level
func (l *simpleLogger) Errorf(format string, args ...interface{}) {
	l.log(ErrorLevel, fmt.Sprintf(format, args...))
}

// Entry methods - these allow method chaining like logrus.Entry

// WithField returns a new entry with the specified field
func (e *entry) WithField(key string, value interface{}) Logger {
	newFields := make(map[string]interface{})
	for k, v := range e.fields {
		newFields[k] = v
	}
	newFields[key] = value

	return &entry{
		logger: e.logger,
		fields: newFields,
		ctx:    e.ctx,
		err:    e.err,
	}
}

// WithFields returns a new entry with additional fields
func (e *entry) WithFields(fields map[string]interface{}) Logger {
	newFields := make(map[string]interface{})
	for k, v := range e.fields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}

	return &entry{
		logger: e.logger,
		fields: newFields,
		ctx:    e.ctx,
		err:    e.err,
	}
}

// WithError returns a new entry with an error field
func (e *entry) WithError(err error) Logger {
	return &entry{
		logger: e.logger,
		fields: e.fields,
		ctx:    e.ctx,
		err:    err,
	}
}

// WithContext returns a new entry with context
func (e *entry) WithContext(ctx context.Context) Logger {
	return &entry{
		logger: e.logger,
		fields: e.fields,
		ctx:    ctx,
		err:    e.err,
	}
}

// Debug logs at debug level with accumulated fields
func (e *entry) Debug(args ...interface{}) {
	e.logWithFields(DebugLevel, fmt.Sprint(args...))
}

// Info logs at info level with accumulated fields
func (e *entry) Info(args ...interface{}) {
	e.logWithFields(InfoLevel, fmt.Sprint(args...))
}

// Warn logs at warn level with accumulated fields
func (e *entry) Warn(args ...interface{}) {
	e.logWithFields(WarnLevel, fmt.Sprint(args...))
}

// Error logs at error level with accumulated fields
func (e *entry) Error(args ...interface{}) {
	e.logWithFields(ErrorLevel, fmt.Sprint(args...))
}

// Debugf logs formatted message at debug level with accumulated fields
func (e *entry) Debugf(format string, args ...interface{}) {
	e.logWithFields(DebugLevel, fmt.Sprintf(format, args...))
}

// Infof logs formatted message at info level with accumulated fields
func (e *entry) Infof(format string, args ...interface{}) {
	e.logWithFields(InfoLevel, fmt.Sprintf(format, args...))
}

// Warnf logs formatted message at warn level with accumulated fields
func (e *entry) Warnf(format string, args ...interface{}) {
	e.logWithFields(WarnLevel, fmt.Sprintf(format, args...))
}

// Errorf logs formatted message at error level with accumulated fields
func (e *entry) Errorf(format string, args ...interface{}) {
	e.logWithFields(ErrorLevel, fmt.Sprintf(format, args...))
}

// log outputs a message at the specified level (base logger)
func (l *simpleLogger) log(level Level, message string) {
	if level < l.config.Level {
		return
	}

	entry := logEntry{
		Time:    time.Now(),
		Level:   level.String(),
		Message: message,
	}

	l.writeEntry(entry)
}

// logWithFields outputs a message with fields at the specified level (entry logger)
func (e *entry) logWithFields(level Level, message string) {
	if level < e.logger.config.Level {
		return
	}

	// Check context cancellation
	if e.ctx != nil {
		select {
		case <-e.ctx.Done():
			return // Skip logging if context is canceled
		default:
		}
	}

	entry := logEntry{
		Time:    time.Now(),
		Level:   level.String(),
		Message: message,
		Fields:  make(map[string]interface{}),
	}

	// Add accumulated fields
	for k, v := range e.fields {
		entry.Fields[k] = v
	}

	// Add error if present
	if e.err != nil {
		entry.Fields["error"] = e.err.Error()
	}

	e.logger.writeEntry(entry)
}

// logEntry represents a structured log entry
type logEntry struct {
	Time    time.Time              `json:"time"`
	Level   string                 `json:"level"`
	Message string                 `json:"message"`
	Fields  map[string]interface{} `json:"fields,omitempty"`
}

// writeEntry outputs the log entry in the configured format
func (l *simpleLogger) writeEntry(entry logEntry) {
	var output string

	switch l.config.Format {
	case "json":
		data, err := json.Marshal(entry)
		if err != nil {
			// Fallback to simple text if JSON marshal fails
			output = fmt.Sprintf("%s [%s] %s\n", entry.Time.Format(time.RFC3339), entry.Level, entry.Message)
		} else {
			output = string(data) + "\n"
		}
	default: // text
		fieldsStr := ""
		if len(entry.Fields) > 0 {
			var parts []string
			for k, v := range entry.Fields {
				parts = append(parts, fmt.Sprintf("%s=%v", k, v))
			}
			fieldsStr = " " + strings.Join(parts, " ")
		}
		output = fmt.Sprintf("%s [%s] %s%s\n", entry.Time.Format("2006-01-02 15:04:05"), entry.Level, entry.Message, fieldsStr)
	}

	// Write to configured output (normally stderr)
	_, _ = l.config.Output.Write([]byte(output))
}
