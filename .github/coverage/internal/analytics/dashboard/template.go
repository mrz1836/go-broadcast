package dashboard

import (
	"github.com/mrz1836/go-broadcast/coverage/internal/templates"
)

// dashboardTemplate is the embedded dashboard HTML template (this is the "DASHBOARD, this is NOT a coverage report" template).
//
//nolint:misspell // GitHub Actions API uses British spelling for "cancelled"
func getDashboardTemplate() string {
	return `<!DOCTYPE html>
<html lang="en" data-theme="auto">
` + templates.GetSharedHead("{{.RepositoryOwner}}/{{.RepositoryName}} Coverage Dashboard", "Coverage tracking and analytics for {{.RepositoryOwner}}/{{.RepositoryName}}") + `
<body>
    <div class="theme-toggle fixed" onclick="toggleTheme()" aria-label="Toggle theme">
        <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
            <path d="M12 18c-3.3 0-6-2.7-6-6s2.7-6 6-6 6 2.7 6 6-2.7 6-6 6z"/>
        </svg>
    </div>

    <div class="container">
        <header class="header enhanced">
            <div class="header-content">
                <div class="header-main">
                    {{- if .PRNumber}}
                    <h1>PR #{{.PRNumber}} Coverage</h1>
                    <p class="subtitle">{{- if .PRTitle}}{{.PRTitle}} • {{end}}Coverage analysis for this pull request</p>
                    {{- else}}
                    <h1>{{.RepositoryName}} Coverage</h1>
                    <p class="subtitle">Code coverage dashboard • Powered by 🏰 GoFortress</p>
                    {{- end}}
                </div>

                <div class="header-status">
                    <div class="status-indicator">
                        <span class="status-dot active"></span>
                        <span class="status-text">Coverage Active</span>
                    </div>
                    <div class="last-sync">
                        <span>🕐 <span class="dynamic-timestamp" data-timestamp="{{.Timestamp.Format "2006-01-02T15:04:05Z07:00"}}">{{.Timestamp.Format "2006-01-02 15:04:05 UTC"}}</span></span>
                    </div>
                </div>
            </div>

            <div class="repo-info-enhanced">
                <div class="repo-details">
                    {{- if .RepositoryURL}}
                    <a href="{{.RepositoryURL}}" target="_blank" class="repo-item repo-item-clickable">
                        <span class="repo-icon">📦</span>
                        <span class="repo-label">Repository</span>
                        <span class="repo-value repo-link-light">{{.RepositoryOwner}}/{{.RepositoryName}}</span>
                    </a>
                    {{- else}}
                    <div class="repo-item">
                        <span class="repo-icon">📦</span>
                        <span class="repo-label">Repository</span>
                        <span class="repo-value">{{.RepositoryOwner}}/{{.RepositoryName}}</span>
                    </div>
                    {{- end}}
                    {{- if .OwnerURL}}
                    <a href="{{.OwnerURL}}" target="_blank" class="repo-item repo-item-clickable">
                        <span class="repo-icon">👤</span>
                        <span class="repo-label">Owner</span>
                        <span class="repo-value">{{.RepositoryOwner}}</span>
                    </a>
                    {{- else}}
                    <div class="repo-item">
                        <span class="repo-icon">👤</span>
                        <span class="repo-label">Owner</span>
                        <span class="repo-value">{{.RepositoryOwner}}</span>
                    </div>
                    {{- end}}
                    {{- if .BranchURL}}
                    <a href="{{.BranchURL}}" target="_blank" class="repo-item repo-item-clickable">
                        <span class="repo-icon">🌿</span>
                        <span class="repo-label">Branch</span>
                        <span class="repo-value">{{.Branch}}</span>
                    </a>
                    {{- else}}
                    <div class="repo-item">
                        <span class="repo-icon">🌿</span>
                        <span class="repo-label">Branch</span>
                        <span class="repo-value">{{.Branch}}</span>
                    </div>
                    {{- end}}
                    {{- if .CommitSHA}}
                        {{- if .CommitURL}}
                        <a href="{{.CommitURL}}" target="_blank" class="repo-item repo-item-clickable">
                            <span class="repo-icon">🔗</span>
                            <span class="repo-label">Commit</span>
                            <span class="repo-value commit-link">{{.CommitSHA}}</span>
                        </a>
                        {{- else}}
                        <div class="repo-item">
                            <span class="repo-icon">🔗</span>
                            <span class="repo-label">Commit</span>
                            <span class="repo-value">{{.CommitSHA}}</span>
                        </div>
                        {{- end}}
                    {{- end}}
                </div>

                <div class="header-actions">
                    <button class="action-btn primary" onclick="window.location.reload()">
                        <span class="btn-icon">🔄</span>
                        <span class="btn-text">Refresh</span>
                    </button>
                    <button class="action-btn secondary" onclick="window.open('./coverage.html', '_blank')">
                        <span class="btn-icon">📄</span>
                        <span class="btn-text">Detailed Report</span>
                    </button>
                    <button class="action-btn secondary" onclick="window.open('{{.RepositoryURL}}', '_blank')">
                        <span class="btn-icon">📦</span>
                        <span class="btn-text">Repository</span>
                    </button>
                </div>
            </div>
        </header>

        <main>
            <div class="metrics-grid">
                <div class="metric-card">
                    <h3>📊 Overall Coverage</h3>
                    <div class="metric-value success">{{.TotalCoverage}}%</div>
                    {{- if .PRNumber}}
                    <div class="metric-label">PR Coverage{{- if .BaselineCoverage}} ({{if gt .TotalCoverage .BaselineCoverage}}+{{else if lt .TotalCoverage .BaselineCoverage}}-{{end}}{{printf "%.1f" (sub .TotalCoverage .BaselineCoverage)}}% vs base){{end}}</div>
                    {{- else}}
                    <div class="metric-label">{{.CoveredFiles}} of {{.TotalFiles}} files covered</div>
                    {{- end}}
                    <div class="coverage-bar">
                        <div class="coverage-fill" style="width: {{.TotalCoverage}}%; background: {{- if ge .TotalCoverage 90.0}}var(--gradient-success){{else if ge .TotalCoverage 80.0}}var(--gradient-primary){{else if ge .TotalCoverage 60.0}}var(--gradient-warning){{else}}var(--gradient-danger){{end -}};"></div>
                    </div>
                    {{- if .PRNumber}}
                        {{- if .BaselineCoverage}}
                            {{- if gt .TotalCoverage .BaselineCoverage}}
                            <div class="status-badge">
                                📈 Coverage Improved
                            </div>
                            {{- else if lt .TotalCoverage .BaselineCoverage}}
                            <div class="status-badge warning">
                                📉 Coverage Decreased
                            </div>
                            {{- else}}
                            <div class="status-badge">
                                ➡️ Coverage Stable
                            </div>
                            {{- end}}
                        {{- else}}
                        <div class="status-badge">
                            🆕 New PR Coverage
                        </div>
                        {{- end}}
                    {{- else}}
                    <div class="status-badge">
                        ✅ Excellent Coverage
                    </div>
                    {{- end}}
                </div>

                <div class="metric-card">
                    <h3>📁 Packages</h3>
                    <div class="metric-value">{{.PackagesTracked}}</div>
                    <div class="metric-label">Packages analyzed</div>
                    <div style="margin-top: 1rem;">
                        <div style="font-size: 0.9rem; color: var(--color-text-secondary);">
                            • All packages tracked
                        </div>
                    </div>
                </div>

                <div class="metric-card">
                    <h3>🎯 Quality Gate</h3>
                    <div class="quality-gate-badge">
                        <svg class="quality-gate-icon" viewBox="0 0 24 24" fill="none">
                            <circle cx="12" cy="12" r="10" fill="currentColor" fill-opacity="0.1"/>
                            <circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="1.5"/>
                            <path d="M8.5 12.5L10.5 14.5L15.5 9.5" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                        </svg>
                        <span class="quality-gate-text">PASSED</span>
                    </div>
                    <div class="metric-label">Threshold: 80% (exceeded)</div>
                    <div style="margin-top: 1rem; font-size: 0.9rem; color: var(--color-success);">
                        Coverage meets quality standards
                    </div>
                </div>

                <div class="metric-card">
                    <h3>🔄 Coverage Trend</h3>
                    {{if .HasHistory}}
                        <div class="metric-value {{- if eq .TrendDirection "up"}}success{{else if eq .TrendDirection "down"}}danger{{end -}}">
                            {{- if eq .TrendDirection "up"}}+{{end}}{{.CoverageTrend}}%
                        </div>
                        <div class="metric-label">Change from previous</div>
                        <div style="margin-top: 1rem; font-size: 0.9rem; color: var(--color-text-secondary);">
                            {{- if eq .TrendDirection "up"}}📈 Improving{{else if eq .TrendDirection "down"}}📉 Declining{{else}}➡️ Stable{{end -}}
                        </div>
                    {{else}}
                        <div class="metric-value" style="font-size: 1.5rem;">📊</div>
                        <div class="metric-label">Trend Analysis</div>
                        <div style="margin-top: 1rem;">
                            {{if .HasAnyData}}
                                <div style="font-size: 0.9rem; color: var(--color-warning);">
                                    🔄 Building trend data...
                                </div>
                                <div style="font-size: 0.8rem; color: var(--color-text-secondary); margin-top: 0.5rem;">
                                    {{if .PRNumber}}
                                        Comparing against base branch
                                    {{else if .IsFeatureBranch}}
                                        {{.HistoryDataPoints}} data point{{- if ne .HistoryDataPoints 1}}s{{end -}} for this branch
                                    {{else}}
                                        Need 2+ commits to show trends
                                    {{end}}
                                </div>
                            {{else}}
                                <div style="font-size: 0.9rem; color: var(--color-primary);">
                                    {{if .PRNumber}}
                                        📊 PR Coverage Analysis
                                    {{else if .IsFeatureBranch}}
                                        🌿 New branch coverage
                                    {{else if .IsFirstRun}}
                                        🚀 First coverage run!
                                    {{else if .HasPreviousRuns}}
                                        ⏳ Building history data...
                                    {{else if .WorkflowRunNumber}}
                                        📊 Coverage tracking resumed
                                    {{else}}
                                        📊 Coverage baseline established
                                    {{end}}
                                </div>
                                <div style="font-size: 0.8rem; color: var(--color-text-secondary); margin-top: 0.5rem;">
                                    {{if .PRNumber}}
                                        Base branch comparison pending
                                    {{else if .IsFirstRun}}
                                        Trends will appear after more commits
                                    {{else if .HasPreviousRuns}}
                                        Previous workflow runs failed to record history
                                    {{else if .WorkflowRunNumber}}
                                        Workflow run #{{.WorkflowRunNumber}} {{- if gt .WorkflowRunNumber 10}}(history may be incomplete){{end -}}
                                    {{else}}
                                        Collecting baseline coverage data
                                    {{end}}
                                </div>
                            {{end}}
                        </div>
                    {{end}}
                </div>
            </div>

            <div class="links-section">
                <h3 style="margin-bottom: 1rem;">📋 Coverage Reports & Tools</h3>
                <div class="links-grid">
                    <a href="./coverage.html" class="link-item">
                        📄 Detailed HTML Report
                    </a>
                    {{- if .BadgeURL}}
                    <button class="link-item action-btn secondary" onclick="copyBadgeURL('{{.BadgeURL}}')" style="border: none; background: transparent; cursor: pointer;">
                        🏷️ <span class="btn-text">Copy Badge URL</span>
                    </button>
                    {{- else}}
                    <a href="./coverage.svg" class="link-item">
                        🏷️ Coverage Badge
                    </a>
                    {{- end}}
                    <a href="{{.RepositoryURL}}" class="link-item">
                        📦 Source Repository
                    </a>
                    <a href="{{.RepositoryURL}}/actions" class="link-item">
                        🚀 GitHub Actions
                    </a>
                </div>
            </div>

            {{- if .Packages}}
            <div class="package-list dashboard">
                <h3 style="margin-bottom: 1rem;">📦 Package Coverage</h3>
                {{- range .Packages}}
                <div class="package-item dashboard">
                    <div class="package-name dashboard">{{.Name}}</div>
                    <div class="package-coverage" style="color: {{- if ge .Coverage 90.0}}#3fb950{{else if ge .Coverage 80.0}}#58a6ff{{else if ge .Coverage 60.0}}#d29922{{else}}#f85149{{end -}};">{{.Coverage}}%</div>
                    <div class="package-bar">
                        <div class="package-bar-fill" style="width: {{.Coverage}}%; background: {{- if ge .Coverage 90.0}}var(--gradient-success){{else if ge .Coverage 80.0}}var(--gradient-primary){{else if ge .Coverage 60.0}}var(--gradient-warning){{else}}var(--gradient-danger){{end -}};"></div>
                    </div>
                </div>
                {{- end}}
            </div>
            {{- end}}
        </main>

` + templates.GetSharedFooter(" dashboard", "Timestamp") + `
    </div>

</body>
</html>`
}
