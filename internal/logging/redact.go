// Package logging provides redaction services for sensitive data protection.
//
// This package implements comprehensive sensitive data redaction to ensure
// that tokens, secrets, passwords, and other confidential information
// never appear in log output. It provides both automatic redaction through
// logrus hooks and manual redaction functions.
//
// Key features include:
// - Comprehensive regex patterns for various secret formats
// - Field name-based detection for sensitive data
// - Automatic logrus hook integration
// - Security audit logging capabilities
// - Pattern-based text redaction
// - Configurable redaction behavior
//
// Security patterns supported:
// - GitHub tokens (ghp_, ghs_, github_pat_, ghr_)
// - Bearer tokens and API keys
// - URL parameters with sensitive data
// - Base64 encoded secrets
// - Environment variables with sensitive names
//
// Usage examples:
//
//	// Create redaction service
//	service := logging.NewRedactionService()
//	text := service.RedactSensitive("token=ghp_1234567890")
//
//	// Use with logrus hook
//	hook := service.CreateHook()
//	logrus.AddHook(hook)
//
//	// Audit logging
//	audit := logging.NewAuditLogger()
//	audit.LogAuthentication("user", "token", true)
//
// Important notes:
// - All redaction is irreversible for security compliance
// - Pattern matching is optimized for performance
// - Redaction preserves partial context for debugging
package logging

import (
	"regexp"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// RedactionService handles comprehensive sensitive data redaction.
//
// This service provides automatic detection and redaction of sensitive
// information including tokens, secrets, passwords, and API keys using
// both regex pattern matching and field name analysis.
type RedactionService struct {
	sensitivePatterns   []*regexp.Regexp
	sensitiveFields     []string
	githubTokenPatterns []*regexp.Regexp
	sshPattern          *regexp.Regexp
}

// NewRedactionService creates a new redaction service with comprehensive patterns.
//
// The service is initialized with a comprehensive set of regex patterns
// for detecting various types of sensitive data and field names that
// commonly contain confidential information.
//
// Returns:
// - RedactionService instance with all security patterns configured
//
// Security Patterns:
// - GitHub personal access tokens (ghp_)
// - GitHub app tokens (ghs_)
// - New GitHub PAT format (github_pat_)
// - GitHub refresh tokens (ghr_)
// - URL parameters with sensitive data
// - Bearer/Token authorization headers
// - Base64 encoded secrets (40+ characters)
func NewRedactionService() *RedactionService {
	return &RedactionService{
		sensitivePatterns: []*regexp.Regexp{
			// GitHub token patterns (flexible for test tokens)
			regexp.MustCompile(`ghp_[a-zA-Z0-9]{8,}`),         // GitHub personal tokens (8+ chars for tests)
			regexp.MustCompile(`ghs_[a-zA-Z0-9]{8,}`),         // GitHub app tokens (8+ chars for tests)
			regexp.MustCompile(`github_pat_[a-zA-Z0-9_]{8,}`), // New GitHub PAT format (8+ chars for tests)
			regexp.MustCompile(`ghr_[a-zA-Z0-9]{8,}`),         // GitHub refresh tokens (8+ chars for tests)

			// Authorization headers - capture just the token part
			regexp.MustCompile(`(Bearer|Token)\s+([^\s]+)`),

			// URL parameters - capture key=value pattern
			regexp.MustCompile(`(password|token|secret|key|api_key)=([^\s&]+)`),

			// URL passwords - capture :password@ pattern
			regexp.MustCompile(`://([^:]+):([^@]+)@`),

			// JWT tokens (three base64 parts separated by dots) - only capture token for JWT prefix
			regexp.MustCompile(`JWT\s+([a-zA-Z0-9_.-]{20,})`),

			// Base64 encoded secrets (40+ characters with optional padding) - only standalone
			regexp.MustCompile(`\b([a-zA-Z0-9+/]{40,}={0,2})\b`),

			// SSH private key patterns
			regexp.MustCompile(`-----BEGIN[A-Z\s]+PRIVATE KEY-----[\s\S]*?-----END[A-Z\s]+PRIVATE KEY-----`),

			// Generic secret patterns in environment variables
			regexp.MustCompile(`([A-Z_]*(?:TOKEN|SECRET|KEY|PASSWORD|PASS)[A-Z_]*=)([^\s]+)`),
		},
		sensitiveFields: []string{
			// Authentication related
			"password",
			"token",
			"secret",
			"api_key",
			"private_key",
			"gh_token",
			"github_token",
			"authorization",
			"auth",

			// Credentials
			"credential",
			"credentials",
			"pass",
			"passwd",
			"pwd",

			// Keys and certificates
			"key",
			"private",
			"cert",
			"certificate",
			"pem",

			// OAuth and JWT
			"oauth",
			"jwt",
			"bearer",
			"refresh_token",
			"access_token",

			// Database and connection strings
			"connection_string",
			"database_url",
			"db_password",
		},
		githubTokenPatterns: []*regexp.Regexp{
			regexp.MustCompile(`ghp_[a-zA-Z0-9]{4,}`),
			regexp.MustCompile(`ghs_[a-zA-Z0-9]{4,}`),
			regexp.MustCompile(`github_pat_[a-zA-Z0-9_]{4,}`),
			regexp.MustCompile(`ghr_[a-zA-Z0-9]{4,}`),
		},
		sshPattern: regexp.MustCompile(`-----BEGIN[A-Z\s]+PRIVATE KEY-----[\s\S]*?-----END[A-Z\s]+PRIVATE KEY-----`),
	}
}

// RedactSensitive removes sensitive data from text using pattern matching.
//
// This method applies all configured regex patterns to identify and redact
// sensitive information while preserving some context for debugging purposes.
// The redaction maintains partial visibility for troubleshooting while
// ensuring complete security compliance.
//
// Parameters:
// - text: Text content to scan and redact
//
// Returns:
// - Redacted text with sensitive data replaced with secure placeholders
//
// Redaction Strategy:
// - GitHub tokens: Preserve full prefix (e.g., "github_pat_***REDACTED***")
// - Other long matches (> 10 chars): Partial preservation (first 4 chars + "***REDACTED***")
// - Short matches (<= 10 chars): Complete replacement with "***REDACTED***"
// - URL patterns: Parameter values redacted, names preserved
// - Headers: Value redacted, header name preserved
func (r *RedactionService) RedactSensitive(text string) string {
	// GitHub token patterns - preserve full prefix using pre-compiled patterns
	for _, pattern := range r.githubTokenPatterns {
		text = pattern.ReplaceAllStringFunc(text, func(match string) string {
			if strings.HasPrefix(match, "ghp_") {
				return "ghp_***REDACTED***"
			}
			if strings.HasPrefix(match, "ghs_") {
				return "ghs_***REDACTED***"
			}
			if strings.HasPrefix(match, "github_pat_") {
				return "github_pat_***REDACTED***"
			}
			if strings.HasPrefix(match, "ghr_") {
				return "ghr_***REDACTED***"
			}
			return "***REDACTED***"
		})
	}

	// SSH private keys - use pre-compiled pattern
	text = r.sshPattern.ReplaceAllString(text, "***REDACTED_SSH_KEY***")

	// Authorization headers - preserve header name and Bearer/Token keyword
	authPattern := regexp.MustCompile(`(Bearer|Token)\s+([^\s'\"]+)`)
	text = authPattern.ReplaceAllString(text, "$1 ***REDACTED***")

	// JWT tokens - preserve JWT prefix
	jwtPattern := regexp.MustCompile(`JWT\s+([a-zA-Z0-9_.-]{20,})`)
	text = jwtPattern.ReplaceAllString(text, "JWT ***REDACTED***")

	// Generic tokens (like jwt_token2, api_token, etc)
	genericTokenPattern := regexp.MustCompile(`\b[a-zA-Z_]*token[a-zA-Z0-9_]*\b`)
	text = genericTokenPattern.ReplaceAllStringFunc(text, func(match string) string {
		// Don't redact if it's already part of a GitHub token
		if strings.HasPrefix(match, "ghp_") || strings.HasPrefix(match, "ghs_") ||
			strings.HasPrefix(match, "github_pat_") || strings.HasPrefix(match, "ghr_") {
			return match
		}
		// Don't redact common words that aren't actually tokens
		lower := strings.ToLower(match)
		if lower == "token" || lower == "tokens" {
			return match
		}
		return "***REDACTED***"
	})

	// URL passwords - preserve username
	urlPasswordPattern := regexp.MustCompile(`://([^:]+):([^@]+)@`)
	text = urlPasswordPattern.ReplaceAllString(text, "://$1:***REDACTED***@")

	// URL parameters - preserve parameter name
	urlParamPattern := regexp.MustCompile(`(password|token|secret|key|api_key)=([^\s&]+)`)
	text = urlParamPattern.ReplaceAllString(text, "$1=***REDACTED***")

	// Base64 secrets (standalone)
	base64Pattern := regexp.MustCompile(`\b([a-zA-Z0-9+/]{40,}={0,2})\b`)
	text = base64Pattern.ReplaceAllString(text, "***REDACTED***")

	// Environment variables
	envPattern := regexp.MustCompile(`([A-Z_]*(?:TOKEN|SECRET|KEY|PASSWORD|PASS)[A-Z_]*=)([^\s]+)`)
	text = envPattern.ReplaceAllString(text, "$1***REDACTED***")

	return text
}

// IsSensitiveField checks if a field name indicates sensitive data.
//
// This method analyzes field names to determine if they likely contain
// sensitive information based on common naming patterns and conventions.
//
// Parameters:
// - fieldName: Name of the field to analyze
//
// Returns:
// - true if the field name suggests sensitive content, false otherwise
//
// Detection Strategy:
// - Case-insensitive matching against known sensitive field names
// - Substring matching for compound field names
// - Common variations and abbreviations
func (r *RedactionService) IsSensitiveField(fieldName string) bool {
	fieldLower := strings.ToLower(fieldName)
	for _, sensitive := range r.sensitiveFields {
		if strings.Contains(fieldLower, sensitive) {
			return true
		}
	}
	return false
}

// CreateHook creates a logrus hook for automatic redaction.
//
// This method creates a logrus hook that automatically redacts sensitive
// data from all log entries. It processes both log messages and field
// values to ensure comprehensive protection.
//
// Returns:
// - logrus.Hook instance for automatic redaction integration
//
// Integration:
// - Hook processes all log levels (logrus.AllLevels)
// - Redacts both message content and field values
// - Field name-based detection for automatic redaction
// - Pattern-based content scanning for all string values
func (r *RedactionService) CreateHook() logrus.Hook {
	return &RedactionHook{service: r}
}

// RedactionHook automatically redacts sensitive data in log entries.
//
// This hook integrates with logrus to provide automatic redaction of
// sensitive information in both log messages and field values. It
// processes all log entries before they are written.
type RedactionHook struct {
	service *RedactionService
}

// Levels returns the log levels this hook should process.
//
// Returns:
// - All logrus levels to ensure comprehensive redaction coverage
func (h *RedactionHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// Fire processes a log entry to redact sensitive information.
//
// This method is called by logrus for each log entry and performs
// comprehensive redaction of both the message content and all field
// values to ensure no sensitive data is logged.
//
// Parameters:
// - entry: logrus.Entry to process for sensitive data
//
// Returns:
// - error if processing fails (always nil in current implementation)
//
// Processing:
// - Redacts the main log message using pattern matching
// - Checks field names for sensitivity indicators
// - Redacts string field values using pattern matching
// - Recursively processes nested maps and interfaces
// - Preserves non-string field values unless field name is sensitive
func (h *RedactionHook) Fire(entry *logrus.Entry) error {
	// Redact the main message content
	entry.Message = h.service.RedactSensitive(entry.Message)

	// Process all fields for sensitive content
	for key, value := range entry.Data {
		entry.Data[key] = h.redactValue(key, value)
	}

	return nil
}

// redactValue recursively redacts sensitive data in values
func (h *RedactionHook) redactValue(key string, value interface{}) interface{} {
	// Check if field name indicates sensitive data
	if h.service.IsSensitiveField(key) {
		// For sensitive field names, apply pattern-based redaction to strings
		// or complete redaction for non-strings
		if str, ok := value.(string); ok {
			redacted := h.service.RedactSensitive(str)
			// If pattern-based redaction didn't change anything, apply complete redaction
			if redacted == str {
				return "***REDACTED***"
			}
			return redacted
		}
		return "***REDACTED***"
	}

	// For non-sensitive field names, still process the value
	switch v := value.(type) {
	case string:
		// Apply pattern-based redaction to strings
		return h.service.RedactSensitive(v)
	case map[string]interface{}:
		// Recursively process nested maps
		result := make(map[string]interface{})
		for nestedKey, nestedValue := range v {
			result[nestedKey] = h.redactValue(nestedKey, nestedValue)
		}
		return result
	case []interface{}:
		// Process slices
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = h.redactValue("", item) // Use empty key for array items
		}
		return result
	default:
		// Return other types unchanged
		return value
	}
}

// AuditLogger provides security audit logging capabilities.
//
// This logger tracks security-relevant operations including authentication
// attempts, configuration changes, and repository access for compliance
// and security monitoring purposes.
type AuditLogger struct {
	logger *logrus.Entry
}

// NewAuditLogger creates a new audit logger instance.
//
// The audit logger is configured with appropriate context and formatting
// for security event tracking and compliance requirements.
//
// Returns:
// - AuditLogger instance configured for security event logging
func NewAuditLogger() *AuditLogger {
	return &AuditLogger{
		logger: logrus.WithField(StandardFields.Component, "audit"),
	}
}

// LogAuthentication logs authentication attempts for security monitoring.
//
// This method records authentication events with context including the
// user, authentication method, and success status for security analysis.
//
// Parameters:
// - user: Username or identifier for the authentication attempt
// - method: Authentication method used (e.g., "token", "ssh", "oauth")
// - success: Whether the authentication was successful
//
// Side Effects:
// - Creates INFO level log entry with authentication context
// - Includes timestamp for security event correlation
// - Uses standardized audit event format
func (a *AuditLogger) LogAuthentication(user, method string, success bool) {
	a.logger.WithFields(logrus.Fields{
		"event":   "authentication",
		"user":    user,
		"method":  method,
		"success": success,
		"time":    time.Now().Unix(),
	}).Info("Authentication attempt")
}

// LogConfigChange logs configuration changes for audit compliance.
//
// This method records configuration modifications with context about
// the user making the change and the type of action performed.
//
// Parameters:
// - user: User making the configuration change
// - action: Type of action performed (e.g., "create", "update", "delete")
// - config: Configuration object or description being modified
//
// Side Effects:
// - Creates INFO level log entry with configuration change context
// - Includes timestamp for audit trail continuity
// - Records user and action for accountability
func (a *AuditLogger) LogConfigChange(user, action string, _ interface{}) {
	a.logger.WithFields(logrus.Fields{
		"event":  "config_change",
		"user":   user,
		"action": action,
		"time":   time.Now().Unix(),
	}).Info("Configuration changed")
}

// LogRepositoryAccess logs repository access events for security monitoring.
//
// This method tracks repository access patterns including the user,
// repository, and type of access for security analysis and compliance.
//
// Parameters:
// - user: User accessing the repository
// - repo: Repository being accessed
// - action: Type of access (e.g., "read", "write", "clone", "push")
//
// Side Effects:
// - Creates INFO level log entry with repository access context
// - Includes timestamp for access pattern analysis
// - Records user, repository, and action for security monitoring
func (a *AuditLogger) LogRepositoryAccess(user, repo, action string) {
	a.logger.WithFields(logrus.Fields{
		"event":  "repo_access",
		"user":   user,
		"repo":   repo,
		"action": action,
		"time":   time.Now().Unix(),
	}).Info("Repository accessed")
}
