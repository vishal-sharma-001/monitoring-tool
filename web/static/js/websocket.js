// WebSocket Manager for real-time alert updates with persistent reconnection

class WebSocketManager {
    constructor(url, onMessage, onError, onStatusChange) {
        this.url = url;
        this.ws = null;
        this.reconnectAttempts = 0;
        this.baseReconnectDelay = 1000; // Start with 1 second
        this.maxReconnectDelay = 30000; // Max 30 seconds between attempts
        this.onMessage = onMessage;
        this.onError = onError;
        this.onStatusChange = onStatusChange;
        this.reconnectTimer = null;
        this.heartbeatInterval = null;
        this.intentionallyClosed = false;
        this.missedHeartbeats = 0;
        this.maxMissedHeartbeats = 3;

        // Automatically reconnect when page becomes visible
        this.setupVisibilityHandling();
    }

    setupVisibilityHandling() {
        document.addEventListener('visibilitychange', () => {
            if (!document.hidden && (!this.ws || this.ws.readyState !== WebSocket.OPEN)) {
                console.log('Page visible, reconnecting WebSocket...');
                this.connect();
            }
        });

        // Also reconnect when coming back online
        window.addEventListener('online', () => {
            console.log('Network online, reconnecting WebSocket...');
            this.connect();
        });

        window.addEventListener('offline', () => {
            console.log('Network offline, WebSocket connection will be lost');
        });
    }

    connect() {
        // Clear any existing reconnect timer
        if (this.reconnectTimer) {
            clearTimeout(this.reconnectTimer);
            this.reconnectTimer = null;
        }

        // Close existing connection if any
        if (this.ws) {
            this.ws.onclose = null; // Prevent triggering reconnect
            this.ws.close();
            this.ws = null;
        }

        this.intentionallyClosed = false;

        try {
            console.log(`Connecting to WebSocket at ${this.url}...`);
            this.ws = new WebSocket(this.url);

            this.ws.onopen = () => {
                console.log('WebSocket connected successfully');
                this.reconnectAttempts = 0;
                this.missedHeartbeats = 0;
                if (this.onStatusChange) {
                    this.onStatusChange(true);
                }
                // Start heartbeat to keep connection alive
                this.startHeartbeat();
            };

            this.ws.onmessage = (event) => {
                try {
                    const data = JSON.parse(event.data);

                    // Reset heartbeat counter on any message
                    this.missedHeartbeats = 0;

                    // Handle pong messages
                    if (data.type === 'pong') {
                        return;
                    }

                    if (this.onMessage) {
                        this.onMessage(data);
                    }
                } catch (error) {
                    console.error('Failed to parse WebSocket message:', error);
                }
            };

            this.ws.onerror = (error) => {
                console.error('WebSocket error:', error);
                if (this.onError) {
                    this.onError(error);
                }
            };

            this.ws.onclose = (event) => {
                console.log(`WebSocket closed: code=${event.code}, reason=${event.reason || 'none'}, clean=${event.wasClean}`);
                this.stopHeartbeat();

                if (this.onStatusChange) {
                    this.onStatusChange(false);
                }

                // Reconnect unless intentionally closed
                if (!this.intentionallyClosed) {
                    this.reconnect();
                }
            };
        } catch (error) {
            console.error('Failed to create WebSocket:', error);
            if (this.onError) {
                this.onError(error);
            }
            this.reconnect();
        }
    }

    startHeartbeat() {
        this.stopHeartbeat();

        // Send ping every 30 seconds
        this.heartbeatInterval = setInterval(() => {
            if (this.ws && this.ws.readyState === WebSocket.OPEN) {
                this.missedHeartbeats++;

                if (this.missedHeartbeats >= this.maxMissedHeartbeats) {
                    console.warn('Too many missed heartbeats, reconnecting...');
                    this.ws.close();
                    return;
                }

                // Send ping
                this.send({ type: 'ping' });
            }
        }, 30000);
    }

    stopHeartbeat() {
        if (this.heartbeatInterval) {
            clearInterval(this.heartbeatInterval);
            this.heartbeatInterval = null;
        }
        this.missedHeartbeats = 0;
    }

    reconnect() {
        if (this.intentionallyClosed) {
            return;
        }

        // Clear any existing timer
        if (this.reconnectTimer) {
            clearTimeout(this.reconnectTimer);
        }

        this.reconnectAttempts++;

        // Exponential backoff with jitter
        const exponentialDelay = Math.min(
            this.baseReconnectDelay * Math.pow(2, this.reconnectAttempts - 1),
            this.maxReconnectDelay
        );

        // Add random jitter (0-1000ms) to prevent thundering herd
        const jitter = Math.random() * 1000;
        const delay = exponentialDelay + jitter;

        console.log(`Reconnecting in ${Math.round(delay / 1000)}s (attempt ${this.reconnectAttempts})...`);

        this.reconnectTimer = setTimeout(() => {
            this.connect();
        }, delay);
    }

    send(message) {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify(message));
        } else {
            console.warn('WebSocket is not connected, message not sent:', message);
        }
    }

    disconnect() {
        console.log('Disconnecting WebSocket intentionally');
        this.intentionallyClosed = true;
        this.stopHeartbeat();

        if (this.reconnectTimer) {
            clearTimeout(this.reconnectTimer);
            this.reconnectTimer = null;
        }

        if (this.ws) {
            this.ws.close();
            this.ws = null;
        }
    }

    // Get current connection state
    isConnected() {
        return this.ws && this.ws.readyState === WebSocket.OPEN;
    }
}
