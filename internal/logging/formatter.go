// Package logging provides logging configuration types and utilities.
package logging

import (
	"fmt"
	"time"

	"github.com/mrz1836/go-broadcast/internal/jsonutil"
	"github.com/sirupsen/logrus"
)

// StructuredFormatter provides JSON output formatting for structured logging.
//
// This formatter ensures consistent JSON output with standardized field names
// and proper correlation ID inclusion for log aggregation systems.
type StructuredFormatter struct {
	// DisableTimestamp disables automatic timestamp generation
	DisableTimestamp bool
	// TimestampFormat sets the format for the timestamp field
	TimestampFormat string
}

// NewStructuredFormatter creates a new StructuredFormatter with default settings.
//
// The formatter uses RFC3339 timestamp format by default and includes
// all logrus fields in the JSON output.
func NewStructuredFormatter() *StructuredFormatter {
	return &StructuredFormatter{
		TimestampFormat: time.RFC3339,
	}
}

// Format formats a logrus.Entry as JSON with standardized fields.
//
// The formatter automatically includes correlation_id if present in the
// LogConfig and ensures all field names follow the StandardFields schema.
func (f *StructuredFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	data := make(logrus.Fields)

	// Copy all fields from the entry
	for k, v := range entry.Data {
		data[k] = v
	}

	// Add standard fields
	data["level"] = entry.Level.String()
	data["message"] = entry.Message

	// Add timestamp if not disabled
	if !f.DisableTimestamp {
		timestampFormat := f.TimestampFormat
		if timestampFormat == "" {
			timestampFormat = time.RFC3339
		}
		data[StandardFields.Timestamp] = entry.Time.Format(timestampFormat)
	}

	// Serialize to JSON
	jsonBytes, err := jsonutil.MarshalJSON(data)
	if err != nil {
		return nil, err // Error already wrapped by jsonutil
	}

	// Add newline for proper log formatting
	jsonBytes = append(jsonBytes, '\n')

	return jsonBytes, nil
}

// ConfigureLogger configures a logrus.Logger instance based on LogConfig settings.
//
// This function sets up the appropriate formatter (JSON or text), log level,
// and any other logging configuration based on the provided LogConfig.
func ConfigureLogger(logger *logrus.Logger, config *LogConfig) error {
	if config == nil {
		return nil
	}

	// Set log level - verbose flags override explicit log level
	var level logrus.Level
	var err error

	if config.Verbose > 0 {
		// Map verbose level to logrus level
		switch config.Verbose {
		case 1:
			level = logrus.DebugLevel
		case 2:
			level = logrus.TraceLevel
		default: // 3 or higher
			level = logrus.TraceLevel
		}
	} else if config.LogLevel != "" {
		level, err = logrus.ParseLevel(config.LogLevel)
		if err != nil {
			return fmt.Errorf("invalid log level %q: %w", config.LogLevel, err)
		}
	} else {
		level = logrus.InfoLevel // Default level
	}

	logger.SetLevel(level)

	// Install redaction hook for automatic sensitive data protection
	redactionService := NewRedactionService()
	redactionHook := redactionService.CreateHook()
	logger.AddHook(redactionHook)

	// Configure formatter based on JSON output setting
	if config.JSONOutput || config.LogFormat == "json" {
		logger.SetFormatter(NewStructuredFormatter())
	} else {
		// Use default text formatter for human-readable output
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	}

	return nil
}

// WithStandardFields creates a logrus.Entry with correlation ID and component info.
//
// This helper function automatically includes the correlation ID from LogConfig
// and sets up standard fields for consistent logging across components.
func WithStandardFields(logger *logrus.Logger, config *LogConfig, component string) *logrus.Entry {
	fields := logrus.Fields{
		StandardFields.Component: component,
	}

	// Add correlation ID if available
	if config != nil && config.CorrelationID != "" {
		fields[StandardFields.CorrelationID] = config.CorrelationID
	}

	return logger.WithFields(fields)
}
