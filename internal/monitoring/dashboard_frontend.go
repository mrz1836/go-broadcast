package monitoring

// dashboardHTML contains the main dashboard HTML template
const dashboardHTML = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>go-broadcast Performance Dashboard</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    <link rel="stylesheet" href="/dashboard.css">
</head>
<body>
    <div class="container">
        <header>
            <h1>go-broadcast Performance Dashboard</h1>
            <div class="status-bar">
                <span id="status">●</span>
                <span id="last-update">Last updated: --</span>
                <span id="uptime">Uptime: --</span>
            </div>
        </header>

        <div class="metrics-grid">
            <!-- Real-time Stats -->
            <div class="card">
                <h3>Real-time Stats</h3>
                <div class="stats-grid">
                    <div class="stat">
                        <span class="label">Memory Usage</span>
                        <span class="value" id="memory-usage">-- MB</span>
                    </div>
                    <div class="stat">
                        <span class="label">Goroutines</span>
                        <span class="value" id="goroutines">--</span>
                    </div>
                    <div class="stat">
                        <span class="label">GC Count</span>
                        <span class="value" id="gc-count">--</span>
                    </div>
                    <div class="stat">
                        <span class="label">Heap Objects</span>
                        <span class="value" id="heap-objects">--</span>
                    </div>
                </div>
            </div>

            <!-- Memory Chart -->
            <div class="card chart-card">
                <h3>Memory Usage Over Time</h3>
                <canvas id="memoryChart"></canvas>
            </div>

            <!-- Goroutines Chart -->
            <div class="card chart-card">
                <h3>Goroutines Over Time</h3>
                <canvas id="goroutinesChart"></canvas>
            </div>

            <!-- GC Activity Chart -->
            <div class="card chart-card">
                <h3>Garbage Collection Activity</h3>
                <canvas id="gcChart"></canvas>
            </div>

            <!-- System Info -->
            <div class="card">
                <h3>System Information</h3>
                <div class="info-grid">
                    <div class="info-item">
                        <span class="info-label">Go Version:</span>
                        <span class="info-value" id="go-version">--</span>
                    </div>
                    <div class="info-item">
                        <span class="info-label">CPU Cores:</span>
                        <span class="info-value" id="cpu-cores">--</span>
                    </div>
                    <div class="info-item">
                        <span class="info-label">GOMAXPROCS:</span>
                        <span class="info-value" id="gomaxprocs">--</span>
                    </div>
                    <div class="info-item">
                        <span class="info-label">Total Allocated:</span>
                        <span class="info-value" id="total-alloc">-- MB</span>
                    </div>
                    <div class="info-item">
                        <span class="info-label">System Memory:</span>
                        <span class="info-value" id="sys-memory">-- MB</span>
                    </div>
                    <div class="info-item">
                        <span class="info-label">Next GC:</span>
                        <span class="info-value" id="next-gc">-- MB</span>
                    </div>
                </div>
            </div>

            <!-- Performance Alerts -->
            <div class="card">
                <h3>Performance Alerts</h3>
                <div id="alerts-container">
                    <div class="alert-item">
                        <span class="alert-status">●</span>
                        <span class="alert-message">System monitoring active</span>
                    </div>
                </div>
            </div>
        </div>
    </div>

    <!-- Footer -->
    <footer style="text-align: center; margin-top: 3rem; padding: 2rem 0; color: #7f8c8d; font-size: 0.875rem; border-top: 1px solid #ecf0f1; display: flex; justify-content: center; align-items: center; gap: 0.5rem;">
        <span style="font-size: 1.1rem;">🏰</span>
        <span>Powered by <a href="https://github.com/mrz1836/go-broadcast" target="_blank" style="color: #7f8c8d; text-decoration: none; transition: color 0.2s ease;" onmouseover="this.style.color='#3498db'" onmouseout="this.style.color='#7f8c8d'">GoFortress Coverage</a></span>
    </footer>

    <script src="/dashboard.js"></script>
</body>
</html>
`

// dashboardCSS contains the dashboard styling
const dashboardCSS = `
* {
    margin: 0;
    padding: 0;
    box-sizing: border-box;
}

body {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
    background: #f5f5f5;
    color: #333;
}

.container {
    max-width: 1400px;
    margin: 0 auto;
    padding: 20px;
}

header {
    background: #fff;
    padding: 20px;
    border-radius: 8px;
    margin-bottom: 20px;
    display: flex;
    justify-content: space-between;
    align-items: center;
    box-shadow: 0 2px 4px rgba(0,0,0,0.1);
}

header h1 {
    color: #2c3e50;
    font-size: 24px;
}

.status-bar {
    display: flex;
    gap: 20px;
    align-items: center;
    font-size: 14px;
}

#status {
    color: #27ae60;
    font-size: 16px;
}

.status-bar span {
    color: #7f8c8d;
}

.metrics-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
    gap: 20px;
}

.card {
    background: #fff;
    padding: 20px;
    border-radius: 8px;
    box-shadow: 0 2px 4px rgba(0,0,0,0.1);
}

.card h3 {
    margin-bottom: 15px;
    color: #2c3e50;
    font-size: 18px;
}

.chart-card {
    min-height: 300px;
}

.stats-grid {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 15px;
}

.stat {
    display: flex;
    flex-direction: column;
    padding: 10px;
    background: #f8f9fa;
    border-radius: 4px;
}

.stat .label {
    font-size: 12px;
    color: #7f8c8d;
    text-transform: uppercase;
    letter-spacing: 0.5px;
}

.stat .value {
    font-size: 24px;
    font-weight: 600;
    color: #2c3e50;
    margin-top: 5px;
}

.info-grid {
    display: flex;
    flex-direction: column;
    gap: 10px;
}

.info-item {
    display: flex;
    justify-content: space-between;
    padding: 8px 0;
    border-bottom: 1px solid #ecf0f1;
}

.info-item:last-child {
    border-bottom: none;
}

.info-label {
    color: #7f8c8d;
    font-weight: 500;
}

.info-value {
    color: #2c3e50;
    font-weight: 600;
}

#alerts-container {
    max-height: 200px;
    overflow-y: auto;
}

.alert-item {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 8px 0;
    border-bottom: 1px solid #ecf0f1;
}

.alert-item:last-child {
    border-bottom: none;
}

.alert-status {
    font-size: 12px;
}

.alert-status.good {
    color: #27ae60;
}

.alert-status.warning {
    color: #f39c12;
}

.alert-status.error {
    color: #e74c3c;
}

.alert-message {
    font-size: 14px;
    color: #2c3e50;
}

/* Responsive design */
@media (max-width: 768px) {
    .container {
        padding: 10px;
    }
    
    header {
        flex-direction: column;
        gap: 10px;
        text-align: center;
    }
    
    .status-bar {
        justify-content: center;
    }
    
    .metrics-grid {
        grid-template-columns: 1fr;
    }
    
    .stats-grid {
        grid-template-columns: 1fr;
    }
}
`

// dashboardJS contains the dashboard JavaScript functionality
const dashboardJS = `
class PerformanceDashboard {
    constructor() {
        this.charts = {};
        this.data = {
            memory: [],
            goroutines: [],
            gc: [],
            timestamps: []
        };
        this.maxDataPoints = 50;
        this.alerts = [];
        
        this.initCharts();
        this.startDataCollection();
    }

    initCharts() {
        // Memory Chart
        const memoryCtx = document.getElementById('memoryChart').getContext('2d');
        this.charts.memory = new Chart(memoryCtx, {
            type: 'line',
            data: {
                labels: [],
                datasets: [{
                    label: 'Heap Alloc (MB)',
                    data: [],
                    borderColor: '#3498db',
                    backgroundColor: 'rgba(52, 152, 219, 0.1)',
                    fill: true,
                    tension: 0.4
                }, {
                    label: 'System Memory (MB)',
                    data: [],
                    borderColor: '#e74c3c',
                    backgroundColor: 'rgba(231, 76, 60, 0.1)',
                    fill: false,
                    tension: 0.4
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                interaction: {
                    intersect: false,
                },
                scales: {
                    x: {
                        display: true,
                        title: {
                            display: true,
                            text: 'Time'
                        }
                    },
                    y: {
                        display: true,
                        title: {
                            display: true,
                            text: 'Memory (MB)'
                        }
                    }
                }
            }
        });

        // Goroutines Chart
        const goroutinesCtx = document.getElementById('goroutinesChart').getContext('2d');
        this.charts.goroutines = new Chart(goroutinesCtx, {
            type: 'line',
            data: {
                labels: [],
                datasets: [{
                    label: 'Active Goroutines',
                    data: [],
                    borderColor: '#2ecc71',
                    backgroundColor: 'rgba(46, 204, 113, 0.1)',
                    fill: true,
                    tension: 0.4
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                scales: {
                    x: {
                        display: true,
                        title: {
                            display: true,
                            text: 'Time'
                        }
                    },
                    y: {
                        display: true,
                        title: {
                            display: true,
                            text: 'Count'
                        }
                    }
                }
            }
        });

        // GC Chart
        const gcCtx = document.getElementById('gcChart').getContext('2d');
        this.charts.gc = new Chart(gcCtx, {
            type: 'bar',
            data: {
                labels: [],
                datasets: [{
                    label: 'GC Pause (ms)',
                    data: [],
                    backgroundColor: '#f39c12',
                    borderColor: '#e67e22',
                    borderWidth: 1
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                scales: {
                    x: {
                        display: true,
                        title: {
                            display: true,
                            text: 'Time'
                        }
                    },
                    y: {
                        display: true,
                        title: {
                            display: true,
                            text: 'Pause Time (ms)'
                        }
                    }
                }
            }
        });
    }

    async fetchMetrics() {
        try {
            const response = await fetch('/api/metrics');
            if (!response.ok) {
                throw new Error('Failed to fetch metrics');
            }
            return await response.json();
        } catch (error) {
            console.error('Error fetching metrics:', error);
            this.updateStatus('error');
            return null;
        }
    }

    updateCharts(metrics) {
        if (!metrics) return;

        const timestamp = new Date(metrics.timestamp * 1000);
        const timeLabel = timestamp.toLocaleTimeString();

        // Update data arrays
        this.data.timestamps.push(timeLabel);
        this.data.memory.push({
            heapAlloc: metrics.memory.alloc_mb,
            sysMem: metrics.memory.sys_mb
        });
        this.data.goroutines.push(metrics.runtime.goroutines);
        
        if (metrics.gc && metrics.gc.last_pause_ms !== undefined) {
            this.data.gc.push(metrics.gc.last_pause_ms);
        } else {
            this.data.gc.push(0);
        }

        // Limit data points
        if (this.data.timestamps.length > this.maxDataPoints) {
            this.data.timestamps.shift();
            this.data.memory.shift();
            this.data.goroutines.shift();
            this.data.gc.shift();
        }

        // Update charts
        this.charts.memory.data.labels = this.data.timestamps;
        this.charts.memory.data.datasets[0].data = this.data.memory.map(d => d.heapAlloc);
        this.charts.memory.data.datasets[1].data = this.data.memory.map(d => d.sysMem);
        this.charts.memory.update('none');

        this.charts.goroutines.data.labels = this.data.timestamps;
        this.charts.goroutines.data.datasets[0].data = this.data.goroutines;
        this.charts.goroutines.update('none');

        this.charts.gc.data.labels = this.data.timestamps;
        this.charts.gc.data.datasets[0].data = this.data.gc;
        this.charts.gc.update('none');
    }

    updateRealTimeStats(metrics) {
        if (!metrics) return;

        // Update real-time stats
        document.getElementById('memory-usage').textContent = 
            metrics.memory.alloc_mb.toFixed(1) + ' MB';
        document.getElementById('goroutines').textContent = 
            metrics.runtime.goroutines;
        document.getElementById('gc-count').textContent = 
            metrics.gc.num_gc || '--';
        document.getElementById('heap-objects').textContent = 
            (metrics.memory.heap_objects || 0).toLocaleString();

        // Update system info
        document.getElementById('go-version').textContent = 
            metrics.runtime.go_version || '--';
        document.getElementById('cpu-cores').textContent = 
            metrics.runtime.num_cpu || '--';
        document.getElementById('gomaxprocs').textContent = 
            metrics.runtime.gomaxprocs || '--';
        document.getElementById('total-alloc').textContent = 
            (metrics.memory.total_alloc_mb || 0).toFixed(1) + ' MB';
        document.getElementById('sys-memory').textContent = 
            (metrics.memory.sys_mb || 0).toFixed(1) + ' MB';
        document.getElementById('next-gc').textContent = 
            (metrics.memory.next_gc_mb || 0).toFixed(1) + ' MB';
    }

    checkAlerts(metrics) {
        if (!metrics) return;

        const newAlerts = [];

        // Memory usage alert
        if (metrics.memory.alloc_mb > 500) {
            newAlerts.push({
                type: 'warning',
                message: 'High memory usage: ' + metrics.memory.alloc_mb.toFixed(1) + ' MB'
            });
        }

        // Goroutine leak alert
        if (metrics.runtime.goroutines > 1000) {
            newAlerts.push({
                type: 'warning',
                message: 'High goroutine count: ' + metrics.runtime.goroutines
            });
        }

        // GC pressure alert
        if (metrics.gc && metrics.gc.avg_pause_ms > 10) {
            newAlerts.push({
                type: 'warning',
                message: 'High GC pause time: ' + metrics.gc.avg_pause_ms.toFixed(2) + ' ms'
            });
        }

        this.updateAlerts(newAlerts);
    }

    updateAlerts(newAlerts) {
        const container = document.getElementById('alerts-container');
        
        // Clear existing alerts
        container.innerHTML = '';

        if (newAlerts.length === 0) {
            container.innerHTML = '<div class="alert-item"><span class="alert-status good">●</span><span class="alert-message">All systems normal</span></div>';
        } else {
            newAlerts.forEach(alert => {
                const alertElement = document.createElement('div');
                alertElement.className = 'alert-item';
                alertElement.innerHTML = 
                    '<span class="alert-status ' + alert.type + '">●</span>' +
                    '<span class="alert-message">' + alert.message + '</span>';
                container.appendChild(alertElement);
            });
        }
    }

    updateStatus(status = 'connected') {
        const statusElement = document.getElementById('status');
        const lastUpdateElement = document.getElementById('last-update');

        switch (status) {
            case 'connected':
                statusElement.style.color = '#27ae60';
                statusElement.title = 'Connected';
                break;
            case 'error':
                statusElement.style.color = '#e74c3c';
                statusElement.title = 'Connection Error';
                break;
            default:
                statusElement.style.color = '#f39c12';
                statusElement.title = 'Connecting';
        }

        lastUpdateElement.textContent = 'Last updated: ' + new Date().toLocaleTimeString();
    }

    async updateUptime() {
        try {
            const response = await fetch('/api/health');
            if (response.ok) {
                const health = await response.json();
                const uptimeElement = document.getElementById('uptime');
                const uptimeSeconds = health.uptime;
                const hours = Math.floor(uptimeSeconds / 3600);
                const minutes = Math.floor((uptimeSeconds % 3600) / 60);
                const seconds = Math.floor(uptimeSeconds % 60);
                uptimeElement.textContent = 'Uptime: ' + hours + 'h ' + minutes + 'm ' + seconds + 's';
            }
        } catch (error) {
            console.error('Error fetching uptime:', error);
        }
    }

    startDataCollection() {
        // Initial fetch
        this.fetchMetrics().then(metrics => {
            if (metrics) {
                this.updateCharts(metrics);
                this.updateRealTimeStats(metrics);
                this.checkAlerts(metrics);
                this.updateStatus('connected');
            }
        });

        // Set up periodic updates
        setInterval(async () => {
            const metrics = await this.fetchMetrics();
            if (metrics) {
                this.updateCharts(metrics);
                this.updateRealTimeStats(metrics);
                this.checkAlerts(metrics);
                this.updateStatus('connected');
            }
        }, 1000);

        // Update uptime every 30 seconds
        setInterval(() => {
            this.updateUptime();
        }, 30000);

        // Initial uptime update
        this.updateUptime();
    }
}

// Initialize dashboard when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    new PerformanceDashboard();
});
`
