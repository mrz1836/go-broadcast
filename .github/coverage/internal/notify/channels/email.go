// Package channels provides Email notification channel implementation
package channels

import (
	"context"
	"crypto/tls"
	"fmt"
	"html/template"
	"net/smtp"
	"strings"
	"time"

	"github.com/mrz1836/go-broadcast/coverage/internal/types"
)

// EmailChannel implements SMTP email notifications
type EmailChannel struct {
	config    *types.EmailConfig
	rateLimit *types.RateLimit
	auth      smtp.Auth
	templates *EmailTemplates
}

// EmailTemplates holds email templates
type EmailTemplates struct {
	PlainText *template.Template
	HTML      *template.Template
}

// EmailData represents data for email template rendering
type EmailData struct {
	Notification *types.Notification
	Subject      string
	Repository   string
	Branch       string
	Author       string
	Coverage     *CoverageEmailData
	Links        []LinkData
	GeneratedAt  string
}

// CoverageEmailData represents coverage data for email templates
type CoverageEmailData struct {
	Current    float64
	Previous   float64
	Change     float64
	Target     float64
	Threshold  float64
	ChangeIcon string
	Status     string
}

// LinkData represents link information for emails
type LinkData struct {
	Text string
	URL  string
	Type string
}

// NewEmailChannel creates a new Email notification channel
func NewEmailChannel(config *types.EmailConfig) *EmailChannel {
	var auth smtp.Auth
	if config.Username != "" && config.Password != "" {
		auth = smtp.PlainAuth("", config.Username, config.Password, config.SMTPHost)
	}

	channel := &EmailChannel{
		config: config,
		rateLimit: &types.RateLimit{
			RequestsPerMinute: 10, // Conservative email rate limit
			RequestsPerHour:   300,
			RequestsPerDay:    2000,
			BurstSize:         3,
		},
		auth: auth,
	}

	// Initialize templates
	channel.templates = channel.initializeTemplates()

	return channel
}

// Send implements the NotificationChannel interface for Email
func (e *EmailChannel) Send(ctx context.Context, notification *types.Notification) (*types.DeliveryResult, error) {
	startTime := time.Now()
	result := &types.DeliveryResult{
		Channel:   types.ChannelEmail,
		Timestamp: startTime,
	}

	// Build email message
	message, err := e.buildEmailMessage(notification)
	if err != nil {
		result.Error = fmt.Errorf("failed to build email message: %w", err)
		return result, result.Error
	}

	// Send email via SMTP
	err = e.sendSMTP(ctx, message)
	if err != nil {
		result.Error = fmt.Errorf("failed to send email: %w", err)
		return result, result.Error
	}

	result.DeliveryTime = time.Since(startTime)
	result.Success = true
	result.MessageID = fmt.Sprintf("email_%d", time.Now().Unix())

	return result, nil
}

// ValidateConfig validates the Email channel configuration
func (e *EmailChannel) ValidateConfig() error {
	if e.config == nil {
		return fmt.Errorf("Email config is nil")
	}

	if e.config.SMTPHost == "" {
		return fmt.Errorf("SMTP host is required")
	}

	if e.config.SMTPPort <= 0 || e.config.SMTPPort > 65535 {
		return fmt.Errorf("invalid SMTP port: %d", e.config.SMTPPort)
	}

	if e.config.FromEmail == "" {
		return fmt.Errorf("from email is required")
	}

	if len(e.config.ToEmails) == 0 {
		return fmt.Errorf("at least one recipient email is required")
	}

	// Validate email addresses
	if !isValidEmail(e.config.FromEmail) {
		return fmt.Errorf("invalid from email: %s", e.config.FromEmail)
	}

	for _, addr := range e.config.ToEmails {
		if !isValidEmail(addr) {
			return fmt.Errorf("invalid recipient email: %s", addr)
		}
	}

	return nil
}

// GetChannelType returns the channel type
func (e *EmailChannel) GetChannelType() types.ChannelType {
	return types.ChannelEmail
}

// SupportsRichContent returns whether the channel supports rich content
func (e *EmailChannel) SupportsRichContent() bool {
	return true
}

// GetRateLimit returns the rate limit configuration
func (e *EmailChannel) GetRateLimit() *types.RateLimit {
	return e.rateLimit
}

// buildEmailMessage builds an email message from a notification
func (e *EmailChannel) buildEmailMessage(notification *types.Notification) (string, error) {
	// Prepare email data
	emailData := &EmailData{
		Notification: notification,
		Subject:      notification.Subject,
		Repository:   notification.Repository,
		Branch:       notification.Branch,
		Author:       notification.Author,
		GeneratedAt:  time.Now().Format("2006-01-02 15:04:05"),
		Links:        make([]LinkData, 0),
	}

	// Prepare coverage data
	if notification.CoverageData != nil {
		emailData.Coverage = &CoverageEmailData{
			Current:    notification.CoverageData.Current,
			Previous:   notification.CoverageData.Previous,
			Change:     notification.CoverageData.Change,
			Target:     notification.CoverageData.Target,
			Threshold:  80.0,
			ChangeIcon: e.getCoverageChangeIcon(notification.CoverageData.Change),
			Status:     e.getCoverageStatus(notification.CoverageData.Current, notification.CoverageData.Target),
		}
	}

	// Add commit information
	if notification.CommitSHA != "" {
		commitURL := fmt.Sprintf("https://github.com/%s/commit/%s", notification.Repository, notification.CommitSHA)
		shortSHA := notification.CommitSHA
		if len(shortSHA) > 8 {
			shortSHA = shortSHA[:8]
		}
		emailData.Links = append(emailData.Links, LinkData{
			Text: fmt.Sprintf("View Commit %s", shortSHA),
			URL:  commitURL,
			Type: "commit",
		})
	}

	// Add default repository link
	if notification.Repository != "" {
		emailData.Links = append(emailData.Links, LinkData{
			Text: "View Repository",
			URL:  fmt.Sprintf("https://github.com/%s", notification.Repository),
			Type: "repository",
		})
	}

	// Add PR link
	if notification.Repository != "" && notification.PRNumber > 0 {
		emailData.Links = append(emailData.Links, LinkData{
			Text: fmt.Sprintf("View PR #%d", notification.PRNumber),
			URL:  fmt.Sprintf("https://github.com/%s/pull/%d", notification.Repository, notification.PRNumber),
			Type: "pull_request",
		})
	}

	// Build email headers
	headers := e.buildEmailHeaders(emailData)

	// Build email body
	var body string
	var err error

	if e.templates.HTML != nil {
		body, err = e.renderHTMLTemplate(emailData)
	} else {
		body, err = e.renderPlainTextTemplate(emailData)
	}

	if err != nil {
		return "", fmt.Errorf("failed to render email template: %w", err)
	}

	// Combine headers and body
	message := headers + "\r\n" + body

	return message, nil
}

// buildEmailHeaders builds email headers
func (e *EmailChannel) buildEmailHeaders(emailData *EmailData) string {
	var headers strings.Builder

	// From header
	headers.WriteString(fmt.Sprintf("From: %s\r\n", e.config.FromEmail))

	// To header
	headers.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(e.config.ToEmails, ", ")))

	// Subject header with severity indicator
	severityPrefix := e.getSeverityPrefix(emailData.Notification.Severity)
	subject := fmt.Sprintf("%s %s", severityPrefix, emailData.Subject)
	headers.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))

	// Content-Type header
	if e.templates.HTML != nil {
		headers.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	} else {
		headers.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	}

	// Additional headers
	headers.WriteString("MIME-Version: 1.0\r\n")
	headers.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))
	headers.WriteString("X-Mailer: GoFortress Coverage System\r\n")
	headers.WriteString(fmt.Sprintf("X-Priority: %s\r\n", e.getPriorityLevel(emailData.Notification.Priority)))

	return headers.String()
}

// renderPlainTextTemplate renders the plain text email template
func (e *EmailChannel) renderPlainTextTemplate(emailData *EmailData) (string, error) {
	if e.templates.PlainText == nil {
		// Fallback to simple text if no template
		return e.buildSimplePlainText(emailData), nil
	}

	var body strings.Builder
	err := e.templates.PlainText.Execute(&body, emailData)
	if err != nil {
		return "", err
	}

	return body.String(), nil
}

// renderHTMLTemplate renders the HTML email template
func (e *EmailChannel) renderHTMLTemplate(emailData *EmailData) (string, error) {
	if e.templates.HTML == nil {
		// Fallback to simple HTML if no template
		return e.buildSimpleHTML(emailData), nil
	}

	var body strings.Builder
	err := e.templates.HTML.Execute(&body, emailData)
	if err != nil {
		return "", err
	}

	return body.String(), nil
}

// buildSimplePlainText builds a simple plain text email
func (e *EmailChannel) buildSimplePlainText(emailData *EmailData) string {
	var body strings.Builder

	body.WriteString(fmt.Sprintf("Coverage Notification: %s\n", emailData.Subject))
	body.WriteString(strings.Repeat("=", 50) + "\n\n")

	body.WriteString(fmt.Sprintf("Message: %s\n\n", emailData.Notification.Message))

	if emailData.Repository != "" {
		body.WriteString(fmt.Sprintf("Repository: %s\n", emailData.Repository))
	}

	if emailData.Branch != "" {
		body.WriteString(fmt.Sprintf("Branch: %s\n", emailData.Branch))
	}

	if emailData.Author != "" {
		body.WriteString(fmt.Sprintf("Author: %s\n", emailData.Author))
	}

	if emailData.Coverage != nil {
		body.WriteString("\nCoverage Information:\n")
		body.WriteString(fmt.Sprintf("  Current: %.1f%%\n", emailData.Coverage.Current))

		if emailData.Coverage.Previous > 0 {
			body.WriteString(fmt.Sprintf("  Previous: %.1f%%\n", emailData.Coverage.Previous))
			body.WriteString(fmt.Sprintf("  Change: %+.1f%% %s\n", emailData.Coverage.Change, emailData.Coverage.ChangeIcon))
		}

		if emailData.Coverage.Target > 0 {
			body.WriteString(fmt.Sprintf("  Target: %.1f%%\n", emailData.Coverage.Target))
			body.WriteString(fmt.Sprintf("  Status: %s\n", emailData.Coverage.Status))
		}
	}

	body.WriteString(fmt.Sprintf("\nSeverity: %s\n", emailData.Notification.Severity))
	body.WriteString(fmt.Sprintf("Priority: %s\n", emailData.Notification.Priority))
	body.WriteString(fmt.Sprintf("Time: %s\n", emailData.Notification.Timestamp.Format("2006-01-02 15:04:05")))

	if len(emailData.Links) > 0 {
		body.WriteString("\nLinks:\n")
		for _, link := range emailData.Links {
			body.WriteString(fmt.Sprintf("  %s: %s\n", link.Text, link.URL))
		}
	}

	body.WriteString("\n---\n")
	body.WriteString("Generated by GoFortress Coverage System\n")
	body.WriteString(fmt.Sprintf("Generated at: %s\n", emailData.GeneratedAt))

	return body.String()
}

// buildSimpleHTML builds a simple HTML email
func (e *EmailChannel) buildSimpleHTML(emailData *EmailData) string {
	var body strings.Builder

	body.WriteString("<!DOCTYPE html>\n")
	body.WriteString("<html>\n<head>\n")
	body.WriteString("<meta charset=\"UTF-8\">\n")
	body.WriteString("<style>\n")
	body.WriteString("  body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }\n")
	body.WriteString("  .header { background: #f4f4f4; padding: 20px; text-align: center; }\n")
	body.WriteString("  .content { padding: 20px; }\n")
	body.WriteString("  .coverage-info { background: #e8f4f8; padding: 15px; border-left: 4px solid #007cba; }\n")
	body.WriteString("  .links { margin-top: 20px; }\n")
	body.WriteString("  .link { display: inline-block; margin: 5px 10px 5px 0; }\n")
	body.WriteString("  .footer { background: #f4f4f4; padding: 10px; text-align: center; font-size: 12px; }\n")
	body.WriteString("  .severity-critical { color: #d73027; }\n")
	body.WriteString("  .severity-warning { color: #fc8d59; }\n")
	body.WriteString("  .severity-info { color: #4575b4; }\n")
	body.WriteString("</style>\n")
	body.WriteString("</head>\n<body>\n")

	// Header
	severityClass := fmt.Sprintf("severity-%s", emailData.Notification.Severity)
	body.WriteString("<div class=\"header\">\n")
	body.WriteString(fmt.Sprintf("  <h1 class=\"%s\">%s</h1>\n", severityClass, emailData.Subject))
	body.WriteString("</div>\n")

	// Content
	body.WriteString("<div class=\"content\">\n")
	body.WriteString(fmt.Sprintf("  <p><strong>%s</strong></p>\n", emailData.Notification.Message))

	if emailData.Repository != "" || emailData.Branch != "" || emailData.Author != "" {
		body.WriteString("  <table>\n")

		if emailData.Repository != "" {
			body.WriteString(fmt.Sprintf("    <tr><td><strong>Repository:</strong></td><td>%s</td></tr>\n", emailData.Repository))
		}

		if emailData.Branch != "" {
			body.WriteString(fmt.Sprintf("    <tr><td><strong>Branch:</strong></td><td>%s</td></tr>\n", emailData.Branch))
		}

		if emailData.Author != "" {
			body.WriteString(fmt.Sprintf("    <tr><td><strong>Author:</strong></td><td>%s</td></tr>\n", emailData.Author))
		}

		body.WriteString("  </table>\n")
	}

	// Coverage information
	if emailData.Coverage != nil {
		body.WriteString("  <div class=\"coverage-info\">\n")
		body.WriteString("    <h3>Coverage Information</h3>\n")
		body.WriteString(fmt.Sprintf("    <p><strong>Current Coverage:</strong> %.1f%%</p>\n", emailData.Coverage.Current))

		if emailData.Coverage.Previous > 0 {
			body.WriteString(fmt.Sprintf("    <p><strong>Previous Coverage:</strong> %.1f%%</p>\n", emailData.Coverage.Previous))
			body.WriteString(fmt.Sprintf("    <p><strong>Change:</strong> %+.1f%% %s</p>\n", emailData.Coverage.Change, emailData.Coverage.ChangeIcon))
		}

		if emailData.Coverage.Target > 0 {
			body.WriteString(fmt.Sprintf("    <p><strong>Target:</strong> %.1f%%</p>\n", emailData.Coverage.Target))
			body.WriteString(fmt.Sprintf("    <p><strong>Status:</strong> %s</p>\n", emailData.Coverage.Status))
		}

		body.WriteString("  </div>\n")
	}

	// Links
	if len(emailData.Links) > 0 {
		body.WriteString("  <div class=\"links\">\n")
		body.WriteString("    <h3>Links</h3>\n")
		for _, link := range emailData.Links {
			body.WriteString(fmt.Sprintf("    <a href=\"%s\" class=\"link\">%s</a>\n", link.URL, link.Text))
		}
		body.WriteString("  </div>\n")
	}

	body.WriteString("</div>\n")

	// Footer
	body.WriteString("<div class=\"footer\">\n")
	body.WriteString("  <p>Generated by GoFortress Coverage System</p>\n")
	body.WriteString(fmt.Sprintf("  <p>Generated at: %s</p>\n", emailData.GeneratedAt))
	body.WriteString("</div>\n")

	body.WriteString("</body>\n</html>\n")

	return body.String()
}

// sendSMTP sends the email via SMTP
func (e *EmailChannel) sendSMTP(ctx context.Context, message string) error {
	// Connect to SMTP server
	addr := fmt.Sprintf("%s:%d", e.config.SMTPHost, e.config.SMTPPort)

	var client *smtp.Client
	var err error

	if e.config.UseTLS {
		// Use TLS connection
		tlsConfig := &tls.Config{
			ServerName: e.config.SMTPHost,
			MinVersion: tls.VersionTLS12,
		}

		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("failed to connect with TLS: %w", err)
		}
		defer func() { _ = conn.Close() }()

		client, err = smtp.NewClient(conn, e.config.SMTPHost)
		if err != nil {
			return fmt.Errorf("failed to create SMTP client: %w", err)
		}
	} else {
		// Use plain connection
		client, err = smtp.Dial(addr)
		if err != nil {
			return fmt.Errorf("failed to connect to SMTP server: %w", err)
		}
	}
	defer func() { _ = client.Close() }()

	// Authenticate if credentials provided
	if e.auth != nil {
		if err := client.Auth(e.auth); err != nil {
			return fmt.Errorf("SMTP authentication failed: %w", err)
		}
	}

	// Set sender
	if err := client.Mail(e.config.FromEmail); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipients
	for _, addr := range e.config.ToEmails {
		if err := client.Rcpt(addr); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", addr, err)
		}
	}

	// Send message
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to start data transfer: %w", err)
	}
	defer func() { _ = writer.Close() }()

	_, err = writer.Write([]byte(message))
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

// initializeTemplates initializes email templates
func (e *EmailChannel) initializeTemplates() *EmailTemplates {
	templates := &EmailTemplates{}

	// Initialize plain text template
	plainTextTemplate := `Coverage Notification: {{.Subject}}
{{repeat "=" 50}}

{{.Notification.Message}}

{{if .Repository}}Repository: {{.Repository}}{{end}}
{{if .Branch}}Branch: {{.Branch}}{{end}}
{{if .Author}}Author: {{.Author}}{{end}}

{{if .Coverage}}
Coverage Information:
  Current: {{printf "%.1f%%" .Coverage.Current}}
  {{if gt .Coverage.Previous 0}}Previous: {{printf "%.1f%%" .Coverage.Previous}}{{end}}
  {{if ne .Coverage.Change 0}}Change: {{printf "%+.1f%%" .Coverage.Change}} {{.Coverage.ChangeIcon}}{{end}}
  {{if gt .Coverage.Target 0}}Target: {{printf "%.1f%%" .Coverage.Target}}{{end}}
  {{if ne .Coverage.Status ""}}Status: {{.Coverage.Status}}{{end}}
{{end}}

Severity: {{.Notification.Severity}}
Priority: {{.Notification.Priority}}
Time: {{.Notification.Timestamp.Format "2006-01-02 15:04:05"}}

{{if .Links}}
Links:
{{range .Links}}  {{.Text}}: {{.URL}}
{{end}}{{end}}

---
Generated by GoFortress Coverage System
Generated at: {{.GeneratedAt}}`

	templates.PlainText, _ = template.New("plaintext").Funcs(template.FuncMap{
		"repeat": func(s string, n int) string {
			return strings.Repeat(s, n)
		},
		"printf": fmt.Sprintf,
	}).Parse(plainTextTemplate)

	return templates
}

// Helper methods

func (e *EmailChannel) getCoverageChangeIcon(change float64) string {
	if change > 0 {
		return "↗"
	} else if change < 0 {
		return "↘"
	}
	return "→"
}

func (e *EmailChannel) getCoverageStatus(current, target float64) string {
	if target <= 0 {
		return "No target set"
	}

	if current >= target {
		return "Target met"
	} else if current >= target*0.9 {
		return "Close to target"
	} else if current >= target*0.8 {
		return "Below target"
	} else {
		return "Well below target"
	}
}

func (e *EmailChannel) getSeverityPrefix(severity types.SeverityLevel) string {
	switch severity {
	case types.SeverityInfo:
		return "[INFO]"
	case types.SeverityWarning:
		return "[WARNING]"
	case types.SeverityCritical:
		return "[CRITICAL]"
	case types.SeverityEmergency:
		return "[EMERGENCY]"
	default:
		return "[NOTIFICATION]"
	}
}

func (e *EmailChannel) getPriorityLevel(priority types.Priority) string {
	switch priority {
	case types.PriorityLow:
		return "3"
	case types.PriorityNormal:
		return "2"
	case types.PriorityHigh:
		return "1"
	case types.PriorityUrgent:
		return "1"
	default:
		return "2"
	}
}

// isValidEmail validates an email address
func isValidEmail(email string) bool {
	// Simple email validation
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}

	if len(parts[0]) == 0 || len(parts[1]) == 0 {
		return false
	}

	if !strings.Contains(parts[1], ".") {
		return false
	}

	return true
}
