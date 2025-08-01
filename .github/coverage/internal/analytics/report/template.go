package report

// reportTemplate is the embedded coverage report HTML template (this IS A Coverage Report) (this is NOT a Dashboard)
//
//nolint:misspell // British spelling for "cancelled"
const reportTemplate = `<!DOCTYPE html>
<html lang="en" data-theme="auto">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{- if .Title}}{{.Title}}{{else}}{{.RepositoryOwner}}/{{.RepositoryName}} Coverage Report{{end -}}</title>
    <meta name="description" content="Detailed coverage analysis for {{.RepositoryOwner}}/{{.RepositoryName}}">

    <!-- Favicon -->
    <link rel="icon" type="image/x-icon" href="./assets/images/favicon.ico">
    <link rel="icon" type="image/svg+xml" href="./assets/images/favicon.svg">
    <link rel="shortcut icon" href="./assets/images/favicon.ico">
    <link rel="manifest" href="./assets/site.webmanifest">

    <!-- Preload critical resources -->
    <link rel="preconnect" href="https://fonts.googleapis.com" crossorigin>
    <link rel="preload" href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&family=JetBrains+Mono:wght@400;500&display=swap" as="style">
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&family=JetBrains+Mono:wght@400;500&display=swap" rel="stylesheet">
    
    <!-- Coverage styles -->
    <link rel="stylesheet" href="./assets/css/coverage.css">

    <!-- Meta tags for social sharing -->
    <meta property="og:title" content="{{.RepositoryOwner}}/{{.RepositoryName}} Coverage Report">
    <meta property="og:description" content="Code coverage analysis for {{.RepositoryOwner}}/{{.RepositoryName}}">
    <meta property="og:type" content="website">

    {{- if .GoogleAnalyticsID}}
    <!-- Google Analytics -->
    <script async src="https://www.googletagmanager.com/gtag/js?id={{.GoogleAnalyticsID}}"></script>
    <script>
      window.dataLayer = window.dataLayer || [];
      function gtag(){dataLayer.push(arguments);}
      gtag('js', new Date());
      gtag('config', '{{.GoogleAnalyticsID}}');
    </script>
    {{- end}}
</head>
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
                <h1>Coverage Report</h1>
                <p class="subtitle">
                    {{- if .ProjectName}}
                        {{.ProjectName}} • 
                    {{else}}
                        {{.RepositoryOwner}}/{{.RepositoryName}} • 
                    {{end -}}
                    Detailed coverage analysis • Generated {{.GeneratedAt.Format "2006-01-02 15:04:05 UTC"}}
                </p>
            </div>

            <div class="header-stats">
                <div class="coverage-circle {{- if ge .Summary.TotalPercentage 90.0}} success{{else if ge .Summary.TotalPercentage 80.0}} primary{{else if ge .Summary.TotalPercentage 60.0}} warning{{else}} danger{{end -}}">
                    <svg viewBox="0 0 100 100">
                        <circle cx="50" cy="50" r="45" fill="none" stroke="currentColor" stroke-width="6" opacity="0.2"/>
                        <circle cx="50" cy="50" r="45" fill="none" stroke="currentColor" stroke-width="6" 
                                stroke-dasharray="{{.Summary.TotalPercentage | multiply 2.827}} 282.7" 
                                stroke-dashoffset="0" 
                                transform="rotate(-90 50 50)"/>
                    </svg>
                    <div class="coverage-percentage">{{.Summary.TotalPercentage | printf "%.1f"}}%</div>
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
                        <div class="coverage-fill {{- if ge .Summary.TotalPercentage 90.0}} success{{else if ge .Summary.TotalPercentage 80.0}} primary{{else if ge .Summary.TotalPercentage 60.0}} warning{{else}} danger{{end -}}" 
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
                            <span class="coverage-percentage {{- if ge .Percentage 90.0}} success{{else if ge .Percentage 80.0}} primary{{else if ge .Percentage 60.0}} warning{{else}} danger{{end -}}">
                                {{.Percentage | printf "%.1f"}}%
                            </span>
                            <div class="coverage-bar mini">
                                <div class="coverage-fill {{- if ge .Percentage 90.0}} success{{else if ge .Percentage 80.0}} primary{{else if ge .Percentage 60.0}} warning{{else}} danger{{end -}}" 
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
                                <span class="file-name">{{.Name}}</span>
                                <span class="file-stats">{{.CoveredLines}} / {{.TotalLines}} lines</span>
                            </div>
                            <div class="file-coverage">
                                <span class="coverage-percentage {{- if ge .Percentage 90.0}} success{{else if ge .Percentage 80.0}} primary{{else if ge .Percentage 60.0}} warning{{else}} danger{{end -}}">
                                    {{.Percentage | printf "%.1f"}}%
                                </span>
                                <div class="coverage-bar mini">
                                    <div class="coverage-fill {{- if ge .Percentage 90.0}} success{{else if ge .Percentage 80.0}} primary{{else if ge .Percentage 60.0}} warning{{else}} danger{{end -}}" 
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

    <!-- Footer -->
    <footer class="footer">
        <div class="footer-content">
            <div class="footer-info">
                {{- if .LatestTag}}
                <div class="footer-version">
                    <span class="version-icon">🏷️</span>
                    <span class="version-text">{{.LatestTag}}</span>
                </div>
                <span class="footer-separator">•</span>
                {{- end}}
                <div class="footer-powered">
                    <span class="powered-text">Powered by</span>
                    <a href="https://github.com/{{.RepositoryOwner}}/{{.RepositoryName}}" class="gofortress-link">
                        <span class="fortress-icon">🏰</span>
                        <span class="fortress-text">GoFortress Coverage</span>
                    </a>
                </div>
                <span class="footer-separator">•</span>
                <div class="footer-timestamp">
                    <span class="timestamp-icon">🕐</span>
                    <span class="timestamp-text">{{.GeneratedAt.Format "2006-01-02 15:04:05 UTC"}}</span>
                </div>
            </div>
        </div>
    </footer>

    <script>
        // Theme management
        function toggleTheme() {
            const html = document.documentElement;
            const currentTheme = html.getAttribute('data-theme');
            const newTheme = currentTheme === 'light' ? 'dark' : 'light';
            html.setAttribute('data-theme', newTheme);
            localStorage.setItem('theme', newTheme);
        }

        // Initialize theme
        const savedTheme = localStorage.getItem('theme');
        const systemPrefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
        const theme = savedTheme || (systemPrefersDark ? 'dark' : 'light');
        document.documentElement.setAttribute('data-theme', theme);

        // Package toggle
        function togglePackage(packageName) {
            const packageEl = document.getElementById('package-' + packageName);
            const toggleIcon = document.querySelector('[data-package="' + packageName + '"] .package-toggle');
            
            if (packageEl.style.display === 'none' || !packageEl.style.display) {
                packageEl.style.display = 'block';
                toggleIcon.textContent = '▼';
            } else {
                packageEl.style.display = 'none';
                toggleIcon.textContent = '▶';
            }
        }

        // Search functionality
        const searchInput = document.getElementById('searchInput');
        searchInput.addEventListener('input', function(e) {
            const searchTerm = e.target.value.toLowerCase();
            const packages = document.querySelectorAll('.package-card');
            
            packages.forEach(pkg => {
                const packageName = pkg.querySelector('.package-name').textContent.toLowerCase();
                const files = pkg.querySelectorAll('.file-item');
                let hasMatch = packageName.includes(searchTerm);
                
                files.forEach(file => {
                    const fileName = file.querySelector('.file-name').textContent.toLowerCase();
                    if (fileName.includes(searchTerm)) {
                        hasMatch = true;
                        file.style.display = 'flex';
                    } else if (searchTerm) {
                        file.style.display = 'none';
                    } else {
                        file.style.display = 'flex';
                    }
                });
                
                pkg.style.display = hasMatch || !searchTerm ? 'block' : 'none';
                
                // Auto-expand packages with matching files
                if (hasMatch && searchTerm) {
                    const filesContainer = pkg.querySelector('.package-files');
                    if (filesContainer && filesContainer.style.display === 'none') {
                        togglePackage(pkg.dataset.package);
                    }
                }
            });
        });

        // Copy badge URL
        function copyBadgeURL(url) {
            navigator.clipboard.writeText(url).then(() => {
                const btn = event.target.closest('button');
                const originalText = btn.querySelector('.btn-text').textContent;
                btn.querySelector('.btn-text').textContent = 'Copied!';
                setTimeout(() => {
                    btn.querySelector('.btn-text').textContent = originalText;
                }, 2000);
            });
        }
    </script>
</body>
</html>`
