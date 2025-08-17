// Server-Sent Events client for real-time updates

class SSEClient {
    constructor() {
        this.eventSource = null;
        this.reconnectDelay = 1000;
        this.maxReconnectDelay = 30000;
        this.currentReconnectDelay = this.reconnectDelay;
        this.listeners = new Map();
        this.connected = false;
    }

    connect() {
        if (this.eventSource) {
            this.disconnect();
        }

        try {
            const token = window.app?.getToken();
            if (!token) {
                console.warn('No bearer token available for SSE connection');
                return;
            }

            // Note: EventSource doesn't support custom headers, so we'll include the token in the URL
            // In production, consider using WebSocket for authenticated real-time connections
            this.eventSource = new EventSource(`/api/events`);

            this.eventSource.onopen = () => {
                console.log('SSE connection opened');
                this.connected = true;
                this.currentReconnectDelay = this.reconnectDelay;
                this.emit('connected');
            };

            this.eventSource.onmessage = (event) => {
                try {
                    const data = JSON.parse(event.data);
                    this.handleMessage(data);
                } catch (error) {
                    console.error('Failed to parse SSE message:', error);
                }
            };

            this.eventSource.onerror = (error) => {
                console.error('SSE connection error:', error);
                this.connected = false;
                this.emit('error', error);
                
                if (this.eventSource.readyState === EventSource.CLOSED) {
                    this.scheduleReconnect();
                }
            };

        } catch (error) {
            console.error('Failed to establish SSE connection:', error);
            this.scheduleReconnect();
        }
    }

    disconnect() {
        if (this.eventSource) {
            this.eventSource.close();
            this.eventSource = null;
        }
        this.connected = false;
        this.emit('disconnected');
    }

    scheduleReconnect() {
        setTimeout(() => {
            console.log(`Attempting to reconnect SSE in ${this.currentReconnectDelay}ms`);
            this.connect();
            
            // Exponential backoff with jitter
            this.currentReconnectDelay = Math.min(
                this.currentReconnectDelay * 2 + Math.random() * 1000,
                this.maxReconnectDelay
            );
        }, this.currentReconnectDelay);
    }

    handleMessage(data) {
        console.log('SSE message received:', data);
        
        switch (data.type) {
            case 'connected':
                // Server confirmed connection
                break;
                
            case 'heartbeat':
                // Keep-alive message
                this.emit('heartbeat', data);
                break;
                
            case 'email_received':
                this.emit('email_received', data);
                this.showEmailNotification(data);
                break;
                
            case 'domain_health_updated':
                this.emit('domain_health_updated', data);
                this.updateDomainHealthUI(data);
                break;
                
            case 'system_health_updated':
                this.emit('system_health_updated', data);
                this.updateSystemHealthUI(data);
                break;
                
            case 'routing_rule_changed':
                this.emit('routing_rule_changed', data);
                break;
                
            default:
                console.log('Unknown SSE message type:', data.type);
                this.emit('message', data);
        }
    }

    showEmailNotification(data) {
        if (window.app && data.email) {
            const message = `New email from ${data.email.from}: ${data.email.subject}`;
            window.app.showNotification(message, 'info');
        }
    }

    updateDomainHealthUI(data) {
        // Update domain health displays if currently visible
        if (window.router?.currentRoute?.path.includes('/domains/') && 
            window.router.currentRoute.path.includes('/health')) {
            
            const healthDashboard = document.getElementById('health-dashboard');
            if (healthDashboard && window.HealthDashboard) {
                // Refresh the health dashboard with new data
                const dashboard = new HealthDashboard(healthDashboard);
                dashboard.render(data.health);
            }
        }
        
        // Update any domain health indicators in lists
        const healthIndicators = document.querySelectorAll(`[data-domain-health="${data.domain}"]`);
        healthIndicators.forEach(indicator => {
            indicator.className = `status-${data.health.dns.status}`;
            indicator.textContent = data.health.overall_score;
        });
    }

    updateSystemHealthUI(data) {
        // Update system health indicator in sidebar
        if (window.app) {
            window.app.systemHealth = data.health;
            window.app.updateSystemStatus();
        }
    }

    // Event listener management
    on(event, callback) {
        if (!this.listeners.has(event)) {
            this.listeners.set(event, []);
        }
        this.listeners.get(event).push(callback);
    }

    off(event, callback) {
        if (this.listeners.has(event)) {
            const callbacks = this.listeners.get(event);
            const index = callbacks.indexOf(callback);
            if (index > -1) {
                callbacks.splice(index, 1);
            }
        }
    }

    emit(event, data = null) {
        if (this.listeners.has(event)) {
            this.listeners.get(event).forEach(callback => {
                try {
                    callback(data);
                } catch (error) {
                    console.error(`Error in SSE event listener for ${event}:`, error);
                }
            });
        }
    }

    // Utility methods
    isConnected() {
        return this.connected && this.eventSource && this.eventSource.readyState === EventSource.OPEN;
    }

    getConnectionState() {
        if (!this.eventSource) return 'disconnected';
        
        switch (this.eventSource.readyState) {
            case EventSource.CONNECTING:
                return 'connecting';
            case EventSource.OPEN:
                return 'connected';
            case EventSource.CLOSED:
                return 'disconnected';
            default:
                return 'unknown';
        }
    }
}