// Main Application Logic

function dashboardApp() {
    return {
        // State
        alerts: [],
        severityCounts: {
            critical: 0,
            high: 0,
            medium: 0,
            low: 0
        },
        wsConnected: false,
        filterSeverity: 'all',
        filterSource: 'all',
        filterStatus: 'all',
        autoRefresh: true,
        autoRefreshInterval: null,
        selectedAlert: null,
        showModal: false,
        toasts: [],
        wsManager: null,
        showNotifications: false,
        unreadCount: 0,
        maxUnreadCount: 50, // Cap unread notifications at 50
        readAlertIds: new Set(), // Track which alerts have been marked as read
        maxToasts: 3, // Maximum number of toast notifications to show at once

        // New Enhanced Metrics
        loading: false,
        totalAlerts: 0,
        activeAlertsCount: 0,
        systemHealthStatus: 'healthy',
        lastUpdated: new Date().toISOString(),
        avgResponseTime: 2.5, // Average alert detection time in seconds

        // Initialization
        async init() {
            console.log('Dashboard initializing...');

            // Load read alerts from localStorage
            this.loadReadAlertsFromStorage();

            // Fetch initial data
            await this.fetchAlerts();

            // Connect WebSocket
            this.connectWebSocket();

            // Start auto-refresh
            if (this.autoRefresh) {
                this.startAutoRefresh();
            }
        },

        // Load/Save read alerts to localStorage
        loadReadAlertsFromStorage() {
            try {
                const stored = localStorage.getItem('readAlertIds');
                if (stored) {
                    this.readAlertIds = new Set(JSON.parse(stored));
                    console.log(`Loaded ${this.readAlertIds.size} read alerts from storage`);
                }
            } catch (error) {
                console.error('Failed to load read alerts from storage:', error);
            }
        },

        saveReadAlertsToStorage() {
            try {
                localStorage.setItem('readAlertIds', JSON.stringify([...this.readAlertIds]));
            } catch (error) {
                console.error('Failed to save read alerts to storage:', error);
            }
        },

        // API Methods
        async fetchAlerts() {
            try {
                this.loading = true;

                // Fetch ALL data from API - no frontend computation
                const [alertsData, countData, activeCountData, severityData] = await Promise.all([
                    apiClient.getRecentAlerts(100),
                    apiClient.getAlertCount(),
                    apiClient.getActiveAlertCount(),
                    apiClient.getSeverityCounts()
                ]);

                this.alerts = alertsData.alerts || [];
                this.totalAlerts = countData.count || 0;
                this.activeAlertsCount = activeCountData.count || 0;
                this.severityCounts = severityData || { critical: 0, high: 0, medium: 0, low: 0 };

                // Count unread alerts (alerts not in readAlertIds set)
                this.unreadCount = this.alerts.filter(a =>
                    a.status === 'firing' && !this.readAlertIds.has(a.id)
                ).length;

                this.updateSystemHealth();
                this.lastUpdated = new Date().toISOString();
                console.log(`Loaded ${this.alerts.length} alerts, ${this.totalAlerts} total, ${this.activeAlertsCount} active, ${this.unreadCount} unread`);
            } catch (error) {
                console.error('Failed to fetch alerts:', error);
                this.showToast('Failed to load alerts', 'high');
                this.systemHealthStatus = 'degraded';
            } finally {
                this.loading = false;
            }
        },

        // Refresh ALL counts from API (used after WebSocket updates)
        async refreshCounts() {
            try {
                const [countData, activeCountData, severityData] = await Promise.all([
                    apiClient.getAlertCount(),
                    apiClient.getActiveAlertCount(),
                    apiClient.getSeverityCounts()
                ]);

                this.totalAlerts = countData.count || 0;
                this.activeAlertsCount = activeCountData.count || 0;
                this.severityCounts = severityData || { critical: 0, high: 0, medium: 0, low: 0 };
                this.updateSystemHealth();
                this.lastUpdated = new Date().toISOString();
            } catch (error) {
                console.error('Failed to refresh counts:', error);
            }
        },

        // WebSocket Methods
        connectWebSocket() {
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const wsUrl = `${protocol}//${window.location.host}/ws`;

            this.wsManager = new WebSocketManager(
                wsUrl,
                (message) => this.handleWebSocketMessage(message),
                (error) => this.handleWebSocketError(error),
                (connected) => {
                    this.wsConnected = connected;
                }
            );

            this.wsManager.connect();
        },

        handleWebSocketMessage(message) {
            console.log('WebSocket message received:', message);

            if (message.type === 'alert' && message.payload) {
                this.addAlert(message.payload);

                // Show toast for critical and high severity
                if (message.payload.severity === 'critical' || message.payload.severity === 'high') {
                    this.showToast(message.payload.message, message.payload.severity);
                }
            }
        },

        handleWebSocketError(error) {
            console.error('WebSocket error:', error);
        },

        // Alert Management
        addAlert(alert) {
            // Check if alert already exists (by ID or fingerprint)
            const existingIndex = this.alerts.findIndex(a =>
                a.id === alert.id || (a.fingerprint && a.fingerprint === alert.fingerprint)
            );

            if (existingIndex !== -1) {
                // Update existing alert - check if status changed
                const oldAlert = this.alerts[existingIndex];
                const statusChanged = oldAlert.status !== alert.status;

                this.alerts[existingIndex] = alert;
                console.log(`[UPDATE] Alert ID=${alert.id}, fingerprint=${alert.fingerprint}`);

                // Update counts if status changed
                if (statusChanged) {
                    if (oldAlert.status === 'firing' && alert.status !== 'firing') {
                        // Alert resolved: decrement active count
                        this.activeAlertsCount = Math.max(0, this.activeAlertsCount - 1);
                        // Decrement severity count
                        if (this.severityCounts[oldAlert.severity] !== undefined) {
                            this.severityCounts[oldAlert.severity] = Math.max(0, this.severityCounts[oldAlert.severity] - 1);
                        }
                    } else if (oldAlert.status !== 'firing' && alert.status === 'firing') {
                        // Alert fired: increment active count
                        this.activeAlertsCount++;
                        // Increment severity count
                        if (this.severityCounts[alert.severity] !== undefined) {
                            this.severityCounts[alert.severity]++;
                        }
                    }
                    this.updateSystemHealth();
                }
            } else {
                // Add new alert at the beginning
                this.alerts.unshift(alert);
                console.log(`[NEW ALERT] ID=${alert.id}, status=${alert.status}`);

                // Increment counts for new alerts
                this.totalAlerts++;
                if (alert.status === 'firing') {
                    this.activeAlertsCount++;
                    // Increment severity count
                    if (this.severityCounts[alert.severity] !== undefined) {
                        this.severityCounts[alert.severity]++;
                    }
                    this.updateSystemHealth();
                }

                // Increment unread count for new firing alerts if not already read
                if (alert.status === 'firing' && !this.readAlertIds.has(alert.id)) {
                    if (this.unreadCount < this.maxUnreadCount) {
                        this.unreadCount++;
                    }
                }

                // Keep only last 100 alerts in memory
                if (this.alerts.length > 100) {
                    this.alerts = this.alerts.slice(0, 100);
                }
            }

            this.lastUpdated = new Date().toISOString();
            // Counts are updated optimistically above
            // Full refresh from API happens every 30 seconds to ensure accuracy
        },

        updateSystemHealth() {
            const criticalCount = this.severityCounts.critical;
            const highCount = this.severityCounts.high;

            if (criticalCount > 0) {
                this.systemHealthStatus = 'critical';
            } else if (highCount > 3) {
                this.systemHealthStatus = 'degraded';
            } else if (highCount > 0) {
                this.systemHealthStatus = 'warning';
            } else {
                this.systemHealthStatus = 'healthy';
            }
        },

        // Filtering
        filteredAlerts() {
            return this.alerts.filter(alert => {
                // Filter by severity
                if (this.filterSeverity !== 'all' && alert.severity !== this.filterSeverity) {
                    return false;
                }

                // Filter by source
                if (this.filterSource !== 'all' && alert.source !== this.filterSource) {
                    return false;
                }

                // Filter by status
                if (this.filterStatus !== 'all' && alert.status !== this.filterStatus) {
                    return false;
                }

                return true;
            });
        },

        resetFilters() {
            this.filterSeverity = 'all';
            this.filterSource = 'all';
            this.filterStatus = 'all';
        },

        filterBySeverity(severity) {
            // Toggle filter: if already selected, clear it; otherwise set it
            if (this.filterSeverity === severity) {
                this.filterSeverity = 'all';
            } else {
                this.filterSeverity = severity;
            }
        },

        // Alert Detail Modal
        showAlertDetail(alert) {
            this.selectedAlert = alert;
            this.showModal = true;
        },

        // Auto-refresh
        toggleAutoRefresh() {
            this.autoRefresh = !this.autoRefresh;

            if (this.autoRefresh) {
                this.startAutoRefresh();
            } else {
                this.stopAutoRefresh();
            }
        },

        startAutoRefresh() {
            this.autoRefreshInterval = setInterval(() => {
                this.fetchAlerts();
            }, 30000); // 30 seconds
        },

        stopAutoRefresh() {
            if (this.autoRefreshInterval) {
                clearInterval(this.autoRefreshInterval);
                this.autoRefreshInterval = null;
            }
        },

        // Toast Notifications
        showToast(message, severity) {
            // Remove oldest toast if we're at the limit
            if (this.toasts.length >= this.maxToasts) {
                this.toasts.shift(); // Remove the oldest toast
            }

            const toast = {
                id: generateId(),
                message,
                severity
            };

            this.toasts.push(toast);

            // Auto-remove after 5 seconds
            setTimeout(() => {
                this.removeToast(toast.id);
            }, 5000);
        },

        removeToast(id) {
            this.toasts = this.toasts.filter(t => t.id !== id);
        },

        // Utility Methods
        formatRelativeTime(timestamp) {
            return formatRelativeTime(timestamp);
        },

        // Notification Bell Methods
        recentAlerts() {
            return this.alerts
                .filter(a => a.status === 'firing')
                .sort((a, b) => new Date(b.last_occurrence_at) - new Date(a.last_occurrence_at))
                .slice(0, 10);
        },

        isAlertUnread(alert) {
            return !this.readAlertIds.has(alert.id);
        },

        markAlertAsRead(alert) {
            if (!this.readAlertIds.has(alert.id)) {
                this.readAlertIds.add(alert.id);
                if (this.unreadCount > 0) {
                    this.unreadCount--;
                }
                this.saveReadAlertsToStorage();
                console.log(`Marked alert ${alert.id} as read`);
            }
        },

        markAllRead() {
            // Mark all firing alerts as read
            const firingAlerts = this.alerts.filter(a => a.status === 'firing');
            firingAlerts.forEach(alert => {
                this.readAlertIds.add(alert.id);
            });
            this.unreadCount = 0;
            this.showNotifications = false;
            this.saveReadAlertsToStorage();
            console.log(`Marked ${firingAlerts.length} alerts as read`);
        },

        getSeverityColor(severity) {
            const colors = {
                critical: 'text-red-600 dark:text-red-400',
                high: 'text-orange-600 dark:text-orange-400',
                medium: 'text-yellow-600 dark:text-yellow-400',
                low: 'text-blue-600 dark:text-blue-400'
            };
            return colors[severity] || 'text-gray-600 dark:text-gray-400';
        }
    };
}
