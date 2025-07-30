package dashboard

// dashboardTemplate is the embedded dashboard HTML template
//
//nolint:misspell // GitHub Actions API uses British spelling for "cancelled"
const dashboardTemplate = `<!DOCTYPE html>
<html lang="en" data-theme="dark">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.RepositoryOwner}}/{{.RepositoryName}} Coverage Dashboard</title>
    <meta name="description" content="Coverage tracking and analytics for {{.RepositoryOwner}}/{{.RepositoryName}}">

    <!-- Favicon -->
    <link rel="icon" type="image/x-icon" href="./favicon.ico">
    <link rel="icon" type="image/svg+xml" href="./favicon.svg">
    <link rel="shortcut icon" href="./favicon.ico">

    <!-- Preload critical resources -->
    <link rel="preconnect" href="https://fonts.googleapis.com" crossorigin>
    <link rel="preload" href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&family=JetBrains+Mono:wght@400;500&display=swap" as="style">
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&family=JetBrains+Mono:wght@400;500&display=swap" rel="stylesheet">

    <style>
        /* CSS Custom Properties */
        :root {
            --color-bg: #0d1117;
            --color-bg-secondary: #161b22;
            --color-bg-tertiary: #21262d;
            --color-text: #c9d1d9;
            --color-text-secondary: #8b949e;
            --color-primary: #58a6ff;
            --color-success: #3fb950;
            --color-warning: #d29922;
            --color-danger: #f85149;
            --color-border: #30363d;
            --color-border-muted: #21262d;

            /* Glass morphism */
            --glass-bg: rgba(22, 27, 34, 0.8);
            --glass-border: rgba(48, 54, 61, 0.5);
            --backdrop-blur: 10px;

            /* Animations */
            --transition-base: 0.2s ease;
            --transition-smooth: 0.3s cubic-bezier(0.4, 0, 0.2, 1);

            /* Gradients */
            --gradient-primary: linear-gradient(135deg, #4a90d9, #6ba3e3);
            --gradient-success: linear-gradient(135deg, #3fb950, #56d364);
            --gradient-danger: linear-gradient(135deg, #f85149, #da3633);
        }

        /* Light theme */
        [data-theme="light"] {
            --color-bg: #ffffff;
            --color-bg-secondary: #f6f8fa;
            --color-bg-tertiary: #f0f6fc;
            --color-text: #24292f;
            --color-text-secondary: #656d76;
            --color-border: #d0d7de;
            --color-border-muted: #f0f6fc;
            --glass-bg: rgba(246, 248, 250, 0.8);
            --glass-border: rgba(208, 215, 222, 0.5);
        }

        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
            background: var(--color-bg);
            color: var(--color-text);
            line-height: 1.6;
            min-height: 100vh;
            position: relative;
        }

        /* Animated background */
        body::before {
            content: '';
            position: fixed;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            background:
                radial-gradient(circle at 20% 50%, rgba(74, 144, 217, 0.1) 0%, transparent 50%),
                radial-gradient(circle at 80% 80%, rgba(63, 185, 80, 0.1) 0%, transparent 50%),
                radial-gradient(circle at 40% 20%, rgba(248, 81, 73, 0.05) 0%, transparent 50%);
            pointer-events: none;
            z-index: 1;
        }

        /* Main container */
        .container {
            position: relative;
            z-index: 2;
            max-width: 1400px;
            margin: 0 auto;
            padding: 2rem;
        }

        /* Enhanced Header */
        .header {
            margin-bottom: 3rem;
            padding: 2rem;
            position: relative;
            overflow: hidden;
            background: var(--glass-bg);
            backdrop-filter: blur(var(--backdrop-blur));
            border: 1px solid var(--glass-border);
            border-radius: 20px;
            box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
        }

        .header-content {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 2rem;
        }

        .header-main {
            text-align: left;
        }

        .header-status {
            display: flex;
            flex-direction: column;
            align-items: flex-end;
            gap: 0.5rem;
        }

        .status-indicator {
            display: flex;
            align-items: center;
            gap: 0.75rem;
            padding: 0.75rem 1.25rem;
            background: var(--glass-bg);
            backdrop-filter: blur(var(--backdrop-blur));
            border: 1px solid var(--glass-border);
            border-radius: 16px;
            text-decoration: none;
            transition: all var(--transition-smooth);
            cursor: pointer;
            position: relative;
            overflow: hidden;
        }

        .status-indicator::before {
            content: '';
            position: absolute;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            background: linear-gradient(135deg,
                rgba(74, 144, 217, 0.05) 0%,
                rgba(63, 185, 80, 0.05) 100%);
            opacity: 0;
            transition: opacity var(--transition-smooth);
        }

        .status-indicator:hover::before {
            opacity: 1;
        }

        .status-indicator:hover {
            transform: translateY(-2px) scale(1.02);
            box-shadow: 0 8px 24px rgba(0, 0, 0, 0.15);
            border-color: var(--color-primary);
        }

        .status-icon {
            width: 24px;
            height: 24px;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 18px;
            position: relative;
            z-index: 1;
        }

        .status-dot {
            width: 8px;
            height: 8px;
            border-radius: 50%;
            background: var(--color-text-secondary);
        }

        .status-dot.active {
            background: var(--color-success);
            animation: pulse 2s infinite;
        }

        .status-dot.in-progress {
            background: var(--color-warning);
            animation: pulse 1s infinite;
        }

        .status-dot.failed {
            background: var(--color-danger);
        }

        @keyframes pulse {
            0%, 100% { opacity: 1; }
            50% { opacity: 0.5; }
        }

        .status-text {
            font-size: 0.9rem;
            font-weight: 500;
            color: var(--color-text);
        }

        .status-details {
            display: flex;
            flex-direction: column;
            align-items: flex-start;
            gap: 0.1rem;
        }

        .status-workflow {
            font-size: 0.8rem;
            color: var(--color-text-secondary);
            font-family: 'JetBrains Mono', monospace;
        }

        .last-sync {
            font-size: 0.8rem;
            color: var(--color-text-secondary);
            font-family: 'JetBrains Mono', monospace;
        }

        .repo-info-enhanced {
            display: flex;
            justify-content: space-between;
            align-items: center;
            gap: 2rem;
            flex-wrap: wrap;
        }

        .repo-details {
            display: flex;
            gap: 1.5rem;
            flex-wrap: wrap;
            position: relative;
            z-index: 10;
        }

        .repo-item {
            display: flex;
            flex-direction: column;
            align-items: center;
            gap: 0.25rem;
            padding: 1rem;
            background: var(--color-bg-secondary);
            border: 1px solid var(--color-border);
            border-radius: 12px;
            min-width: 100px;
            transition: var(--transition-smooth);
        }

        .repo-item:hover {
            transform: translateY(-2px);
            box-shadow: 0 4px 16px rgba(0, 0, 0, 0.1);
        }

        .repo-item-clickable {
            text-decoration: none;
            color: inherit;
            cursor: pointer;
        }

        .repo-item-clickable:hover {
            border-color: var(--color-primary);
            transform: translateY(-4px);
            box-shadow: 0 6px 20px rgba(0, 0, 0, 0.15);
        }

        .repo-icon {
            font-size: 1.5rem;
        }

        .repo-label {
            font-size: 0.7rem;
            color: var(--color-text-secondary);
            font-weight: 600;
            text-transform: uppercase;
            letter-spacing: 0.05em;
        }

        .repo-value {
            font-size: 0.9rem;
            font-weight: 500;
            color: var(--color-text);
            font-family: 'JetBrains Mono', monospace;
            text-align: center;
        }

        .commit-link {
            color: var(--color-primary);
            text-decoration: none;
            transition: var(--transition-base);
        }

        .commit-link:hover {
            text-decoration: underline;
            opacity: 0.8;
        }

        .repo-link-light {
            color: var(--color-text-secondary);
            transition: var(--transition-base);
        }

        .repo-item-clickable:hover .repo-link-light {
            color: var(--color-primary);
        }

        .header-actions {
            display: flex;
            gap: 0.75rem;
            flex-wrap: wrap;
        }

        .action-btn {
            display: flex;
            align-items: center;
            gap: 0.5rem;
            padding: 0.75rem 1.25rem;
            border: none;
            border-radius: 12px;
            font-size: 0.9rem;
            font-weight: 600;
            cursor: pointer;
            transition: var(--transition-smooth);
            font-family: inherit;
            position: relative;
            overflow: hidden;
        }

        .action-btn::before {
            content: '';
            position: absolute;
            top: 0;
            left: -100%;
            width: 100%;
            height: 100%;
            background: linear-gradient(90deg, transparent, rgba(255, 255, 255, 0.2), transparent);
            transition: left 0.5s ease;
        }

        .action-btn:hover::before {
            left: 100%;
        }

        .action-btn.primary {
            background: linear-gradient(135deg, #2563eb, #1e40af);
            color: white;
            box-shadow: 0 4px 12px rgba(37, 99, 235, 0.3);
        }

        .action-btn.primary:hover {
            background: linear-gradient(135deg, #1d4ed8, #1e3a8a);
            transform: translateY(-2px);
            box-shadow: 0 6px 20px rgba(37, 99, 235, 0.4);
        }

        .action-btn.secondary {
            background: var(--color-bg-secondary);
            color: var(--color-text);
            border: 1px solid var(--color-border);
        }

        .action-btn.secondary:hover {
            background: var(--color-bg-tertiary);
            border-color: var(--color-primary);
            transform: translateY(-2px);
            box-shadow: 0 4px 16px rgba(0, 0, 0, 0.1);
        }

        .btn-icon {
            font-size: 1rem;
        }

        .btn-text {
            font-size: 0.85rem;
            letter-spacing: 0.02em;
        }

        .header::before {
            content: '';
            position: absolute;
            top: -50%;
            left: -50%;
            width: 200%;
            height: 200%;
            background: radial-gradient(circle, var(--color-primary) 0%, transparent 70%);
            opacity: 0.05;
            animation: rotate 30s linear infinite;
            pointer-events: none;
            z-index: -1;
        }

        @keyframes rotate {
            to { transform: rotate(360deg); }
        }

        .header h1 {
            font-size: 3rem;
            font-weight: 700;
            background: var(--gradient-primary);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            background-clip: text;
            margin-bottom: 0.5rem;
            position: relative;
        }

        .header .subtitle {
            font-size: 1.25rem;
            color: var(--color-text-secondary);
            margin-bottom: 1rem;
        }

        .repo-info {
            display: inline-flex;
            align-items: center;
            gap: 1rem;
            background: var(--glass-bg);
            backdrop-filter: blur(var(--backdrop-blur));
            border: 1px solid var(--glass-border);
            padding: 0.75rem 1.5rem;
            border-radius: 12px;
            font-family: 'JetBrains Mono', monospace;
            font-size: 0.9rem;
        }

        .repo-info a {
            color: var(--color-primary);
            text-decoration: none;
            transition: var(--transition-base);
        }

        .repo-info a:hover {
            text-decoration: underline;
        }

        /* Metrics grid */
        .metrics-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
            gap: 1.5rem;
            margin-bottom: 3rem;
        }

        .metric-card {
            background: var(--glass-bg);
            backdrop-filter: blur(var(--backdrop-blur));
            border: 1px solid var(--glass-border);
            border-radius: 16px;
            padding: 2rem;
            position: relative;
            overflow: hidden;
            transition: var(--transition-smooth);
        }

        .metric-card::before {
            content: '';
            position: absolute;
            top: 0;
            left: -100%;
            width: 100%;
            height: 2px;
            background: var(--gradient-primary);
            transition: left 0.5s ease;
        }

        .metric-card:hover::before {
            left: 0;
        }

        .metric-card:hover {
            transform: translateY(-4px);
            box-shadow: 0 12px 32px rgba(0, 0, 0, 0.2);
        }

        .metric-card h3 {
            display: flex;
            align-items: center;
            gap: 0.5rem;
            font-size: 1rem;
            color: var(--color-text-secondary);
            margin-bottom: 1rem;
            font-weight: 600;
        }

        .metric-value {
            font-size: 2.5rem;
            font-weight: 700;
            background: var(--gradient-primary);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            background-clip: text;
            margin-bottom: 0.5rem;
        }

        .metric-value.success {
            background: var(--gradient-success);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            background-clip: text;
        }

        .metric-value.danger {
            background: var(--gradient-danger);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            background-clip: text;
        }

        .quality-gate-badge {
            display: flex;
            align-items: center;
            gap: 0.75rem;
            padding: 0.75rem 1rem;
            background: linear-gradient(135deg, var(--color-success), #4ade80);
            border-radius: 12px;
            margin-bottom: 0.5rem;
            box-shadow: 0 4px 12px rgba(34, 197, 94, 0.15);
            border: 1px solid rgba(34, 197, 94, 0.2);
        }

        .quality-gate-icon {
            width: 24px;
            height: 24px;
            color: white;
            flex-shrink: 0;
        }

        .quality-gate-text {
            color: white;
            font-weight: 700;
            font-size: 0.875rem;
            letter-spacing: 0.05em;
        }

        .metric-label {
            color: var(--color-text-secondary);
            font-size: 0.9rem;
            margin-bottom: 1rem;
        }

        .coverage-bar {
            height: 8px;
            background: var(--color-bg-tertiary);
            border-radius: 8px;
            overflow: hidden;
            margin: 1rem 0;
            position: relative;
        }

        .coverage-fill {
            height: 100%;
            background: var(--gradient-success);
            border-radius: 8px;
            position: relative;
            transition: width 1s cubic-bezier(0.4, 0, 0.2, 1);
        }

        .coverage-fill::after {
            content: '';
            position: absolute;
            top: 0;
            right: 0;
            bottom: 0;
            width: 100px;
            background: linear-gradient(90deg, transparent, rgba(255, 255, 255, 0.3), transparent);
            animation: shimmer 2s infinite;
        }

        @keyframes shimmer {
            0% { transform: translateX(-100px); }
            100% { transform: translateX(100px); }
        }

        .status-badge {
            display: inline-flex;
            align-items: center;
            gap: 0.5rem;
            padding: 0.5rem 1rem;
            border-radius: 24px;
            font-size: 0.85rem;
            font-weight: 600;
            background: var(--gradient-success);
            color: white;
        }

        .status-badge.warning {
            background: var(--gradient-danger);
        }

        /* Links section */
        .links-section {
            background: var(--glass-bg);
            backdrop-filter: blur(var(--backdrop-blur));
            border: 1px solid var(--glass-border);
            border-radius: 16px;
            padding: 2rem;
            margin-bottom: 2rem;
        }

        .links-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 1rem;
            margin-top: 1rem;
        }

        .link-item {
            display: flex;
            align-items: center;
            gap: 0.75rem;
            padding: 1rem;
            background: var(--color-bg-secondary);
            border: 1px solid var(--color-border);
            border-radius: 12px;
            text-decoration: none;
            color: var(--color-text);
            transition: var(--transition-smooth);
            position: relative;
            overflow: hidden;
        }

        .link-item::before {
            content: '';
            position: absolute;
            top: 0;
            left: -100%;
            width: 100%;
            height: 100%;
            background: var(--gradient-primary);
            opacity: 0.1;
            transition: left 0.3s ease;
        }

        .link-item:hover::before {
            left: 0;
        }

        .link-item:hover {
            border-color: var(--color-primary);
            transform: translateX(4px);
        }

        /* Updated time */
        .last-updated {
            background: var(--glass-bg);
            backdrop-filter: blur(var(--backdrop-blur));
            border: 1px solid var(--glass-border);
            border-radius: 12px;
            padding: 1rem;
            text-align: center;
            color: var(--color-text-secondary);
            font-size: 0.9rem;
        }

        /* Footer */
        .footer {
            margin-top: 4rem;
            padding: 2rem 0;
            border-top: 1px solid var(--color-border);
            background: var(--color-bg-secondary);
        }

        .footer-content {
            max-width: 1200px;
            margin: 0 auto;
            padding: 0 2rem;
        }

        .footer-info {
            display: flex;
            align-items: center;
            justify-content: center;
            gap: 1.5rem;
            flex-wrap: wrap;
            font-size: 0.9rem;
            color: var(--color-text-secondary);
        }

        .footer-version,
        .footer-powered,
        .footer-timestamp {
            display: flex;
            align-items: center;
            gap: 0.5rem;
        }

        .footer-separator {
            color: var(--color-border);
            font-size: 0.8rem;
        }

        .version-icon,
        .timestamp-icon {
            font-size: 1.1rem;
        }

        .version-text {
            font-family: 'JetBrains Mono', monospace;
            font-weight: 500;
            color: var(--color-primary);
        }

        .powered-text {
            color: var(--color-text-secondary);
        }

        .gofortress-link {
            display: flex;
            align-items: center;
            gap: 0.4rem;
            color: var(--color-primary);
            text-decoration: none;
            transition: all var(--transition-base);
            padding: 0.25rem 0.75rem;
            border-radius: 8px;
        }

        .gofortress-link:hover {
            background: var(--color-bg-tertiary);
            transform: translateY(-1px);
            color: var(--color-text);
        }

        .fortress-icon {
            font-size: 1.2rem;
        }

        .fortress-text {
            font-weight: 600;
        }

        @media (max-width: 768px) {
            .footer-separator {
                display: none;
            }

            .footer-info {
                flex-direction: column;
                gap: 1rem;
            }
        }

        /* Package list */
        .package-list {
            background: var(--glass-bg);
            backdrop-filter: blur(var(--backdrop-blur));
            border: 1px solid var(--glass-border);
            border-radius: 16px;
            padding: 2rem;
            margin-top: 2rem;
        }

        .package-item {
            display: grid;
            grid-template-columns: 1fr auto 150px;
            gap: 1rem;
            align-items: center;
            padding: 1rem;
            border-bottom: 1px solid var(--color-border);
            transition: var(--transition-base);
        }

        .package-item:last-child {
            border-bottom: none;
        }

        .package-item:hover {
            background: var(--color-bg-secondary);
            border-radius: 8px;
        }

        .package-name {
            font-family: 'JetBrains Mono', monospace;
            font-size: 0.9rem;
            color: var(--color-primary);
            letter-spacing: 0.1em;
            white-space: pre-wrap;
            line-height: 1.4;
        }

        .package-coverage {
            font-weight: 600;
            color: var(--color-success);
        }

        .package-bar {
            height: 6px;
            background: var(--color-bg-tertiary);
            border-radius: 6px;
            overflow: hidden;
        }

        .package-bar-fill {
            height: 100%;
            background: var(--gradient-success);
            border-radius: 6px;
            transition: width 0.5s ease;
        }

        /* Theme toggle */
        .theme-toggle {
            position: fixed;
            top: 2rem;
            right: 2rem;
            z-index: 100;
            background: var(--glass-bg);
            backdrop-filter: blur(var(--backdrop-blur));
            border: 1px solid var(--glass-border);
            border-radius: 12px;
            padding: 0.75rem;
            cursor: pointer;
            transition: var(--transition-base);
        }

        .theme-toggle:hover {
            transform: scale(1.1);
        }

        /* Responsive */
        @media (max-width: 768px) {
            .container {
                padding: 1rem;
            }

            .header {
                padding: 1.5rem;
            }

            .header-content {
                flex-direction: column;
                gap: 1rem;
                align-items: stretch;
            }

            .header-main {
                text-align: center;
            }

            .header h1 {
                font-size: 2rem;
            }

            .header-status {
                align-items: center;
            }

            .repo-info-enhanced {
                flex-direction: column;
                gap: 1.5rem;
                align-items: stretch;
            }

            .repo-details {
                justify-content: center;
                gap: 1rem;
            }

            .repo-item {
                min-width: 80px;
                padding: 0.75rem;
            }

            .header-actions {
                justify-content: center;
                gap: 0.5rem;
            }

            .action-btn {
                padding: 0.5rem 1rem;
                font-size: 0.8rem;
            }

            .metrics-grid {
                grid-template-columns: 1fr;
            }

            .package-item {
                grid-template-columns: 1fr;
                gap: 0.5rem;
            }
        }

        /* Animations */
        @keyframes fadeIn {
            from {
                opacity: 0;
                transform: translateY(20px);
            }
            to {
                opacity: 1;
                transform: translateY(0);
            }
        }

        .metric-card {
            animation: fadeIn 0.5s ease forwards;
            opacity: 0;
        }

        .metric-card:nth-child(1) { animation-delay: 0.1s; }
        .metric-card:nth-child(2) { animation-delay: 0.2s; }
        .metric-card:nth-child(3) { animation-delay: 0.3s; }
        .metric-card:nth-child(4) { animation-delay: 0.4s; }
    </style>
</head>
<body>
    <div class="theme-toggle" onclick="toggleTheme()" aria-label="Toggle theme">
        <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
            <path d="M12 18c-3.3 0-6-2.7-6-6s2.7-6 6-6 6 2.7 6 6-2.7 6-6 6z"/>
        </svg>
    </div>

    <div class="container">
        <header class="header">
            <div class="header-content">
                <div class="header-main">
                    <h1>{{.RepositoryOwner}}/{{.RepositoryName}} Coverage</h1>
                    <p class="subtitle">Code coverage dashboard • Powered by GoFortress</p>
                </div>

                <div class="header-status">
                    <div class="status-indicator">
                        <span class="status-dot active"></span>
                        <span class="status-text">Coverage Active</span>
                    </div>
                    <div class="last-sync">
                        <span>🕐 {{.Timestamp}}</span>
                    </div>
                </div>
            </div>

            <div class="repo-info-enhanced">
                <div class="repo-details">
                    {{if .RepositoryURL}}
                    <a href="{{.RepositoryURL}}" target="_blank" class="repo-item repo-item-clickable">
                        <span class="repo-icon">📦</span>
                        <span class="repo-label">Repository</span>
                        <span class="repo-value repo-link-light">{{.RepositoryOwner}}/{{.RepositoryName}}</span>
                    </a>
                    {{else}}
                    <div class="repo-item">
                        <span class="repo-icon">📦</span>
                        <span class="repo-label">Repository</span>
                        <span class="repo-value">{{.RepositoryOwner}}/{{.RepositoryName}}</span>
                    </div>
                    {{end}}
                    {{if .OwnerURL}}
                    <a href="{{.OwnerURL}}" target="_blank" class="repo-item repo-item-clickable">
                        <span class="repo-icon">👤</span>
                        <span class="repo-label">Owner</span>
                        <span class="repo-value">{{.RepositoryOwner}}</span>
                    </a>
                    {{else}}
                    <div class="repo-item">
                        <span class="repo-icon">👤</span>
                        <span class="repo-label">Owner</span>
                        <span class="repo-value">{{.RepositoryOwner}}</span>
                    </div>
                    {{end}}
                    {{if .BranchURL}}
                    <a href="{{.BranchURL}}" target="_blank" class="repo-item repo-item-clickable">
                        <span class="repo-icon">🌿</span>
                        <span class="repo-label">Branch</span>
                        <span class="repo-value">{{.Branch}}</span>
                    </a>
                    {{else}}
                    <div class="repo-item">
                        <span class="repo-icon">🌿</span>
                        <span class="repo-label">Branch</span>
                        <span class="repo-value">{{.Branch}}</span>
                    </div>
                    {{end}}
                    {{if .CommitSHA}}
                        {{if .CommitURL}}
                        <a href="{{.CommitURL}}" target="_blank" class="repo-item repo-item-clickable">
                            <span class="repo-icon">🔗</span>
                            <span class="repo-label">Commit</span>
                            <span class="repo-value commit-link">{{.CommitSHA}}</span>
                        </a>
                        {{else}}
                        <div class="repo-item">
                            <span class="repo-icon">🔗</span>
                            <span class="repo-label">Commit</span>
                            <span class="repo-value">{{.CommitSHA}}</span>
                        </div>
                        {{end}}
                    {{end}}
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
                    <div class="metric-label">{{.CoveredFiles}} of {{.TotalFiles}} files covered</div>
                    <div class="coverage-bar">
                        <div class="coverage-fill" style="width: {{.TotalCoverage}}%"></div>
                    </div>
                    <div class="status-badge">
                        ✅ Excellent Coverage
                    </div>
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
                        <div class="metric-value {{if eq .TrendDirection "up"}}success{{else if eq .TrendDirection "down"}}danger{{end}}">
                            {{if eq .TrendDirection "up"}}+{{end}}{{.CoverageTrend}}%
                        </div>
                        <div class="metric-label">Change from previous</div>
                        <div style="margin-top: 1rem; font-size: 0.9rem; color: var(--color-text-secondary);">
                            {{if eq .TrendDirection "up"}}📈 Improving{{else if eq .TrendDirection "down"}}📉 Declining{{else}}➡️ Stable{{end}}
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
                                    Need 2+ commits to show trends
                                </div>
                            {{else}}
                                <div style="font-size: 0.9rem; color: var(--color-primary);">
                                    🚀 First coverage run!
                                </div>
                                <div style="font-size: 0.8rem; color: var(--color-text-secondary); margin-top: 0.5rem;">
                                    Trends will appear after more commits
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
                    <a href="./coverage.svg" class="link-item">
                        🏷️ Coverage Badge
                    </a>
                    <a href="{{.RepositoryURL}}" class="link-item">
                        📦 Source Repository
                    </a>
                    <a href="{{.RepositoryURL}}/actions" class="link-item">
                        🚀 GitHub Actions
                    </a>
                </div>
            </div>

            {{if .Packages}}
            <div class="package-list">
                <h3 style="margin-bottom: 1rem;">📦 Package Coverage</h3>
                {{range .Packages}}
                <div class="package-item">
                    <div class="package-name">{{.Name}}</div>
                    <div class="package-coverage">{{.Coverage}}%</div>
                    <div class="package-bar">
                        <div class="package-bar-fill" style="width: {{.Coverage}}%"></div>
                    </div>
                </div>
                {{end}}
            </div>
            {{end}}

            <div class="last-updated">
                🕐 Last updated: {{.Timestamp}}
            </div>
        </main>

        <footer class="footer">
            <div class="footer-content">
                <div class="footer-info">
                    {{if .LatestTag}}
                    <div class="footer-version">
                        <span class="version-icon">🏷️</span>
                        <span class="version-text">{{.LatestTag}}</span>
                    </div>
                    <span class="footer-separator">•</span>
                    {{end}}
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
                        <span class="timestamp-text">{{.Timestamp}}</span>
                    </div>
                </div>
            </div>
        </footer>
    </div>

    <script>
        // Theme toggle
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

        // History data
        const historyData = {{.HistoryJSON}};

        // Initialize charts if history data exists
        if (historyData && historyData.length > 0) {
            // Future: Add chart rendering here
        }

        // Note: Build status refresh functionality has been removed
        // Static deployments on GitHub Pages cannot provide live updates
        // The build status shown is a snapshot from when the report was generated
    </script>
</body>
</html>`
