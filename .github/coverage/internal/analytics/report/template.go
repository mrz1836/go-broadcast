package report

import (
	"github.com/mrz1836/go-broadcast/coverage/internal/templates"
)

// getReportTemplate returns the embedded coverage report HTML template (this IS A Coverage Report) (this is NOT a Dashboard)
//
//nolint:misspell // British spelling for "cancelled"
func getReportTemplate() string {
	return `<!DOCTYPE html>
<html lang="en" data-theme="auto">
` + templates.GetSharedHead("{{- if .Title}}{{.Title}}{{else}}{{.RepositoryOwner}}/{{.RepositoryName}} Coverage Report{{end -}}", "Detailed coverage analysis for {{.RepositoryOwner}}/{{.RepositoryName}}") + `
<body>
    <!-- Navigation Header -->
    <nav class="nav-header">
        <div class="nav-container">
            <a href="https://{{.RepositoryOwner}}.github.io/{{.RepositoryName}}/" class="nav-title-link">
                <div class="nav-title">{{.RepositoryName}}</div>
            </a>
            <div class="nav-actions">
                <div class="search-box">
                    <span class="search-icon">🔍</span>
                    <input type="text" class="search-input" placeholder="Search packages and files..." id="searchInput">
                </div>
                <div class="theme-toggle" onclick="toggleTheme()" aria-label="Toggle theme">
                    <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
                        <path d="M12 18c-3.3 0-6-2.7-6-6s2.7-6 6-6 6 2.7 6 6-2.7 6-6 6z"/>
                    </svg>
                </div>
            </div>
        </div>
    </nav>

    <!-- Header Section -->
    <header class="header">
        <div class="header-content">
            <div class="header-main">
                <h1>{{- if .PRNumber}}PR #{{.PRNumber}} {{end -}}Coverage Report</h1>
                <p class="subtitle">
                    {{- if .ProjectName}}
                        {{.ProjectName}} •
                    {{else}}
                        {{.RepositoryOwner}}/{{.RepositoryName}} •
                    {{end -}}
                    Detailed coverage analysis • <span class="dynamic-timestamp" data-timestamp="{{.GeneratedAt.Format "2006-01-02T15:04:05Z07:00"}}">Generated {{.GeneratedAt.Format "2006-01-02 15:04:05 UTC"}}</span>
                </p>
            </div>

            <div class="header-stats">
                <div class="coverage-circle {{- if ge .Summary.TotalPercentage 95.0}} excellent{{else if ge .Summary.TotalPercentage 85.0}} success{{else if ge .Summary.TotalPercentage 75.0}} warning{{else if ge .Summary.TotalPercentage 65.0}} low{{else}} danger{{end -}}">
                    <svg viewBox="0 0 100 100">
                        <circle cx="50" cy="50" r="45" fill="none" stroke="currentColor" stroke-width="6" opacity="0.2"/>
                        <circle cx="50" cy="50" r="45" fill="none" stroke="currentColor" stroke-width="6"
                                class="circle-progress"
                                stroke-dasharray="{{.Summary.TotalPercentage | multiply 2.827}} 282.7"
                                stroke-dashoffset="0"
                                transform="rotate(-90 50 50)"/>
                    </svg>
                    <div class="coverage-percentage {{- if ge .Summary.TotalPercentage 95.0}} excellent{{else if ge .Summary.TotalPercentage 85.0}} success{{else if ge .Summary.TotalPercentage 75.0}} warning{{else if ge .Summary.TotalPercentage 65.0}} low{{else}} danger{{end -}}">{{.Summary.TotalPercentage | printf "%.1f"}}%</div>
                </div>
                <div class="header-metrics">
                    <div class="metric">
                        <span class="metric-label">Lines Covered</span>
                        <span class="metric-value">{{.Summary.CoveredLines | commas}} / {{.Summary.TotalLines | commas}}</span>
                    </div>
                    <div class="metric">
                        <span class="metric-label">Files</span>
                        <span class="metric-value">{{.Summary.FileCount}}</span>
                    </div>
                    <div class="metric">
                        <span class="metric-label">Packages</span>
                        <span class="metric-value">{{.Summary.PackageCount}}</span>
                    </div>
                </div>
            </div>
        </div>

        <!-- Repository Info Bar -->
        <div class="repo-info">
            {{- if and .RepositoryOwner .RepositoryName}}
            <a href="https://github.com/{{.RepositoryOwner}}/{{.RepositoryName}}" class="repo-link">
                <span class="repo-icon">📦</span>
                {{.RepositoryOwner}}/{{.RepositoryName}}
            </a>
            {{- else}}
            <span class="repo-link">
                <span class="repo-icon">📦</span>
                {{.RepositoryOwner}}/{{.RepositoryName}}
            </span>
            {{- end}}

            {{- if .BranchName}}
            <span class="repo-separator">•</span>
            <span class="branch-info">
                <span class="branch-icon">🌿</span>
                {{.BranchName}}
            </span>
            {{- end}}

            {{- if .PRNumber}}
            <span class="repo-separator">•</span>
            {{- if .PRURL}}
            <a href="{{.PRURL}}" class="commit-link">
                <span class="commit-icon">🔀</span>
                PR #{{.PRNumber}}
            </a>
            {{- else}}
            <span class="commit-link">
                <span class="commit-icon">🔀</span>
                PR #{{.PRNumber}}
            </span>
            {{- end}}
            {{- end}}

            {{- if .CommitSHA}}
            <span class="repo-separator">•</span>
            {{- if .CommitURL}}
            <a href="{{.CommitURL}}" class="commit-link">
                <span class="commit-icon">🔗</span>
                {{truncate .CommitSHA 7}}
            </a>
            {{- else}}
            <span class="commit-link">
                <span class="commit-icon">🔗</span>
                {{truncate .CommitSHA 7}}
            </span>
            {{- end}}
            {{- end}}

            <div class="repo-actions">
                {{- if .BadgeURL}}
                <button class="action-btn secondary small" onclick="copyBadgeURL('{{.BadgeURL}}')">
                    <span class="btn-icon">🏷️</span>
                    <span class="btn-text">Badge</span>
                </button>
                {{- end}}
                <button class="action-btn secondary small" onclick="window.location.reload()">
                    <span class="btn-icon">🔄</span>
                    <span class="btn-text">Refresh</span>
                </button>
            </div>
        </div>
    </header>

    <!-- Main Content -->
    <main class="main-content">
        <!-- Summary Section -->
        <section class="summary-section">
            <h2>Coverage Summary</h2>
            <div class="summary-grid">
                <div class="summary-card">
                    <h3>Overall Coverage</h3>
                    <div class="coverage-bar large">
                        <div class="coverage-fill {{- if ge .Summary.TotalPercentage 95.0}} excellent{{else if ge .Summary.TotalPercentage 85.0}} success{{else if ge .Summary.TotalPercentage 75.0}} warning{{else if ge .Summary.TotalPercentage 65.0}} low{{else}} danger{{end -}}"
                             style="width: {{.Summary.TotalPercentage}}%"></div>
                    </div>
                    <div class="coverage-stats">
                        <span class="coverage-value">{{.Summary.TotalPercentage | printf "%.1f"}}%</span>
                        <span class="coverage-label">{{.Summary.CoveredLines | commas}} of {{.Summary.TotalLines | commas}} lines</span>
                    </div>
                </div>

                {{- if .Summary.ChangeStatus}}
                <div class="summary-card">
                    <h3>Coverage Trend</h3>
                    <div class="trend-indicator {{.Summary.ChangeStatus}}">
                        {{- if eq .Summary.ChangeStatus "improved"}}
                        <span class="trend-icon">📈</span>
                        <span class="trend-text">Improved</span>
                        {{- else if eq .Summary.ChangeStatus "declined"}}
                        <span class="trend-icon">📉</span>
                        <span class="trend-text">Declined</span>
                        {{- else}}
                        <span class="trend-icon">➡️</span>
                        <span class="trend-text">Stable</span>
                        {{- end}}
                    </div>
                    {{- if .Summary.PreviousCoverage}}
                    <div class="trend-details">
                        Previous: {{.Summary.PreviousCoverage | printf "%.1f"}}%
                    </div>
                    {{- end}}
                </div>
                {{- end}}

                <div class="summary-card">
                    <h3>Package Distribution</h3>
                    <div class="distribution-chart">
                        <div class="chart-placeholder">
                            <span class="chart-icon">📊</span>
                            <span class="chart-text">{{.Summary.PackageCount}} packages</span>
                        </div>
                    </div>
                </div>
            </div>
        </section>

        <!-- Packages Section -->
        {{- if .Packages}}
        <section class="packages-section">
            <h2>Package Coverage</h2>
            <div class="packages-container">
                {{- range .Packages}}
                <div class="package-card" data-package="{{.Name}}">
                    <div class="package-header" onclick="togglePackage('{{.Name}}')">
                        <div class="package-info">
                            <span class="package-toggle">▶</span>
                            <span class="package-name">{{.Name}}</span>
                            <span class="package-stats">{{.CoveredLines}} / {{.TotalLines}} lines</span>
                        </div>
                        <div class="package-coverage">
                            <span class="coverage-percentage {{- if ge .Percentage 95.0}} excellent{{else if ge .Percentage 85.0}} success{{else if ge .Percentage 75.0}} warning{{else if ge .Percentage 65.0}} low{{else}} danger{{end -}}">
                                {{.Percentage | printf "%.1f"}}%
                            </span>
                            <div class="coverage-bar mini">
                                <div class="coverage-fill {{- if ge .Percentage 95.0}} excellent{{else if ge .Percentage 85.0}} success{{else if ge .Percentage 75.0}} warning{{else if ge .Percentage 65.0}} low{{else}} danger{{end -}}"
                                     style="width: {{.Percentage}}%"></div>
                            </div>
                        </div>
                    </div>

                    {{- if .Files}}
                    <div class="package-files" id="package-{{.Name}}" style="display: none;">
                        {{- range .Files}}
                        <div class="file-item">
                            <div class="file-info">
                                <span class="file-icon">📄</span>
                                {{- if .URL}}
                                <a href="{{.URL}}" class="file-name" target="_blank" rel="noopener noreferrer">{{.Name}}</a>
                                {{- else}}
                                <span class="file-name">{{.Name}}</span>
                                {{- end}}
                                <span class="file-stats">{{.CoveredLines}} / {{.TotalLines}} lines</span>
                            </div>
                            <div class="file-coverage">
                                <span class="coverage-percentage {{- if ge .Percentage 95.0}} excellent{{else if ge .Percentage 85.0}} success{{else if ge .Percentage 75.0}} warning{{else if ge .Percentage 65.0}} low{{else}} danger{{end -}}">
                                    {{.Percentage | printf "%.1f"}}%
                                </span>
                                <div class="coverage-bar mini">
                                    <div class="coverage-fill {{- if ge .Percentage 95.0}} excellent{{else if ge .Percentage 85.0}} success{{else if ge .Percentage 75.0}} warning{{else if ge .Percentage 65.0}} low{{else}} danger{{end -}}"
                                         style="width: {{.Percentage}}%"></div>
                                </div>
                            </div>
                        </div>
                        {{- end}}
                    </div>
                    {{- end}}
                </div>
                {{- end}}
            </div>
        </section>
        {{- end}}
    </main>

` + templates.GetSharedFooter("", "GeneratedAt") + `

</body>
</html>`
}
