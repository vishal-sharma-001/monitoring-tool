// API Client for interacting with the monitoring tool backend

class APIClient {
    constructor(baseURL = '') {
        this.baseURL = baseURL;
    }

    async getHealth() {
        return this.request('/health');
    }

    async getRecentAlerts(limit = 50) {
        return this.request(`/api/alerts/recent?limit=${limit}`);
    }

    async getAlertCount() {
        return this.request('/api/alerts/count');
    }

    async getActiveAlertCount() {
        return this.request('/api/alerts/active/count');
    }

    async getSeverityCounts() {
        return this.request('/api/alerts/severity/counts');
    }

    async request(endpoint, options = {}) {
        try {
            const response = await fetch(this.baseURL + endpoint, {
                headers: {
                    'Content-Type': 'application/json',
                    ...options.headers
                },
                ...options
            });

            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }

            return await response.json();
        } catch (error) {
            console.error('API request failed:', error);
            throw error;
        }
    }
}

// Create global API client instance
const apiClient = new APIClient();
