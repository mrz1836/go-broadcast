// Package templates provides template definitions for PR comments and shared HTML components
package templates

import (
	"fmt"
)

// Comprehensive template - detailed coverage report with all features
const comprehensiveTemplate = `[//]: # ({{ .Metadata.Signature }})
[//]: # (metadata: {"version":"{{ .Metadata.Version }}","generated_at":"{{ .Metadata.GeneratedAt.Format "2006-01-02T15:04:05Z07:00" }}","template":"{{ .Metadata.TemplateUsed }}"})

# {{ trendEmoji .Coverage.Summary.Direction }} Coverage Report

{{ statusEmoji .Coverage.Overall.Status }} **Overall Coverage: {{ formatPercent .Coverage.Overall.Percentage }}** {{ gradeEmoji .Coverage.Overall.Grade }}

{{- if .PRFiles -}}
    {{- if not .PRFiles.Summary.HasGoChanges -}}
{{ trendEmoji "stable" }} **No Go files modified in this PR**

Project coverage remains at {{ formatPercent .Coverage.Overall.Percentage }} ({{ formatNumber .Coverage.Overall.CoveredStatements }}/{{ formatNumber .Coverage.Overall.TotalStatements }} statements)

Changes: {{ .PRFiles.Summary.SummaryText }}
    {{- else -}}
        {{- if and (ne .Comparison.BasePercentage 0.0) (.Comparison.IsSignificant) -}}
            {{- if isImproved .Comparison.Direction -}}
{{ trendEmoji "up" }} Coverage **improved** by {{ formatChange .Comparison.Change }} ({{ formatPercent .Comparison.BasePercentage }} → {{ formatPercent .Comparison.CurrentPercentage }})
            {{- else if isDegraded .Comparison.Direction -}}
{{ trendEmoji "down" }} Coverage **decreased** by {{ formatChange .Comparison.Change }} ({{ formatPercent .Comparison.BasePercentage }} → {{ formatPercent .Comparison.CurrentPercentage }})
            {{- else -}}
{{ trendEmoji "stable" }} Coverage remained **stable** at {{ formatPercent .Coverage.Overall.Percentage }}
            {{- end -}}
        {{- else if eq .Comparison.BasePercentage 0.0 -}}
{{ trendEmoji "stable" }} **Initial coverage report** - no baseline available for comparison
        {{- else -}}
{{ trendEmoji "stable" }} Coverage remained stable with {{ formatChange .Comparison.Change }} change
        {{- end -}}
    {{- end -}}
{{- else -}}
    {{- if and (ne .Comparison.BasePercentage 0.0) (.Comparison.IsSignificant) -}}
        {{- if isImproved .Comparison.Direction -}}
{{ trendEmoji "up" }} Coverage **improved** by {{ formatChange .Comparison.Change }} ({{ formatPercent .Comparison.BasePercentage }} → {{ formatPercent .Comparison.CurrentPercentage }})
        {{- else if isDegraded .Comparison.Direction -}}
{{ trendEmoji "down" }} Coverage **decreased** by {{ formatChange .Comparison.Change }} ({{ formatPercent .Comparison.BasePercentage }} → {{ formatPercent .Comparison.CurrentPercentage }})
        {{- else -}}
{{ trendEmoji "stable" }} Coverage remained **stable** at {{ formatPercent .Coverage.Overall.Percentage }}
        {{- end -}}
    {{- else if eq .Comparison.BasePercentage 0.0 -}}
{{ trendEmoji "stable" }} **Initial coverage report** - no baseline available for comparison
    {{- else -}}
{{ trendEmoji "stable" }} Coverage remained stable with {{ formatChange .Comparison.Change }} change
    {{- end -}}
{{- end -}}

## 📊 Coverage Metrics

| Metric | Value | Grade | Trend |
|--------|-------|-------|--------|
| **Percentage** | {{ formatPercent .Coverage.Overall.Percentage }} | {{ formatGrade .Quality.CoverageGrade }} | {{ trendEmoji .Trends.Direction }} {{ .Trends.Direction }} |
| **Statements** | {{ formatNumber .Coverage.Overall.CoveredStatements }}/{{ formatNumber .Coverage.Overall.TotalStatements }} | {{ formatGrade .Quality.OverallGrade }} | {{ if .PRFiles }}{{ if not .PRFiles.Summary.HasGoChanges }}No change{{ else }}{{ if ne .Comparison.BasePercentage 0.0 }}{{ formatChange .Comparison.Change }}{{ else }}First report{{ end }}{{ end }}{{ else }}{{ if ne .Comparison.BasePercentage 0.0 }}{{ formatChange .Comparison.Change }}{{ else }}First report{{ end }}{{ end }} |
| **Quality Score** | {{ round .Quality.Score }}/100 | {{ formatGrade .Quality.OverallGrade }} | {{ if gt .Quality.Score 80.0 }}📈{{ else if lt .Quality.Score 60.0 }}📉{{ else }}📊{{ end }} |

{{ if .Config.IncludeProgressBars }}
### 📈 Coverage Breakdown

{{ coverageBar .Coverage.Overall.Percentage }}

{{ if .Coverage.Packages }}
**Top Packages:**
{{ $filteredPackages := filterPackages .Coverage.Packages }}{{ range $i, $pkg := slice $filteredPackages 0 5 }}
- ` + "`" + `{{ $pkg.Package }}` + "`" + `: {{ progressBar $pkg.Percentage 100.0 10 }} {{ if $pkg.Change }}({{ formatChange $pkg.Change }}){{ end }}
{{ end }}
{{ end }}
{{ end }}

{{ $significantFiles := filterFiles .Coverage.Files }}
{{ if $significantFiles }}
## 📁 File Changes ({{ length $significantFiles }})

{{ if .Config.UseCollapsibleSections }}
<details>
<summary>{{ riskEmoji "medium" }} View file coverage changes</summary>

{{ end }}
| File | Coverage | Change | Status |
|------|----------|--------|--------|
{{ $sortedFiles := sortByChange $significantFiles }}{{ range $file := slice $sortedFiles 0 .Config.MaxFileChanges }}
| {{- if $file.IsNew }}🆕{{- else if $file.IsModified }}📝{{- end }} ` + "`" + `{{ truncate $file.Filename 40 }}` + "`" + ` | {{ formatPercent $file.Percentage }} | {{- if $file.Change }}{{ formatChange $file.Change }}{{- else }}-{{- end }} | {{ riskEmoji $file.Risk }} {{ humanize $file.Status }} |
{{ end }}

{{ if .Config.UseCollapsibleSections }}
</details>
{{ end }}
{{ end }}

## 🎯 Quality Assessment

{{ gradeEmoji .Quality.OverallGrade }} **Overall Grade: {{ .Quality.OverallGrade }}** ({{ riskEmoji .Quality.RiskLevel }} {{ humanize .Quality.RiskLevel }} risk)

{{ if .Quality.Strengths }}
### ✅ Strengths
{{ range .Quality.Strengths }}
- {{ . }}
{{ end }}
{{ end }}

{{ if .Quality.Weaknesses }}
### ⚠️ Areas for Improvement
{{ range .Quality.Weaknesses }}
- {{ . }}
{{ end }}
{{ end }}

{{ $recommendations := filterRecommendations .Recommendations }}
{{ if $recommendations }}
## 💡 Recommendations

{{ range $rec := $recommendations }}
### {{ priorityEmoji $rec.Priority }} {{ $rec.Title }} **({{ humanize $rec.Priority }} priority)**

{{ $rec.Description }}

{{ if $rec.Actions }}
**Action Items:**
{{ range $rec.Actions }}
- [ ] {{ . }}
{{ end }}
{{ end }}

{{ end }}
{{ end }}

{{ if .Trends.Direction }}
## 📈 Trend Analysis

- **Direction**: {{ trendEmoji .Trends.Direction }} {{ humanize .Trends.Direction }}
- **Momentum**: {{ .Trends.Momentum }}
{{- if .Trends.Prediction }}
- **Prediction**: {{ formatPercent .Trends.Prediction }} ({{ round (mul .Trends.Confidence 100) }}% confidence)
{{- end }}
{{- if .Config.IncludeCharts }}
- **Trend**: {{ trendChart .Coverage.Overall.Percentage }}
{{- end }}
{{ end }}

## 🔗 Resources

{{ if .Resources.BadgeURL }}
![Coverage Badge]({{ .Resources.BadgeURL }})
{{ end }}

{{- if or .Resources.ReportURL .Resources.DashboardURL }}
- 📊 [Full Coverage Report]({{ if .Resources.ReportURL }}{{ .Resources.ReportURL }}{{ else }}{{ .Resources.DashboardURL }}{{ end }})
{{- end }}
{{- if .Resources.BadgeURL }}
- 🏷️ [Coverage Badge]({{ .Resources.BadgeURL }})
{{- end }}
{{- if and .PullRequest.Number .Resources.PRReportURL }}
- 🔀 [PR Coverage Report]({{ .Resources.PRReportURL }})
{{- end }}
{{- if and .PullRequest.Number .Resources.PRBadgeURL }}
- 🏷️ [PR Coverage Badge]({{ .Resources.PRBadgeURL }})
{{- end }}

---

{{ if .Config.CustomFooter }}
{{ .Config.CustomFooter }}
{{ else if .Config.BrandingEnabled }}
*Generated by 🏰 [GoFortress](https://github.com/{{ .Repository.Owner }}/{{ .Repository.Name }})*
*Updated: {{ .Metadata.GeneratedAt.Format "2006-01-02 15:04:05 UTC" }}*
{{ else }}
*Coverage report generated at {{ .Metadata.GeneratedAt.Format "2006-01-02 15:04:05 UTC" }}*
{{ end }}`

// GetSharedFooter returns the standardized footer HTML with configurable CSS class and timestamp field
// cssClass: pass " dashboard" for dashboard styling, or "" for regular styling
// timestampField: pass "Timestamp" or "GeneratedAt" for the appropriate timestamp field
func GetSharedFooter(cssClass, timestampField string) string {
	return fmt.Sprintf(`    <!-- Footer -->
    <footer class="footer">
        <div class="footer-content%s">
            <div class="footer-info">
                {{- if .LatestTag}}
                <div class="footer-version">
                    <a href="https://github.com/{{.RepositoryOwner}}/{{.RepositoryName}}/releases/tag/{{.LatestTag}}" target="_blank" class="version-link">
                        <span class="version-icon">🏷️</span>
                        <span class="version-text">{{.LatestTag}}</span>
                    </a>
                </div>
                <span class="footer-separator">•</span>
                {{- end}}
                <div class="footer-powered">
                    <span class="powered-text">Powered by</span>
                    <a href="https://github.com/{{.RepositoryOwner}}/{{.RepositoryName}}" target="_blank" class="gofortress-link">
                        <span class="fortress-icon">🏰</span>
                        <span class="fortress-text">GoFortress Coverage</span>
                    </a>
                </div>
                <span class="footer-separator">•</span>
                <div class="footer-timestamp">
                    <span class="timestamp-icon">🕐</span>
                    <span class="timestamp-text dynamic-timestamp" data-timestamp="{{.%s.Format "2006-01-02T15:04:05Z07:00"}}">Generated {{.%s.Format "2006-01-02 15:04:05 UTC"}}</span>
                </div>
            </div>
        </div>
    </footer>
    <script src="./assets/js/coverage-time.js"></script>
	<script src="./assets/js/theme.js"></script>`, cssClass, timestampField, timestampField)
}
