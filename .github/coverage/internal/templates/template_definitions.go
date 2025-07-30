// Package templates provides template definitions for PR comments
package templates

// ComprehensiveTemplateDebug returns the comprehensive template for debugging
func ComprehensiveTemplateDebug() string {
	return comprehensiveTemplate
}

// Comprehensive template - detailed coverage report with all features
const comprehensiveTemplate = `<!-- {{ .Metadata.Signature }} -->
<!-- metadata: {"version":"{{ .Metadata.Version }}","generated_at":"{{ .Metadata.GeneratedAt.Format "2006-01-02T15:04:05Z07:00" }}","template":"{{ .Metadata.TemplateUsed }}"} -->

# {{ trendEmoji .Coverage.Summary.Direction }} Coverage Report

{{ statusEmoji .Coverage.Overall.Status }} **Overall Coverage: {{ formatPercent .Coverage.Overall.Percentage }}** {{ gradeEmoji .Coverage.Overall.Grade }}

{{ if .Comparison.IsSignificant }}
{{ if isImproved .Comparison.Direction }}{{ trendEmoji "up" }} Coverage **improved** by {{ formatChange .Comparison.Change }} ({{ formatPercent .Comparison.BasePercentage }} → {{ formatPercent .Comparison.CurrentPercentage }}){{ else if isDegraded .Comparison.Direction }}{{ trendEmoji "down" }} Coverage **decreased** by {{ formatChange .Comparison.Change }} ({{ formatPercent .Comparison.BasePercentage }} → {{ formatPercent .Comparison.CurrentPercentage }}){{ else }}{{ trendEmoji "stable" }} Coverage remained **stable** at {{ formatPercent .Coverage.Overall.Percentage }}{{ end }}
{{ else }}
{{ trendEmoji "stable" }} Coverage remained stable with {{ formatChange .Comparison.Change }} change
{{ end }}

## 📊 Coverage Metrics

| Metric | Value | Grade | Trend |
|--------|-------|-------|--------|
| **Percentage** | {{ formatPercent .Coverage.Overall.Percentage }} | {{ formatGrade .Quality.CoverageGrade }} | {{ trendEmoji .Trends.Direction }} {{ .Trends.Direction }} |
| **Statements** | {{ formatNumber .Coverage.Overall.CoveredStatements }}/{{ formatNumber .Coverage.Overall.TotalStatements }} | {{ formatGrade .Quality.OverallGrade }} | {{ formatChange .Comparison.Change }} |
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
| {{ if $file.IsNew }}🆕{{ else if $file.IsModified }}📝{{ end }} ` + "`" + `{{ truncate $file.Filename 40 }}` + "`" + ` | {{ formatPercent $file.Percentage }} | {{ if $file.Change }}{{ formatChange $file.Change }}{{ else }}-{{ end }} | {{ riskEmoji $file.Risk }} {{ humanize $file.Status }} |
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
{{ if .Trends.Prediction }}
- **Prediction**: {{ formatPercent .Trends.Prediction }} ({{ round (mul .Trends.Confidence 100) }}% confidence)
{{ end }}
{{ if .Config.IncludeCharts }}
- **Trend**: {{ trendChart .Coverage.Overall.Percentage }}
{{ end }}
{{ end }}

## 🔗 Resources

{{ if .Resources.BadgeURL }}
![Coverage Badge]({{ .Resources.BadgeURL }})
{{ end }}

{{ if or .Resources.ReportURL .Resources.DashboardURL }}
- 📊 [Full Coverage Report]({{ if .Resources.ReportURL }}{{ .Resources.ReportURL }}{{ else }}{{ .Resources.DashboardURL }}{{ end }})
{{ end }}
{{ if .Resources.BadgeURL }}
- 🏷️ [Coverage Badge]({{ .Resources.BadgeURL }})
{{ end }}
{{ if and .PullRequest.Number .Resources.PRReportURL }}
- 🔀 [PR Coverage Report]({{ .Resources.PRReportURL }})
{{ end }}
{{ if and .PullRequest.Number .Resources.PRBadgeURL }}
- 🏷️ [PR Coverage Badge]({{ .Resources.PRBadgeURL }})
{{ end }}

---

{{ if .Config.CustomFooter }}
{{ .Config.CustomFooter }}
{{ else if .Config.BrandingEnabled }}
*Generated by [GoFortress Coverage](https://github.com/{{ .Repository.Owner }}/{{ .Repository.Name }}) 🤖*  
*Updated: {{ .Metadata.GeneratedAt.Format "2006-01-02 15:04:05 UTC" }}*
{{ else }}
*Coverage report generated at {{ .Metadata.GeneratedAt.Format "2006-01-02 15:04:05 UTC" }}*
{{ end }}`

// Compact template - brief coverage summary
const compactTemplate = `{{ if .Metadata.Signature }}<!-- {{ .Metadata.Signature }} -->{{ end }}
## {{ statusEmoji .Coverage.Overall.Status }} Coverage: {{ formatPercent .Coverage.Overall.Percentage }} {{ if .Comparison.IsSignificant }}({{ formatChange .Comparison.Change }}){{ end }}

{{ if .Comparison.IsSignificant }}
{{ if isImproved .Comparison.Direction }}{{ trendEmoji "up" }}{{ else if isDegraded .Comparison.Direction }}{{ trendEmoji "down" }}{{ else }}{{ trendEmoji "stable" }}{{ end }} {{ humanize .Comparison.Direction }} by {{ formatChange .Comparison.Change }}
{{ end }}

{{ if needsAttention .Coverage.Overall.Percentage }}
{{ riskEmoji "high" }} **Coverage below {{ formatPercent .Config.WarningThreshold }} threshold**
{{ end }}

{{ $criticalFiles := filterFiles .Coverage.Files }}
{{ if $criticalFiles }}
**{{ length $criticalFiles }} {{ pluralize (length $criticalFiles) "file" "files" }} with changes**
{{ end }}

{{ $highPriorityRecs := filterRecommendations .Recommendations }}
{{ if $highPriorityRecs }}
**{{ length $highPriorityRecs }} {{ pluralize (length $highPriorityRecs) "recommendation" "recommendations" }}** available
{{ end }}

{{ if .Config.BrandingEnabled }}
---
*[GoFortress Coverage](https://github.com/{{ .Repository.Owner }}/{{ .Repository.Name }}) • {{ .Metadata.GeneratedAt.Format "2006-01-02 15:04:05 UTC" }}*
{{ end }}`

// Detailed template - comprehensive analysis with deep insights
const detailedTemplate = `<!-- {{ .Metadata.Signature }} -->
<!-- metadata: {"version":"{{ .Metadata.Version }}","generated_at":"{{ .Metadata.GeneratedAt.Format "2006-01-02T15:04:05Z07:00" }}","template":"{{ .Metadata.TemplateUsed }}"} -->

{{ if .Config.CustomHeader }}
{{ .Config.CustomHeader }}
{{ end }}

# {{ trendEmoji .Coverage.Summary.Direction }} Comprehensive Coverage Analysis

{{ statusEmoji .Coverage.Overall.Status }} **{{ formatPercent .Coverage.Overall.Percentage }} Coverage** {{ gradeEmoji .Coverage.Overall.Grade }} | {{ formatGrade .Quality.OverallGrade }} | {{ riskEmoji .Quality.RiskLevel }} {{ humanize .Quality.RiskLevel }} Risk

---

## 📊 Executive Summary

{{ .Coverage.Summary.OverallImpact }}

{{ if .Coverage.Summary.KeyAchievements }}
### 🏆 Key Achievements
{{ range .Coverage.Summary.KeyAchievements }}
- ✅ {{ . }}
{{ end }}
{{ end }}

{{ if .Coverage.Summary.KeyConcerns }}
### 🚨 Key Concerns
{{ range .Coverage.Summary.KeyConcerns }}
- ⚠️ {{ . }}
{{ end }}
{{ end }}

---

## 📈 Coverage Metrics Deep Dive

### Overall Performance
{{ if .Config.IncludeProgressBars }}
{{ coverageBar .Coverage.Overall.Percentage }}
{{ end }}

| Dimension | Base | Current | Change | Grade |
|-----------|------|---------|---------|-------|
| **Percentage** | {{ formatPercent .Comparison.BasePercentage }} | {{ formatPercent .Comparison.CurrentPercentage }} | {{ formatChange .Comparison.Change }} | {{ formatGrade .Quality.CoverageGrade }} |
| **Statements** | {{ formatNumber .Coverage.Overall.TotalStatements }} | {{ formatNumber .Coverage.Overall.CoveredStatements }} | {{ if .Comparison.Change }}{{ formatChange .Comparison.Change }}{{ else }}±0{{ end }} | {{ formatGrade .Quality.TrendGrade }} |
| **Quality Score** | - | {{ round .Quality.Score }}/100 | - | {{ formatGrade .Quality.OverallGrade }} |

### Statistical Analysis
- **Direction**: {{ trendEmoji .Trends.Direction }} {{ humanize .Trends.Direction }}
- **Magnitude**: {{ humanize .Comparison.Magnitude }}
- **Volatility**: {{ round .Trends.Volatility }}%
- **Momentum**: {{ .Trends.Momentum }}

{{ if .Trends.Prediction }}
### 🔮 Predictive Analysis
Based on current trends, next coverage is predicted to be **{{ formatPercent .Trends.Prediction }}** with {{ round (mul .Trends.Confidence 100) }}% confidence.
{{ end }}

---

## 🗂️ Package Analysis

{{ $packages := filterPackages .Coverage.Packages }}
{{ if $packages }}
| Package | Coverage | Change | Files | Status |
|---------|----------|--------|-------|--------|
{{ range $pkg := $packages }}
| ` + "`" + `{{ $pkg.Package }}` + "`" + ` | {{ formatPercent $pkg.Percentage }} | {{ if $pkg.Change }}{{ formatChange $pkg.Change }}{{ else }}-{{ end }} | {{ $pkg.FileCount }} | {{ statusEmoji $pkg.Status }} {{ humanize $pkg.Status }} |
{{ end }}
{{ else }}
*No package-level changes detected*
{{ end }}

---

## 📄 File-Level Analysis

{{ $files := sortFilesByRisk (filterFiles .Coverage.Files) }}
{{ if $files }}
{{ if .Config.UseCollapsibleSections }}
<details>
<summary>{{ riskEmoji "medium" }} View detailed file analysis ({{ length $files }} files)</summary>
{{ end }}

### High-Impact Files
{{ range $file := slice $files 0 10 }}
{{ if or $file.IsNew (gt (abs $file.Change) 5.0) (lt $file.Percentage 50.0) }}
#### {{ if $file.IsNew }}🆕{{ else if eq $file.Risk "high" }}🚨{{ else if eq $file.Risk "medium" }}⚠️{{ else }}✅{{ end }} ` + "`" + `{{ $file.Filename }}` + "`" + `

- **Coverage**: {{ formatPercent $file.Percentage }} {{ if $file.Change }}({{ formatChange $file.Change }}){{ end }}
- **Risk Level**: {{ riskEmoji $file.Risk }} {{ humanize $file.Risk }}
- **Status**: {{ humanize $file.Status }}
{{ if $file.IsNew }}
- **Lines Added**: {{ $file.LinesAdded }}
{{ else if $file.IsModified }}
- **Lines Modified**: +{{ $file.LinesAdded }} -{{ $file.LinesRemoved }}
{{ end }}

{{ end }}
{{ end }}

### Complete File Listing
| File | Coverage | Change | Risk | Status |
|------|----------|--------|------|--------|
{{ range $file := $files }}
| {{ if $file.IsNew }}🆕{{ else if $file.IsModified }}📝{{ end }} ` + "`" + `{{ truncate $file.Filename 50 }}` + "`" + ` | {{ formatPercent $file.Percentage }} | {{ if $file.Change }}{{ formatChange $file.Change }}{{ else }}-{{ end }} | {{ riskEmoji $file.Risk }} | {{ humanize $file.Status }} |
{{ end }}

{{ if .Config.UseCollapsibleSections }}
</details>
{{ end }}
{{ else }}
*No significant file-level changes detected*
{{ end }}

---

## 🎯 Quality & Risk Assessment

### Overall Quality Profile
- **Grade**: {{ gradeEmoji .Quality.OverallGrade }} {{ .Quality.OverallGrade }}
- **Score**: {{ round .Quality.Score }}/100
- **Risk**: {{ riskEmoji .Quality.RiskLevel }} {{ humanize .Quality.RiskLevel }}

{{ if .Quality.Strengths }}
### ✅ Identified Strengths
{{ range $i, $strength := .Quality.Strengths }}
{{ add $i 1 }}. {{ $strength }}
{{ end }}
{{ end }}

{{ if .Quality.Weaknesses }}
### ⚠️ Areas Requiring Attention
{{ range $i, $weakness := .Quality.Weaknesses }}
{{ add $i 1 }}. {{ $weakness }}
{{ end }}
{{ end }}

---

## 💡 Strategic Recommendations

{{ $recommendations := filterRecommendations .Recommendations }}
{{ if $recommendations }}
{{ range $i, $rec := $recommendations }}
### {{ add $i 1 }}. {{ priorityEmoji $rec.Priority }} {{ $rec.Title }}

**Priority**: {{ humanize $rec.Priority }} | **Type**: {{ humanize $rec.Type }} | **Impact**: {{ humanize $rec.Impact }}

{{ $rec.Description }}

{{ if $rec.Actions }}
**Action Plan**:
{{ range $j, $action := $rec.Actions }}
{{ add $j 1 }}. {{ $action }}
{{ end }}
{{ end }}

{{ end }}
{{ else }}
✅ No specific recommendations at this time. Current coverage practices are satisfactory.
{{ end }}

---

## 🔗 Resources & Links

{{ if .Resources.BadgeURL }}
### Coverage Badge
![Coverage Badge]({{ .Resources.BadgeURL }})
{{ end }}

### Quick Links
{{ if or .Resources.ReportURL .Resources.DashboardURL }}
- 📊 **[Complete Coverage Dashboard]({{ if .Resources.DashboardURL }}{{ .Resources.DashboardURL }}{{ else }}{{ .Resources.ReportURL }}{{ end }})**
{{ end }}
{{ if .Resources.BadgeURL }}
- 🏷️ **[Coverage Badge]({{ .Resources.BadgeURL }})**
{{ end }}
{{ if and .PullRequest.Number .Resources.PRReportURL }}
- 🔀 **[PR-Specific Report]({{ .Resources.PRReportURL }})**
{{ end }}
{{ if and .PullRequest.Number .Resources.PRBadgeURL }}
- 🏷️ **[PR Coverage Badge]({{ .Resources.PRBadgeURL }})**
{{ end }}
{{ if .Resources.HistoricalURL }}
- 📈 **[Historical Trends]({{ .Resources.HistoricalURL }})**
{{ end }}

---

{{ if .Config.CustomFooter }}
{{ .Config.CustomFooter }}
{{ else if .Config.BrandingEnabled }}
*🤖 Powered by [GoFortress Coverage](https://github.com/{{ .Repository.Owner }}/{{ .Repository.Name }}) • Advanced Coverage Analytics*  
*Generated: {{ .Metadata.GeneratedAt.Format "2006-01-02 15:04:05 UTC" }} • Template: {{ .Metadata.TemplateUsed }}*
{{ else }}
*Detailed coverage analysis completed at {{ .Metadata.GeneratedAt.Format "2006-01-02 15:04:05 UTC" }}*
{{ end }}`

// Summary template - high-level overview
const summaryTemplate = `{{ if .Metadata.Signature }}<!-- {{ .Metadata.Signature }} -->{{ end }}
## {{ statusEmoji .Coverage.Overall.Status }} Coverage Summary

**{{ formatPercent .Coverage.Overall.Percentage }}** {{ gradeEmoji .Coverage.Overall.Grade }} {{ if .Comparison.IsSignificant }}({{ formatChange .Comparison.Change }}){{ end }}

{{ if .Coverage.Summary.OverallImpact }}
{{ .Coverage.Summary.OverallImpact }}
{{ end }}

### Key Metrics
- **Quality Grade**: {{ formatGrade .Quality.OverallGrade }}
- **Risk Level**: {{ riskEmoji .Quality.RiskLevel }} {{ humanize .Quality.RiskLevel }}
- **Trend**: {{ trendEmoji .Trends.Direction }} {{ humanize .Trends.Direction }}

{{ $files := filterFiles .Coverage.Files }}
{{ if $files }}
### Notable Changes
{{ $sortedFiles := sortByChange $files }}{{ range $file := slice $sortedFiles 0 3 }}
- ` + "`" + `{{ truncate $file.Filename 30 }}` + "`" + `: {{ formatPercent $file.Percentage }} {{ if $file.Change }}({{ formatChange $file.Change }}){{ end }}
{{ end }}
{{ end }}

{{ $recs := filterRecommendations .Recommendations }}
{{ if $recs }}
{{ $topRec := index $recs 0 }}
### {{ priorityEmoji $topRec.Priority }} Top Recommendation
{{ $topRec.Title }}: {{ $topRec.Description }}
{{ end }}

{{ if or .Resources.ReportURL .Resources.DashboardURL }}
📊 [View Full Report]({{ if .Resources.ReportURL }}{{ .Resources.ReportURL }}{{ else }}{{ .Resources.DashboardURL }}{{ end }})
{{ end }}

{{ if .Config.BrandingEnabled }}
---
*[GoFortress Coverage](https://github.com/{{ .Repository.Owner }}/{{ .Repository.Name }}) • {{ .Metadata.GeneratedAt.Format "2006-01-02 15:04:05 UTC" }}*
{{ end }}`

// Minimal template - essential information only
const minimalTemplate = `{{ if .Metadata.Signature }}<!-- {{ .Metadata.Signature }} -->{{ end }}
{{ statusEmoji .Coverage.Overall.Status }} **{{ formatPercent .Coverage.Overall.Percentage }}** {{ if .Comparison.IsSignificant }}({{ formatChange .Comparison.Change }}){{ end }}

{{ if needsAttention .Coverage.Overall.Percentage }}
{{ riskEmoji "high" }} Below {{ formatPercent .Config.WarningThreshold }} threshold
{{ end }}

{{ $recs := filterRecommendations .Recommendations }}
{{ if $recs }}
{{ $criticalRec := index $recs 0 }}
{{ priorityEmoji $criticalRec.Priority }} {{ $criticalRec.Title }}
{{ end }}

{{ if .Resources.ReportURL }}[Report]({{ .Resources.ReportURL }}){{ else if .Resources.DashboardURL }}[Dashboard]({{ .Resources.DashboardURL }}){{ end }}`
