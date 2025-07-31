package reporting

// markdownTemplate contains the Markdown report template
const markdownTemplate = `# Performance Analysis Report

**Report ID:** {{.ReportID}}  
**Generated:** {{.Timestamp.Format "2006-01-02 15:04:05"}}  
**Version:** {{.Version}}

## Executive Summary

This report summarizes the performance analysis results for go-broadcast.

- **Total Tests:** {{.TotalTests}}
- **Passed Tests:** {{.PassedTests}}
- **Failed Tests:** {{.FailedTests}}
{{- if .BaselineMetrics}}- **Compared Against:** Baseline from {{.Timestamp.Format "2006-01-02"}}{{- end}}

## System Information

- **Go Version:** {{.SystemInfo.GoVersion}}
- **Operating System:** {{.SystemInfo.GOOS}}
- **Architecture:** {{.SystemInfo.GOARCH}}
- **CPU Cores:** {{.SystemInfo.NumCPU}}
- **GOMAXPROCS:** {{.SystemInfo.GOMAXPROCS}}

## Performance Metrics

### Current Performance
{{- range $key, $value := .CurrentMetrics}}
- **{{$key | title}}:** {{$value | formatFloat}}
{{- end}}

{{- if .BaselineMetrics}}
### Performance Changes

{{- if .Improvements}}
#### Improvements ✅
{{- range $key, $value := .Improvements}}
- **{{$key | title}}:** {{$value | formatPercent}} improvement
{{- end}}
{{- end}}

{{- if .Regressions}}
#### Regressions ⚠️
{{- range $key, $value := .Regressions}}
- **{{$key | title}}:** {{$value | formatPercent}} regression
{{- end}}
{{- end}}
{{- end}}

## Test Results

{{- range .TestResults}}
### {{.Name}}
- **Status:** {{if .Success}}✅ PASS{{else}}❌ FAIL{{end}}
- **Duration:** {{.Duration}}
{{- if .Throughput}}- **Throughput:** {{.Throughput | formatFloat}} ops/sec{{- end}}
{{- if .MemoryUsed}}- **Memory Used:** {{.MemoryUsed}} MB{{- end}}
{{- if .Error}}- **Error:** {{.Error}}{{- end}}

{{- end}}

## Profiling Summary

{{- if .ProfileSummary.CPUProfile.Available}}
- **CPU Profile:** {{.ProfileSummary.CPUProfile.Size | formatBytes}} ({{.ProfileSummary.CPUProfile.Path}})
{{- end}}
{{- if .ProfileSummary.MemoryProfile.Available}}
- **Memory Profile:** {{.ProfileSummary.MemoryProfile.Size | formatBytes}} ({{.ProfileSummary.MemoryProfile.Path}})
{{- end}}
{{- if .ProfileSummary.GoroutineProfile.Available}}
- **Goroutine Profile:** {{.ProfileSummary.GoroutineProfile.Size | formatBytes}} ({{.ProfileSummary.GoroutineProfile.Path}})
{{- end}}
- **Total Profile Size:** {{.ProfileSummary.TotalProfileSize | formatBytes}}

## Recommendations

{{- if .Recommendations}}
{{- range .Recommendations}}
### {{.Title}}

**Priority:** {{.Priority | title}}  
**Category:** {{.Category}}

{{.Description}}

**Recommended Action:** {{.Action}}

**Impact:** {{.Impact}}

{{- if .Evidence}}
**Evidence:**
{{- range .Evidence}}
- {{.}}
{{- end}}
{{- end}}

{{- if .References}}
**References:**
{{- range .References}}
- {{.}}
{{- end}}
{{- end}}

---
{{- end}}
{{- else}}
No specific recommendations at this time. All performance metrics are within acceptable ranges.
{{- end}}

## Conclusion

{{- if and .Improvements (not .Regressions)}}
Performance has improved across all measured metrics. The system is performing better than baseline.
{{- else if and .Regressions (not .Improvements)}}
Performance has regressed in several areas. Investigation and optimization are recommended.
{{- else if and .Improvements .Regressions}}
Mixed performance results with both improvements and regressions. Review specific metrics for optimization opportunities.
{{- else}}
Performance is stable with no significant changes from baseline.
{{- end}}

{{- if gt .FailedTests 0}}
**Note:** {{.FailedTests}} performance test(s) failed. These failures should be investigated and resolved.
{{- end}}

---
🏰 *Powered by GoFortress Coverage*
`

// htmlTemplate contains the HTML report template
const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Performance Analysis Report - {{.ReportID}}</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
            background: #f5f5f5;
        }
        .header {
            background: #fff;
            padding: 30px;
            border-radius: 8px;
            margin-bottom: 20px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .header h1 {
            color: #2c3e50;
            margin: 0 0 10px 0;
        }
        .meta {
            color: #7f8c8d;
            font-size: 14px;
        }
        .section {
            background: #fff;
            padding: 25px;
            border-radius: 8px;
            margin-bottom: 20px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .section h2 {
            color: #2c3e50;
            border-bottom: 2px solid #3498db;
            padding-bottom: 10px;
            margin-top: 0;
        }
        .section h3 {
            color: #34495e;
            margin-top: 25px;
        }
        .metrics-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
            gap: 20px;
            margin: 20px 0;
        }
        .metric-card {
            background: #f8f9fa;
            padding: 15px;
            border-radius: 6px;
            border-left: 4px solid #3498db;
        }
        .metric-label {
            font-size: 12px;
            color: #7f8c8d;
            text-transform: uppercase;
            letter-spacing: 0.5px;
            margin-bottom: 5px;
        }
        .metric-value {
            font-size: 24px;
            font-weight: 600;
            color: #2c3e50;
        }
        .improvement {
            color: #27ae60;
        }
        .regression {
            color: #e74c3c;
        }
        .test-result {
            padding: 15px;
            margin: 10px 0;
            border-radius: 6px;
            border-left: 4px solid #95a5a6;
        }
        .test-pass {
            border-left-color: #27ae60;
            background: #d5edd5;
        }
        .test-fail {
            border-left-color: #e74c3c;
            background: #f2d7d5;
        }
        .recommendation {
            padding: 20px;
            margin: 15px 0;
            border-radius: 6px;
            border-left: 4px solid #95a5a6;
        }
        .priority-high {
            border-left-color: #e74c3c;
            background: #fdf2f2;
        }
        .priority-medium {
            border-left-color: #f39c12;
            background: #fef9e7;
        }
        .priority-low {
            border-left-color: #3498db;
            background: #ebf3fd;
        }
        .recommendation h4 {
            margin: 0 0 10px 0;
            color: #2c3e50;
        }
        .priority-badge {
            display: inline-block;
            padding: 4px 8px;
            border-radius: 12px;
            font-size: 12px;
            font-weight: 600;
            text-transform: uppercase;
            margin-left: 10px;
        }
        .priority-high .priority-badge {
            background: #e74c3c;
            color: white;
        }
        .priority-medium .priority-badge {
            background: #f39c12;
            color: white;
        }
        .priority-low .priority-badge {
            background: #3498db;
            color: white;
        }
        .evidence-list {
            background: #f8f9fa;
            padding: 10px 15px;
            border-radius: 4px;
            margin: 10px 0;
        }
        .evidence-list ul {
            margin: 0;
            padding-left: 20px;
        }
        .system-info {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 15px;
        }
        .info-item {
            display: flex;
            justify-content: space-between;
            padding: 8px 0;
            border-bottom: 1px solid #ecf0f1;
        }
        .info-label {
            color: #7f8c8d;
            font-weight: 500;
        }
        .info-value {
            color: #2c3e50;
            font-weight: 600;
        }
        .status-indicator {
            display: inline-block;
            width: 12px;
            height: 12px;
            border-radius: 50%;
            margin-right: 8px;
        }
        .status-pass {
            background: #27ae60;
        }
        .status-fail {
            background: #e74c3c;
        }
        @media (max-width: 768px) {
            body {
                padding: 10px;
            }
            .metrics-grid {
                grid-template-columns: 1fr;
            }
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>Performance Analysis Report</h1>
        <div class="meta">
            <strong>Report ID:</strong> {{.ReportID}} |
            <strong>Generated:</strong> {{.Timestamp.Format "2006-01-02 15:04:05"}} |
            <strong>Version:</strong> {{.Version}}
        </div>
    </div>

    <div class="section">
        <h2>Executive Summary</h2>
        <div class="metrics-grid">
            <div class="metric-card">
                <div class="metric-label">Total Tests</div>
                <div class="metric-value">{{.TotalTests}}</div>
            </div>
            <div class="metric-card">
                <div class="metric-label">Passed Tests</div>
                <div class="metric-value improvement">{{.PassedTests}}</div>
            </div>
            <div class="metric-card">
                <div class="metric-label">Failed Tests</div>
                <div class="metric-value {{if gt .FailedTests 0}}regression{{end}}">{{.FailedTests}}</div>
            </div>
            {{if .BaselineMetrics}}
            <div class="metric-card">
                <div class="metric-label">Improvements</div>
                <div class="metric-value improvement">{{len .Improvements}}</div>
            </div>
            <div class="metric-card">
                <div class="metric-label">Regressions</div>
                <div class="metric-value {{if gt (len .Regressions) 0}}regression{{end}}">{{len .Regressions}}</div>
            </div>
            {{end}}
        </div>
    </div>

    <div class="section">
        <h2>System Information</h2>
        <div class="system-info">
            <div class="info-item">
                <span class="info-label">Go Version:</span>
                <span class="info-value">{{.SystemInfo.GoVersion}}</span>
            </div>
            <div class="info-item">
                <span class="info-label">Operating System:</span>
                <span class="info-value">{{.SystemInfo.GOOS}}</span>
            </div>
            <div class="info-item">
                <span class="info-label">Architecture:</span>
                <span class="info-value">{{.SystemInfo.GOARCH}}</span>
            </div>
            <div class="info-item">
                <span class="info-label">CPU Cores:</span>
                <span class="info-value">{{.SystemInfo.NumCPU}}</span>
            </div>
            <div class="info-item">
                <span class="info-label">GOMAXPROCS:</span>
                <span class="info-value">{{.SystemInfo.GOMAXPROCS}}</span>
            </div>
        </div>
    </div>

    <div class="section">
        <h2>Performance Metrics</h2>
        <h3>Current Performance</h3>
        <div class="metrics-grid">
            {{range $key, $value := .CurrentMetrics}}
            <div class="metric-card">
                <div class="metric-label">{{$key | title}}</div>
                <div class="metric-value">{{$value | formatFloat}}</div>
            </div>
            {{end}}
        </div>

        {{if .BaselineMetrics}}
        {{if .Improvements}}
        <h3>Improvements ✅</h3>
        <div class="metrics-grid">
            {{range $key, $value := .Improvements}}
            <div class="metric-card">
                <div class="metric-label">{{$key | title}}</div>
                <div class="metric-value improvement">{{$value | formatPercent}} improvement</div>
            </div>
            {{end}}
        </div>
        {{end}}

        {{if .Regressions}}
        <h3>Regressions ⚠️</h3>
        <div class="metrics-grid">
            {{range $key, $value := .Regressions}}
            <div class="metric-card">
                <div class="metric-label">{{$key | title}}</div>
                <div class="metric-value regression">{{$value | formatPercent}} regression</div>
            </div>
            {{end}}
        </div>
        {{end}}
        {{end}}
    </div>

    <div class="section">
        <h2>Test Results</h2>
        {{range .TestResults}}
        <div class="test-result {{if .Success}}test-pass{{else}}test-fail{{end}}">
            <h4>
                <span class="status-indicator {{if .Success}}status-pass{{else}}status-fail{{end}}"></span>
                {{.Name}}
            </h4>
            <p><strong>Duration:</strong> {{.Duration}}</p>
            {{if .Throughput}}<p><strong>Throughput:</strong> {{.Throughput | formatFloat}} ops/sec</p>{{end}}
            {{if .MemoryUsed}}<p><strong>Memory Used:</strong> {{.MemoryUsed}} MB</p>{{end}}
            {{if .Error}}<p><strong>Error:</strong> {{.Error}}</p>{{end}}
        </div>
        {{end}}
    </div>

    <div class="section">
        <h2>Profiling Summary</h2>
        <div class="metrics-grid">
            {{if .ProfileSummary.CPUProfile.Available}}
            <div class="metric-card">
                <div class="metric-label">CPU Profile</div>
                <div class="metric-value">{{.ProfileSummary.CPUProfile.Size | formatBytes}}</div>
            </div>
            {{end}}
            {{if .ProfileSummary.MemoryProfile.Available}}
            <div class="metric-card">
                <div class="metric-label">Memory Profile</div>
                <div class="metric-value">{{.ProfileSummary.MemoryProfile.Size | formatBytes}}</div>
            </div>
            {{end}}
            {{if .ProfileSummary.GoroutineProfile.Available}}
            <div class="metric-card">
                <div class="metric-label">Goroutine Profile</div>
                <div class="metric-value">{{.ProfileSummary.GoroutineProfile.Size | formatBytes}}</div>
            </div>
            {{end}}
            <div class="metric-card">
                <div class="metric-label">Total Profile Size</div>
                <div class="metric-value">{{.ProfileSummary.TotalProfileSize | formatBytes}}</div>
            </div>
        </div>
    </div>

    <div class="section">
        <h2>Recommendations</h2>
        {{if .Recommendations}}
        {{range .Recommendations}}
        <div class="recommendation {{.Priority | priorityClass}}">
            <h4>
                {{.Title}}
                <span class="priority-badge">{{.Priority}}</span>
            </h4>
            <p><strong>Category:</strong> {{.Category}}</p>
            <p>{{.Description}}</p>
            <p><strong>Recommended Action:</strong> {{.Action}}</p>
            <p><strong>Impact:</strong> {{.Impact}}</p>
            {{if .Evidence}}
            <div class="evidence-list">
                <strong>Evidence:</strong>
                <ul>
                {{range .Evidence}}
                <li>{{.}}</li>
                {{end}}
                </ul>
            </div>
            {{end}}
            {{if .References}}
            <div class="evidence-list">
                <strong>References:</strong>
                <ul>
                {{range .References}}
                <li>{{.}}</li>
                {{end}}
                </ul>
            </div>
            {{end}}
        </div>
        {{end}}
        {{else}}
        <p>No specific recommendations at this time. All performance metrics are within acceptable ranges.</p>
        {{end}}
    </div>

    <div class="section">
        <h2>Conclusion</h2>
        <p>
        {{if and .Improvements (not .Regressions)}}
        Performance has improved across all measured metrics. The system is performing better than baseline.
        {{else if and .Regressions (not .Improvements)}}
        Performance has regressed in several areas. Investigation and optimization are recommended.
        {{else if and .Improvements .Regressions}}
        Mixed performance results with both improvements and regressions. Review specific metrics for optimization opportunities.
        {{else}}
        Performance is stable with no significant changes from baseline.
        {{end}}
        </p>
        
        {{if gt .FailedTests 0}}
        <p><strong>Note:</strong> {{.FailedTests}} performance test(s) failed. These failures should be investigated and resolved.</p>
        {{end}}
    </div>

    <footer style="text-align: center; margin-top: 40px; color: #7f8c8d; font-size: 14px; display: flex; justify-content: center; align-items: center; gap: 0.5rem;">
        <span style="font-size: 1.1rem;">🏰</span>
        <span>Powered by <a href="https://github.com/mrz1836/go-broadcast" target="_blank" style="color: #7f8c8d; text-decoration: none;">GoFortress Coverage</a></span>
    </footer>
</body>
</html>
`
