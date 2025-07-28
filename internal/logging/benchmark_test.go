package logging

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mrz1836/go-broadcast/internal/benchmark"
	"github.com/sirupsen/logrus"
)

func BenchmarkRedaction_Scenarios(b *testing.B) {
	scenarios := []struct {
		name string
		text string
	}{
		{"NoSensitive", "This is a normal log message without any sensitive data"},
		{"WithGitHubToken", "Authorization: Bearer ghp_1234567890abcdefghijklmnopqrstu"},
		{"WithMultipleTokens", strings.Repeat("token: ghp_abcd1234 secret: ghs_efgh5678 ", 10)},
		{"WithSSHKey", "SSH Key: -----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEA1234567890\n-----END RSA PRIVATE KEY-----"},
		{"WithURLPassword", "Database URL: postgres://user:mypassword123@localhost:5432/db"},
		{"WithBase64Secret", "Config contains: YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXoxMjM0NTY3ODkw base64 encoded secret"},
		{"WithEnvironmentVars", "Environment: API_TOKEN=secret123 DATABASE_PASSWORD=mysecret456"},
		{"MixedSensitiveData", "User ghp_token123 accessed https://user:pass@github.com with JWT eyJ0eXAi"},
		{"LargeText", string(benchmark.GenerateTestData("large")) + " ghp_secrettoken123"},
	}

	redactor := NewRedactionService()

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			benchmark.WithMemoryTracking(b, func() {
				_ = redactor.RedactSensitive(scenario.text)
			})
		})
	}
}

func BenchmarkRedaction_TokenTypes(b *testing.B) {
	tokenTypes := []struct {
		name  string
		token string
	}{
		{"GitHubPersonal", "ghp_1234567890abcdefghijklmnopqrstuvwxyz"},
		{"GitHubApp", "ghs_abcdefghij1234567890"},
		{"GitHubPAT", "github_pat_11ABCDEFG_abcdefghijklmnopqrstuvwxyz1234567890"},
		{"GitHubRefresh", "ghr_1234567890abcdefghijklmnopqrstu"},
		{"BearerToken", "Bearer abc123def456ghi789jklmnop"},
		{"JWTToken", "JWT eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWV9.TJVA95OrM7E2cBab30RMHrHDcEfxjoYZgeFONFh7HgQ"},
		{"Base64Secret", "YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXoxMjM0NTY3ODkwMTIzNDU2Nzg5MA=="},
	}

	redactor := NewRedactionService()

	for _, tokenType := range tokenTypes {
		b.Run(tokenType.name, func(b *testing.B) {
			testText := "Token: " + tokenType.token
			benchmark.WithMemoryTracking(b, func() {
				_ = redactor.RedactSensitive(testText)
			})
		})
	}
}

func BenchmarkFormatting_Types(b *testing.B) {
	// Create test log entries with different complexity
	entries := []struct {
		name   string
		fields map[string]interface{}
	}{
		{"Simple", map[string]interface{}{
			"message": "Simple log message",
			"level":   "INFO",
		}},
		{"WithFields", map[string]interface{}{
			"message":  "Log with fields",
			"level":    "INFO",
			"user":     "test-user",
			"duration": 123.45,
			"count":    100,
			"success":  true,
		}},
		{"ManyFields", func() map[string]interface{} {
			fields := make(map[string]interface{})
			fields["message"] = "Log with many fields"
			fields["level"] = "INFO"
			for i := 0; i < 20; i++ {
				fields[string(rune('a'+i))] = i * 10
			}
			return fields
		}()},
		{"NestedFields", map[string]interface{}{
			"message": "Log with nested data",
			"level":   "INFO",
			"metadata": map[string]interface{}{
				"user": map[string]interface{}{
					"id":   123,
					"name": "John Doe",
				},
				"request": map[string]interface{}{
					"method": "GET",
					"path":   "/api/v1/users",
					"params": []string{"param1", "param2"},
				},
			},
		}},
		{"WithSensitiveData", map[string]interface{}{
			"message":  "Log with sensitive data",
			"level":    "INFO",
			"token":    "ghp_1234567890abcdefghij",
			"password": "secretpassword123",
			"api_key":  "sk_test_1234567890",
		}},
	}

	formatters := []struct {
		name      string
		formatter logrus.Formatter
	}{
		{"Text", &logrus.TextFormatter{DisableTimestamp: true}},
		{"JSON", &logrus.JSONFormatter{DisableTimestamp: true}},
		{"Structured", NewStructuredFormatter()},
	}

	for _, formatter := range formatters {
		for _, entry := range entries {
			b.Run(formatter.name+"_"+entry.name, func(b *testing.B) {
				logEntry := &logrus.Entry{
					Logger: logrus.New(),
					Data:   logrus.Fields(entry.fields),
					Level:  logrus.InfoLevel,
				}
				if msg, ok := entry.fields["message"].(string); ok {
					logEntry.Message = msg
				}

				benchmark.WithMemoryTracking(b, func() {
					_, _ = formatter.formatter.Format(logEntry)
				})
			})
		}
	}
}

func BenchmarkConcurrentLogging(b *testing.B) {
	goroutineCounts := []int{1, 10, 50, 100}

	for _, count := range goroutineCounts {
		b.Run(string(rune('0'+count/10))+"0_Goroutines", func(b *testing.B) {
			logger := logrus.New()
			logger.SetOutput(&bytes.Buffer{}) // Discard output for performance

			// Add redaction hook
			redactor := NewRedactionService()
			logger.AddHook(redactor.CreateHook())

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					logger.WithFields(logrus.Fields{
						"user":      "test-user",
						"operation": "sync",
						"repo":      "user/repo",
						"token":     "ghp_should_be_redacted123",
					}).Info("Concurrent log message with sensitive data")
				}
			})
		})
	}
}

func BenchmarkRedactionHook_Processing(b *testing.B) {
	scenarios := []struct {
		name   string
		fields logrus.Fields
	}{
		{"NoSensitiveFields", logrus.Fields{
			"user":      "test-user",
			"operation": "sync",
			"duration":  123.45,
		}},
		{"WithSensitiveFields", logrus.Fields{
			"user":     "test-user",
			"password": "secret123",
			"api_key":  "key_abc123",
			"token":    "ghp_1234567890",
		}},
		{"MixedSensitiveAndNormal", logrus.Fields{
			"user":       "test-user",
			"operation":  "sync",
			"gh_token":   "ghp_abcdef123456",
			"duration":   123.45,
			"secret_key": "mysecret456",
			"repo_count": 5,
		}},
		{"NestedSensitiveData", logrus.Fields{
			"user": "test-user",
			"config": map[string]interface{}{
				"database_password": "dbpass123",
				"api_settings": map[string]interface{}{
					"token":   "nested_token_123",
					"timeout": 30,
				},
			},
		}},
	}

	redactor := NewRedactionService()
	hook := redactor.CreateHook()

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			benchmark.WithMemoryTracking(b, func() {
				entry := &logrus.Entry{
					Logger:  logrus.New(),
					Data:    scenario.fields,
					Level:   logrus.InfoLevel,
					Message: "Test message with potential token ghp_example123",
				}
				_ = hook.Fire(entry)
			})
		})
	}
}

func BenchmarkFieldSensitivityCheck(b *testing.B) {
	fieldNames := []string{
		"user",              // Not sensitive
		"operation",         // Not sensitive
		"password",          // Sensitive
		"token",             // Sensitive
		"gh_token",          // Sensitive
		"api_key",           // Sensitive
		"secret_value",      // Sensitive
		"database_url",      // Sensitive
		"connection_string", // Sensitive
		"normal_field",      // Not sensitive
		"count",             // Not sensitive
		"private_key",       // Sensitive
	}

	redactor := NewRedactionService()

	b.Run("IsSensitiveField", func(b *testing.B) {
		benchmark.WithMemoryTracking(b, func() {
			fieldName := fieldNames[0] // Use first field for consistent benchmarking
			_ = redactor.IsSensitiveField(fieldName)
		})
	})
}

func BenchmarkMemoryUsage_LoggingOperations(b *testing.B) {
	operations := []struct {
		name string
		fn   func()
	}{
		{"CreateRedactionService", func() {
			_ = NewRedactionService()
		}},
		{"CreateAuditLogger", func() {
			_ = NewAuditLogger()
		}},
		{"CreateStructuredFormatter", func() {
			_ = NewStructuredFormatter()
		}},
		{"LargeMessageRedaction", func() {
			redactor := NewRedactionService()
			largeMessage := string(benchmark.GenerateTestData("large")) + " token: ghp_secret123"
			_ = redactor.RedactSensitive(largeMessage)
		}},
		{"ComplexFieldProcessing", func() {
			redactor := NewRedactionService()
			hook := redactor.CreateHook()
			entry := &logrus.Entry{
				Logger: logrus.New(),
				Data: logrus.Fields{
					"large_data": string(benchmark.GenerateTestData("medium")),
					"tokens": []string{
						"ghp_token1", "ghs_token2", "github_pat_token3",
					},
					"nested": map[string]interface{}{
						"password": "secret123",
						"config": map[string]interface{}{
							"api_key": "key_abc123",
							"timeout": 30,
						},
					},
				},
				Level:   logrus.InfoLevel,
				Message: "Complex log entry with nested sensitive data",
			}
			_ = hook.Fire(entry)
		}},
	}

	for _, op := range operations {
		b.Run(op.name, func(b *testing.B) {
			benchmark.RunWithMemoryTracking(b, op.name, op.fn)
		})
	}
}

func BenchmarkLogEntryGeneration(b *testing.B) {
	entryCounts := []int{10, 100, 1000, 5000}

	for _, count := range entryCounts {
		b.Run(string(rune('0'+count/1000))+"k_Entries", func(b *testing.B) {
			entries := benchmark.GenerateLogEntries(count, true) // With tokens

			benchmark.WithMemoryTracking(b, func() {
				for _, entry := range entries {
					_ = len(entry) // Simulate processing
				}
			})
		})
	}
}

func BenchmarkAuditLogging(b *testing.B) {
	auditOperations := []struct {
		name string
		fn   func(audit *AuditLogger)
	}{
		{"Authentication", func(audit *AuditLogger) {
			audit.LogAuthentication("test-user", "token", true)
		}},
		{"ConfigChange", func(audit *AuditLogger) {
			audit.LogConfigChange("admin-user", "update", nil)
		}},
		{"RepositoryAccess", func(audit *AuditLogger) {
			audit.LogRepositoryAccess("user", "org/repo", "read")
		}},
	}

	for _, op := range auditOperations {
		b.Run(op.name, func(b *testing.B) {
			audit := NewAuditLogger()
			// Redirect output to prevent I/O overhead in benchmark
			audit.logger.Logger.SetOutput(&bytes.Buffer{})

			benchmark.WithMemoryTracking(b, func() {
				op.fn(audit)
			})
		})
	}
}

func BenchmarkPatternMatching_RegexPerformance(b *testing.B) {
	// Test different text sizes with various patterns
	textSizes := []struct {
		name string
		size string
	}{
		{"Small", "small"},
		{"Medium", "medium"},
		{"Large", "large"},
	}

	patterns := []struct {
		name string
		text string
	}{
		{"NoMatches", "This text contains no sensitive patterns at all"},
		{"SingleToken", "Authorization header: Bearer ghp_1234567890abcdef"},
		{"MultipleTokens", "Config: ghp_token1 ghs_token2 github_pat_token3"},
		{"MixedPatterns", "URL: https://user:pass@github.com JWT: eyJ0eXAi token=secret123"},
	}

	redactor := NewRedactionService()

	for _, textSize := range textSizes {
		for _, pattern := range patterns {
			b.Run(textSize.name+"_"+pattern.name, func(b *testing.B) {
				// Create text of appropriate size with the pattern
				baseText := string(benchmark.GenerateTestData(textSize.size))
				testText := baseText + " " + pattern.text

				benchmark.WithMemoryTracking(b, func() {
					_ = redactor.RedactSensitive(testText)
				})
			})
		}
	}
}
